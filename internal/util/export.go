package util

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type ExportFormat string

const (
	ExportCSV       ExportFormat = "CSV"
	ExportJSON      ExportFormat = "JSON"
	ExportSQLInsert ExportFormat = "SQL INSERT"
)

// ExportRows writes rows to w in the requested format.
// columns controls field order; schema and table are used for SQL INSERT only.
// rows is []map[string]any (same underlying type as database.Row).
func ExportRows(w io.Writer, format ExportFormat, columns []string, rows []map[string]any, schema, table string) error {
	switch format {
	case ExportCSV:
		return exportCSV(w, columns, rows)
	case ExportJSON:
		return exportJSON(w, columns, rows)
	case ExportSQLInsert:
		return exportSQLInsert(w, columns, rows, schema, table)
	default:
		return fmt.Errorf("unknown export format: %s", format)
	}
}

func exportCSV(w io.Writer, columns []string, rows []map[string]any) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(columns); err != nil {
		return err
	}
	record := make([]string, len(columns))
	for _, row := range rows {
		for i, col := range columns {
			v := row[col]
			if v == nil {
				record[i] = ""
			} else {
				record[i] = stringifyValue(v)
			}
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func exportJSON(w io.Writer, columns []string, rows []map[string]any) error {
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		obj := make(map[string]any, len(columns))
		for _, col := range columns {
			obj[col] = row[col]
		}
		out[i] = obj
	}
	enc, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(enc)
	return err
}

func exportSQLInsert(w io.Writer, columns []string, rows []map[string]any, schema, table string) error {
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = fmt.Sprintf(`"%s"`, col)
	}
	colList := strings.Join(quotedCols, ", ")

	var target string
	if schema != "" {
		target = fmt.Sprintf(`"%s"."%s"`, schema, table)
	} else {
		target = fmt.Sprintf(`"%s"`, table)
	}

	for _, row := range rows {
		vals := make([]string, len(columns))
		for i, col := range columns {
			v := row[col]
			vals[i] = sqlLiteral(v)
		}
		_, err := fmt.Fprintf(w, "INSERT INTO %s (%s) VALUES (%s);\n",
			target, colList, strings.Join(vals, ", "))
		if err != nil {
			return err
		}
	}
	return nil
}

// stringifyValue converts any value to its string representation.
// nil returns empty string (for CSV); use sqlLiteral for SQL NULL handling.
func stringifyValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// sqlLiteral formats a value as a SQL literal: NULL, unquoted number/bool, or single-quoted string.
func sqlLiteral(v any) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return fmt.Sprintf("%v", val)
	case []byte:
		return "'" + strings.ReplaceAll(string(val), "'", "''") + "'"
	default:
		s := stringifyValue(v)
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
}
