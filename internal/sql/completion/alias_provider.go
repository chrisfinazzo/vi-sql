package completion

import (
	"strings"

	sql "github.com/kopecmaciej/vi-sql/internal/sql"
)

// AliasProvider handles dot-qualified completions (table.column or schema.table).
// For schema qualifiers it returns the schema's tables; for table/alias qualifiers
// it returns the columns of the resolved table.
type AliasProvider struct{}

func (AliasProvider) Applicable(ctx sql.ContextType, _ *QueryScope) bool {
	return ctx == sql.CtxAfterDot
}

func (AliasProvider) Suggest(ctx sql.CompletionContext, scope *QueryScope, partial string, cfg Context) []Symbol {
	qualifier := ctx.TableName
	lowerQualifier := strings.ToLower(qualifier)

	// If qualifier matches a schema name, return its tables (O(1) map lookup).
	if tables, ok := cfg.Index.bySchema[lowerQualifier]; ok {
		var out []Symbol
		for _, it := range tables {
			if partial == "" || strings.HasPrefix(it.lowerTable, partial) {
				out = append(out, Symbol{
					Kind:      KindTable,
					Name:      it.table,
					Qualifier: it.schema,
					Priority:  50,
				})
			}
		}
		return out
	}

	// Resolve alias to real table name.
	realTable := qualifier
	if scope != nil {
		realTable = scope.ResolveTable(qualifier)
	}

	// O(1) lookup: find the schema owning realTable.
	schema := cfg.Index.tableToSchema[strings.ToLower(realTable)]

	cols := fetchColumns(schema, realTable, cfg)
	var out []Symbol
	for _, col := range cols {
		if partial == "" || strings.HasPrefix(strings.ToLower(col.Name), partial) {
			out = append(out, Symbol{
				Kind:      KindColumn,
				Name:      col.Name,
				TypeHint:  col.TypeHint,
				IsPK:      col.IsPK,
				Qualifier: realTable,
				Priority:  60,
			})
		}
	}
	return out
}
