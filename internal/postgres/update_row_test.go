package postgres

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// changedColumns replicates the diff logic from UpdateRow so we can test it
// without a real database connection.
func changedColumns(original, updated map[string]any) []string {
	var changed []string
	for col, newVal := range updated {
		if col == "_pk" {
			continue
		}
		oldVal, exists := original[col]
		if !exists || !reflect.DeepEqual(oldVal, newVal) {
			changed = append(changed, col)
		}
	}
	return changed
}

// ---------------------------------------------------------------------------
// Comparable scalar types
// ---------------------------------------------------------------------------

func TestChangedColumns_NoChange_String(t *testing.T) {
	orig := map[string]any{"name": "Alice"}
	updated := map[string]any{"name": "Alice"}
	assert.Empty(t, changedColumns(orig, updated))
}

func TestChangedColumns_Changed_String(t *testing.T) {
	orig := map[string]any{"name": "Alice"}
	updated := map[string]any{"name": "Bob"}
	assert.ElementsMatch(t, []string{"name"}, changedColumns(orig, updated))
}

func TestChangedColumns_NoChange_Int(t *testing.T) {
	orig := map[string]any{"age": int64(30)}
	updated := map[string]any{"age": int64(30)}
	assert.Empty(t, changedColumns(orig, updated))
}

func TestChangedColumns_Changed_Int(t *testing.T) {
	orig := map[string]any{"age": int64(30)}
	updated := map[string]any{"age": int64(31)}
	assert.ElementsMatch(t, []string{"age"}, changedColumns(orig, updated))
}

func TestChangedColumns_NoChange_Bool(t *testing.T) {
	orig := map[string]any{"active": true}
	updated := map[string]any{"active": true}
	assert.Empty(t, changedColumns(orig, updated))
}

// ---------------------------------------------------------------------------
// Uncomparable types — these would panic with != but must not with DeepEqual
// ---------------------------------------------------------------------------

func TestChangedColumns_NoChange_ByteSlice(t *testing.T) {
	b := []byte{1, 2, 3}
	orig := map[string]any{"data": b}
	updated := map[string]any{"data": []byte{1, 2, 3}}
	// Must not panic and must detect no change
	assert.Empty(t, changedColumns(orig, updated))
}

func TestChangedColumns_Changed_ByteSlice(t *testing.T) {
	orig := map[string]any{"data": []byte{1, 2, 3}}
	updated := map[string]any{"data": []byte{4, 5, 6}}
	assert.ElementsMatch(t, []string{"data"}, changedColumns(orig, updated))
}

func TestChangedColumns_NoChange_JSONArray(t *testing.T) {
	arr := []interface{}{"a", "b", "c"}
	orig := map[string]any{"tags": arr}
	updated := map[string]any{"tags": []interface{}{"a", "b", "c"}}
	// Must not panic
	assert.Empty(t, changedColumns(orig, updated))
}

func TestChangedColumns_Changed_JSONArray(t *testing.T) {
	orig := map[string]any{"tags": []interface{}{"a", "b"}}
	updated := map[string]any{"tags": []interface{}{"a", "b", "c"}}
	assert.ElementsMatch(t, []string{"tags"}, changedColumns(orig, updated))
}

func TestChangedColumns_NoChange_JSONObject(t *testing.T) {
	orig := map[string]any{"meta": map[string]interface{}{"k": "v"}}
	updated := map[string]any{"meta": map[string]interface{}{"k": "v"}}
	assert.Empty(t, changedColumns(orig, updated))
}

func TestChangedColumns_Changed_JSONObject(t *testing.T) {
	orig := map[string]any{"meta": map[string]interface{}{"k": "v1"}}
	updated := map[string]any{"meta": map[string]interface{}{"k": "v2"}}
	assert.ElementsMatch(t, []string{"meta"}, changedColumns(orig, updated))
}

// ---------------------------------------------------------------------------
// time.Time — timezone must be preserved in equality check
// ---------------------------------------------------------------------------

func TestChangedColumns_NoChange_TimeUTC(t *testing.T) {
	ts := time.Date(2026, 3, 6, 15, 16, 45, 0, time.UTC)
	orig := map[string]any{"created_at": ts}
	updated := map[string]any{"created_at": ts}
	assert.Empty(t, changedColumns(orig, updated))
}

func TestChangedColumns_Changed_Time(t *testing.T) {
	ts1 := time.Date(2026, 3, 6, 15, 16, 45, 0, time.UTC)
	ts2 := time.Date(2026, 3, 6, 16, 16, 45, 0, time.UTC)
	orig := map[string]any{"created_at": ts1}
	updated := map[string]any{"created_at": ts2}
	assert.ElementsMatch(t, []string{"created_at"}, changedColumns(orig, updated))
}

func TestChangedColumns_TypeMismatch_StringVsTime(t *testing.T) {
	// When inline-edit passes a string for an originally typed value,
	// it must be detected as changed (the string will be coerced by the DB).
	ts := time.Date(2026, 3, 6, 15, 16, 45, 0, time.UTC)
	orig := map[string]any{"created_at": ts}
	updated := map[string]any{"created_at": ts.Format(time.RFC3339Nano)}
	assert.ElementsMatch(t, []string{"created_at"}, changedColumns(orig, updated),
		"string representation of a time.Time must be detected as changed vs the original typed value")
}

// ---------------------------------------------------------------------------
// _pk column is always skipped
// ---------------------------------------------------------------------------

func TestChangedColumns_SkipsPKColumn(t *testing.T) {
	orig := map[string]any{"_pk": "ignored", "name": "Alice"}
	updated := map[string]any{"_pk": "different", "name": "Alice"}
	assert.Empty(t, changedColumns(orig, updated))
}

// ---------------------------------------------------------------------------
// New column in updated (not present in original) is treated as changed
// ---------------------------------------------------------------------------

func TestChangedColumns_NewColumn(t *testing.T) {
	orig := map[string]any{"name": "Alice"}
	updated := map[string]any{"name": "Alice", "email": "alice@example.com"}
	assert.ElementsMatch(t, []string{"email"}, changedColumns(orig, updated))
}
