package postgres

import "testing"

func TestToSQLStandardTypeName(t *testing.T) {
	tests := map[string]string{
		"bool":    "boolean",
		"boolean": "boolean",
		"int4":    "int4",
	}
	for input, want := range tests {
		if got := toSQLStandardTypeName(input); got != want {
			t.Errorf("toSQLStandardTypeName(%q) = %q, want %q", input, got, want)
		}
	}
}
