package database

import (
	"strings"
)

// TableRef holds a schema-qualified table reference parsed from a FROM/JOIN clause.
// Schema may be empty for unqualified table names; Alias may be empty if none is given.
type TableRef struct {
	Schema string
	Table  string
	Alias  string
}

// tableTerminators are keywords that end a table reference and cannot be an alias.
var tableTerminators = map[string]bool{
	"WHERE": true, "JOIN": true, "LEFT": true, "RIGHT": true, "INNER": true,
	"FULL": true, "CROSS": true, "NATURAL": true, "ON": true, "USING": true,
	"GROUP": true, "ORDER": true, "HAVING": true, "LIMIT": true, "OFFSET": true,
	"UNION": true, "INTERSECT": true, "EXCEPT": true, "SET": true,
	"RETURNING": true, "INTO": true, "VALUES": true, "SELECT": true, "FROM": true,
	"WITH": true, "FETCH": true, "FOR": true,
}

// skipWS returns the index of the first non-whitespace token at or after start.
func skipWS(tokens []Token, start int) int {
	for start < len(tokens) && tokens[start].Type == TokenWhitespace {
		start++
	}
	return start
}

// ExtractFromTableRefs returns all table references found in FROM/JOIN clauses of sql,
// including any alias declared with or without AS.
func ExtractFromTableRefs(sql string) []TableRef {
	tokens := Tokenize(sql)
	var refs []TableRef
	fromKeywords := map[string]bool{
		"FROM": true, "JOIN": true, "INTO": true,
	}
	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		if tok.Type == TokenKeyword && fromKeywords[strings.ToUpper(tok.Value)] {
			j := skipWS(tokens, i+1)
			if j >= len(tokens) {
				break
			}
			first := tokens[j]
			if first.Type != TokenIdentifier && first.Type != TokenQuotedIdentifier {
				i++
				continue
			}

			ref := TableRef{}
			// check for schema.table
			if j+2 < len(tokens) && tokens[j+1].Type == TokenDot {
				second := tokens[j+2]
				if second.Type == TokenIdentifier || second.Type == TokenQuotedIdentifier {
					ref.Schema = first.Value
					ref.Table = second.Value
					j = j + 3
				} else {
					ref.Table = first.Value
					j = j + 1
				}
			} else {
				ref.Table = first.Value
				j = j + 1
			}

			// Look for optional alias: [AS] identifier
			k := skipWS(tokens, j)
			if k < len(tokens) {
				next := tokens[k]
				if next.Type == TokenKeyword && strings.ToUpper(next.Value) == "AS" {
					k = skipWS(tokens, k+1)
					if k < len(tokens) && (tokens[k].Type == TokenIdentifier || tokens[k].Type == TokenQuotedIdentifier) {
						ref.Alias = tokens[k].Value
					}
				} else if next.Type == TokenIdentifier && !tableTerminators[strings.ToUpper(next.Value)] {
					ref.Alias = next.Value
				}
			}

			refs = append(refs, ref)
			i = j
			continue
		}
		i++
	}
	return refs
}

// ExtractCTENames returns the names of all CTEs declared in a WITH clause.
func ExtractCTENames(sql string) []string {
	tokens := Tokenize(sql)
	var names []string
	for i, tok := range tokens {
		if tok.Type != TokenKeyword || strings.ToUpper(tok.Value) != "WITH" {
			continue
		}
		j := skipWS(tokens, i+1)
		for j < len(tokens) {
			if tokens[j].Type != TokenIdentifier {
				break
			}
			cteName := tokens[j].Value
			j = skipWS(tokens, j+1)
			if j >= len(tokens) || strings.ToUpper(tokens[j].Value) != "AS" {
				break
			}
			names = append(names, cteName)
			j = skipWS(tokens, j+1)
			if j >= len(tokens) || tokens[j].Value != "(" {
				break
			}
			// skip the CTE body tracking parenthesis depth
			depth := 0
			for j < len(tokens) {
				if tokens[j].Type == TokenPunctuation {
					switch tokens[j].Value {
					case "(":
						depth++
					case ")":
						depth--
						if depth == 0 {
							j++
							goto afterBody
						}
					}
				}
				j++
			}
		afterBody:
			j = skipWS(tokens, j)
			// comma → another CTE follows
			if j < len(tokens) && tokens[j].Type == TokenPunctuation && tokens[j].Value == "," {
				j = skipWS(tokens, j+1)
				continue
			}
			break
		}
	}
	return names
}

func IsExplainQuery(sql string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "EXPLAIN")
}

// IsReturningDML checks for INSERT/UPDATE/DELETE with a RETURNING clause
func IsReturningDML(sql string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sql))
	hasDML := strings.HasPrefix(upper, "INSERT") ||
		strings.HasPrefix(upper, "UPDATE") ||
		strings.HasPrefix(upper, "DELETE")
	return hasDML && strings.Contains(upper, "RETURNING")
}

type DestructiveStatementInfo struct {
	Operation string
	Table     string
	HasWhere  bool
}

// HasDestructiveStatement returns info when sql is a DELETE / UPDATE / TRUNCATE /
// DROP statement, also reports whether a WHERE is present
func HasDestructiveStatement(sql string) *DestructiveStatementInfo {
	tokens := Tokenize(sql)

	sig := make([]Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Type != TokenWhitespace && t.Type != TokenComment {
			sig = append(sig, t)
		}
	}
	if len(sig) == 0 {
		return nil
	}

	first := strings.ToUpper(sig[0].Value)
	switch first {
	case "DELETE":
		info := &DestructiveStatementInfo{Operation: "DELETE"}
		idx := 1
		if idx < len(sig) && strings.ToUpper(sig[idx].Value) == "FROM" {
			idx++
		}
		info.Table = extractTableName(sig, idx)
		info.HasWhere = topLevelWhere(sig)
		return info

	case "UPDATE":
		info := &DestructiveStatementInfo{Operation: "UPDATE"}
		info.Table = extractTableName(sig, 1)
		info.HasWhere = topLevelWhere(sig)
		return info

	case "TRUNCATE":
		info := &DestructiveStatementInfo{Operation: "TRUNCATE"}
		idx := 1
		if idx < len(sig) && strings.ToUpper(sig[idx].Value) == "TABLE" {
			idx++
		}
		info.Table = extractTableName(sig, idx)
		return info

	case "DROP":
		info := &DestructiveStatementInfo{Operation: "DROP"}
		if len(sig) > 1 {
			switch strings.ToUpper(sig[1].Value) {
			case "TABLE", "VIEW", "INDEX", "SCHEMA", "DATABASE", "TRIGGER", "FUNCTION", "PROCEDURE", "SEQUENCE":
				idx := 2
				if idx+1 < len(sig) && strings.ToUpper(sig[idx].Value) == "IF" && strings.ToUpper(sig[idx+1].Value) == "EXISTS" {
					idx += 2
				}
				info.Table = extractTableName(sig, idx)
			}
		}
		return info
	}

	return nil
}

// extractTableName returns the table name, schema.table if possible
func extractTableName(sig []Token, idx int) string {
	if idx >= len(sig) {
		return ""
	}
	t := sig[idx]
	if t.Type != TokenIdentifier && t.Type != TokenQuotedIdentifier {
		return ""
	}
	name := t.Value
	if idx+2 < len(sig) && sig[idx+1].Type == TokenDot {
		name = name + "." + sig[idx+2].Value
	}
	return name
}

// topLevelWhere reports whether sig contains a WHERE keyword at the top level
func topLevelWhere(sig []Token) bool {
	depth := 0
	for _, t := range sig {
		switch t.Value {
		case "(":
			depth++
		case ")":
			if depth > 0 {
				depth--
			}
		}
		if depth == 0 && t.Type == TokenKeyword && strings.ToUpper(t.Value) == "WHERE" {
			return true
		}
	}
	return false
}

// AutocompleteEntry is a suggestion returned by BuildSQLAutocomplete.
// Main is the text to insert; Secondary is a hint shown alongside it.
type AutocompleteEntry struct {
	Main      string
	Secondary string
}

// BuildSQLAutocomplete returns autocomplete suggestions for a SQL text at the
// given cursor byte position, using the supplied schema, column, and variable lists.
// variables holds dynamically extracted identifiers such as CTE names and table aliases.
func BuildSQLAutocomplete(
	text string,
	cursorBytePos int,
	schemas []Schema,
	columns []string,
	variables []string,
) []AutocompleteEntry {
	tokens := Tokenize(text)
	ctx := DetectContext(tokens, cursorBytePos)
	partial := strings.ToLower(ctx.PartialWord)

	var entries []AutocompleteEntry

	switch ctx.Type {
	case CtxStatementStart:
		if partial == "" {
			return nil
		}
		for _, kw := range append(DMLKeywords, DDLVerbKeywords...) {
			if strings.HasPrefix(strings.ToLower(kw), partial) {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
		}

	case CtxAfterDDLVerb:
		for _, kw := range DDLObjectKeywords {
			if partial == "" || strings.HasPrefix(strings.ToLower(kw), partial) {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
		}

	case CtxAfterDDLObject:
		// Show schema-qualified tables — useful for DROP/ALTER/TRUNCATE to pick an
		// existing name, and for CREATE to prefix with a schema (e.g. public.new_table).
		for _, v := range variables {
			if partial == "" || strings.HasPrefix(strings.ToLower(v), partial) {
				entries = append(entries, AutocompleteEntry{Main: v, Secondary: "cte"})
			}
		}
		for _, schema := range schemas {
			for _, table := range schema.Tables {
				qualified := schema.Schema + "." + table
				if partial == "" || strings.HasPrefix(strings.ToLower(qualified), partial) ||
					strings.HasPrefix(strings.ToLower(table), partial) {
					entries = append(entries, AutocompleteEntry{Main: qualified})
				}
			}
		}

	case CtxAfterFrom, CtxAfterJoin, CtxAfterInto:
		// After a complete table reference (not after a comma), show clause keywords.
		if partial == "" && ctx.PrecedingTokenType == TokenIdentifier {
			for _, kw := range ClauseKeywords {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
			return entries
		}
		for _, v := range variables {
			if partial == "" || strings.HasPrefix(strings.ToLower(v), partial) {
				entries = append(entries, AutocompleteEntry{Main: v, Secondary: "cte"})
			}
		}
		for _, schema := range schemas {
			for _, table := range schema.Tables {
				qualified := schema.Schema + "." + table
				if partial == "" || strings.HasPrefix(strings.ToLower(qualified), partial) ||
					strings.HasPrefix(strings.ToLower(table), partial) {
					entries = append(entries, AutocompleteEntry{
						Main: qualified,
					})
				}
			}
		}
		for _, kw := range ClauseKeywords {
			if partial != "" && strings.HasPrefix(strings.ToLower(kw), partial) {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
		}

	case CtxAfterSelect, CtxAfterWhere, CtxAfterOn,
		CtxAfterOrderBy, CtxAfterGroupBy, CtxAfterSet:
		// Cursor is inside or just after a literal value — no suggestions.
		if partial == "" && (ctx.PrecedingTokenType == TokenNumber || ctx.PrecedingTokenType == TokenString) {
			return nil
		}
		// After a complete column name (not after a comma), show clause keywords.
		if partial == "" && ctx.PrecedingTokenType == TokenIdentifier {
			for _, kw := range ClauseKeywords {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
			return entries
		}
		for _, col := range columns {
			if partial == "" || strings.HasPrefix(strings.ToLower(col), partial) {
				entries = append(entries, AutocompleteEntry{Main: col})
			}
		}
		for _, kw := range SQLKeywords {
			if partial != "" && strings.HasPrefix(strings.ToLower(kw), partial) {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
		}
		for _, kw := range ClauseKeywords {
			if partial != "" && strings.HasPrefix(strings.ToLower(kw), partial) {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
		}

	case CtxAfterDot:
		// If qualifier is a schema name, show its tables instead of columns.
		for _, schema := range schemas {
			if strings.EqualFold(ctx.TableName, schema.Schema) {
				for _, table := range schema.Tables {
					if partial == "" || strings.HasPrefix(strings.ToLower(table), partial) {
						entries = append(entries, AutocompleteEntry{
							Main: table,
						})
					}
				}
				return entries
			}
		}
		for _, col := range columns {
			if partial == "" || strings.HasPrefix(strings.ToLower(col), partial) {
				entries = append(entries, AutocompleteEntry{Main: col})
			}
		}

	default:
		if partial == "" {
			return nil
		}
		for _, kw := range SQLKeywords {
			if strings.HasPrefix(strings.ToLower(kw), partial) {
				entries = append(entries, AutocompleteEntry{Main: kw})
			}
		}
	}

	return entries
}
