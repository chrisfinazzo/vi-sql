package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var exportColumns = []string{"id", "name", "age"}
var exportRows = []map[string]any{
	{"id": 1, "name": "Alice", "age": 30},
	{"id": 2, "name": "Bob", "age": nil},
}

func TestExportCSVWithHeaders(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportCSV, exportColumns, exportRows, "", "", ExportOptions{IncludeHeaders: true})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	assert.Equal(t, "id,name,age", lines[0])
	assert.Equal(t, "1,Alice,30", lines[1])
	assert.Equal(t, "2,Bob,", lines[2])
}

func TestExportCSVWithoutHeaders(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportCSV, exportColumns, exportRows, "", "", ExportOptions{IncludeHeaders: false})
	require.NoError(t, err)

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	assert.Len(t, lines, 2)
	assert.Equal(t, "1,Alice,30", lines[0])
}

func TestExportJSONCompact(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportJSON, exportColumns, exportRows, "", "", ExportOptions{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"name":"Alice"`)
	assert.NotContains(t, output, "\n  ")
}

func TestExportJSONPretty(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportJSON, exportColumns, exportRows, "", "", ExportOptions{PrettyPrint: true})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "\n  ")
	assert.Contains(t, output, `"name": "Alice"`)
}

func TestExportJSONNullValue(t *testing.T) {
	var buf bytes.Buffer
	rows := []map[string]any{{"id": 1, "name": nil}}
	err := ExportRows(&buf, ExportJSON, []string{"id", "name"}, rows, "", "", ExportOptions{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name":null`)
}

func TestExportSQLInsertWithSchema(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportSQLInsert, exportColumns, exportRows, "public", "users", ExportOptions{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `INSERT INTO "public"."users"`)
	assert.Contains(t, output, `VALUES (1, 'Alice', 30)`)
	assert.Contains(t, output, `VALUES (2, 'Bob', NULL)`)
}

func TestExportSQLInsertWithoutSchema(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportSQLInsert, exportColumns, exportRows, "", "users", ExportOptions{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `INSERT INTO "users"`)
	assert.NotContains(t, output, `"".`)
}

func TestExportSQLInsertStringEscaping(t *testing.T) {
	var buf bytes.Buffer
	rows := []map[string]any{{"name": "O'Brien"}}
	err := ExportRows(&buf, ExportSQLInsert, []string{"name"}, rows, "", "t", ExportOptions{})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `'O''Brien'`)
}

func TestExportMarkdownWithHeaders(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportMarkdown, exportColumns, exportRows, "", "", ExportOptions{IncludeHeaders: true})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "| id")
	assert.Contains(t, output, "| name")
	assert.Contains(t, output, "|")
	assert.Contains(t, output, "---")
	assert.Contains(t, output, "Alice")
}

func TestExportMarkdownWithoutHeaders(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportMarkdown, exportColumns, exportRows, "", "", ExportOptions{IncludeHeaders: false})
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "---")
	assert.Contains(t, output, "Alice")
}

func TestExportTextWithHeaders(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportText, exportColumns, exportRows, "", "", ExportOptions{IncludeHeaders: true})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "---")
	assert.Contains(t, output, "Alice")
}

func TestExportUnknownFormatReturnsError(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportFormat("XML"), exportColumns, exportRows, "", "", ExportOptions{})
	assert.Error(t, err)
}

func TestExportEmptyRows(t *testing.T) {
	var buf bytes.Buffer
	err := ExportRows(&buf, ExportCSV, exportColumns, []map[string]any{}, "", "", ExportOptions{IncludeHeaders: true})
	require.NoError(t, err)
	assert.Equal(t, "id,name,age\n", buf.String())
}
