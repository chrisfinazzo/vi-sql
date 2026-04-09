package database

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseValueByType(value string, dataType string) (any, error) {
	if strings.EqualFold(value, "NULL") || strings.EqualFold(value, "null") {
		return nil, nil
	}

	dt := strings.ToLower(dataType)

	switch {
	case strings.Contains(dt, "int"):
		return strconv.ParseInt(value, 10, 64)
	case strings.Contains(dt, "numeric") || strings.Contains(dt, "decimal") ||
		strings.Contains(dt, "real") || strings.Contains(dt, "double") || dt == "float":
		return strconv.ParseFloat(value, 64)
	case strings.Contains(dt, "bool"):
		return strconv.ParseBool(value)
	case strings.Contains(dt, "timestamp") || strings.Contains(dt, "date"):
		return parseTimeValue(value)
	default:
		return value, nil
	}
}

func parseTimeValue(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time value: %s", value)
}

// TypeSymbol returns a short symbol representing the SQL data type.
// When betterSymbols is true, Nerd Font glyphs are used; otherwise safe
// single-width Unicode fallbacks are returned.
func TypeSymbol(dataType string, betterSymbols bool) string {
	dt := strings.ToLower(dataType)

	// Nerd Font BMP-range glyphs — safe for tview/tcell.
	switch {
	case strings.Contains(dt, "timestamp"):
		if betterSymbols {
			return "\uF017" // fa-clock-o
		}
		return "⧗"
	case strings.Contains(dt, "date"):
		if betterSymbols {
			return "\uF133" // fa-calendar-o
		}
		return "▦"
	case strings.Contains(dt, "time"):
		if betterSymbols {
			return "\uF017" // fa-clock-o
		}
		return "◷"
	case strings.Contains(dt, "int") || strings.Contains(dt, "serial") ||
		strings.Contains(dt, "numeric") || strings.Contains(dt, "decimal") ||
		strings.Contains(dt, "real") || strings.Contains(dt, "double") ||
		strings.Contains(dt, "float") || strings.Contains(dt, "money"):
		if betterSymbols {
			return "\uF4F7" // nf-oct-number
		}
		return "~#"
	case strings.Contains(dt, "bool"):
		if betterSymbols {
			return "\uF046" // fa-check-square-o
		}
		return "◉"
	case strings.Contains(dt, "json"):
		if betterSymbols {
			return "\uE60B" // fa-file-code-o
		}
		return "{}"
	case strings.Contains(dt, "uuid"):
		if betterSymbols {
			return "\uF2C1"
		}
		return "⊞"
	case strings.Contains(dt, "char") || strings.Contains(dt, "text") ||
		strings.Contains(dt, "varchar") || dt == "name":
		if betterSymbols {
			return "\uEB69" // fa-file-text-o
		}
		return "T"
	case strings.Contains(dt, "byte") || strings.Contains(dt, "binary") ||
		strings.Contains(dt, "blob"):
		if betterSymbols {
			return "\uEAE8" // fa-archive
		}
		return "⬡"
	default:
		// Custom/enum/domain types
		if betterSymbols {
			return "\uF1B2" // fa-cube
		}
		return "~"
	}
}

// ExtractSelectClauses parses a simple one-line SELECT and returns the WHERE
// and ORDER BY clause bodies (without the keywords). Handles LIMIT as a
// terminator so trailing pagination hints are not included.
func ExtractSelectClauses(sql string) (where, orderBy string) {
	upper := strings.ToUpper(sql)

	find := func(kw string) int {
		// Require a space before and after the keyword to avoid matching
		// inside identifiers (e.g. "password WHERE ..." vs "ELSEWHERE").
		idx := strings.Index(upper, " "+kw+" ")
		if idx >= 0 {
			return idx + 1 // point at start of keyword, not the leading space
		}
		return -1
	}

	wherePos := find("WHERE")
	orderByPos := find("ORDER BY")
	limitPos := find("LIMIT")

	end := len(sql)
	if limitPos >= 0 && limitPos < end {
		end = limitPos
	}

	if orderByPos >= 0 && orderByPos < end {
		orderBy = strings.TrimSpace(sql[orderByPos+8 : end]) // len("ORDER BY") == 8
		end = orderByPos
	}

	if wherePos >= 0 && wherePos < end {
		where = strings.TrimSpace(sql[wherePos+5 : end]) // len("WHERE") == 5
	}

	return where, orderBy
}

// RebuildSelectSQL replaces the WHERE and ORDER BY clauses in a simple
// one-line SELECT, preserving everything before the first modifier clause.
// Pass empty strings to omit a clause.
func RebuildSelectSQL(sql, newWhere, newOrderBy string) string {
	upper := strings.ToUpper(sql)

	find := func(kw string) int {
		idx := strings.Index(upper, " "+kw+" ")
		if idx >= 0 {
			return idx
		}
		return -1
	}

	cutPos := len(sql)
	for _, pos := range []int{find("WHERE"), find("ORDER BY"), find("LIMIT")} {
		if pos >= 0 && pos < cutPos {
			cutPos = pos
		}
	}

	base := strings.TrimSpace(sql[:cutPos])
	if newWhere != "" {
		base += " WHERE " + newWhere
	}
	if newOrderBy != "" {
		base += " ORDER BY " + newOrderBy
	}
	return base
}

// SanitizeWhereClause performs basic sanity checks on user-provided WHERE input.
// It rejects obvious DDL/DML injection attempts in a filter context.
func SanitizeWhereClause(where string) error {
	if where == "" {
		return nil
	}

	upper := strings.ToUpper(strings.TrimSpace(where))
	forbidden := []string{
		"DROP ", "DELETE ", "INSERT ", "UPDATE ", "ALTER ",
		"CREATE ", "TRUNCATE ", "GRANT ", "REVOKE ",
		"EXEC ", "EXECUTE ",
	}
	for _, f := range forbidden {
		if strings.Contains(upper, f) {
			return fmt.Errorf("WHERE clause must not contain %s statements", strings.TrimSpace(f))
		}
	}
	return nil
}
