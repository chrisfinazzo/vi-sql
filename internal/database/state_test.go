package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeepCopyRow_Independence(t *testing.T) {
	original := Row{"a": "hello", "b": int64(42), "c": nil}
	copied := deepCopyRow(original)

	//Mutation of copy must not affect original
	copied["a"] = "world"
	assert.Equal(t, "hello", original["a"], "original must not be mutated through copy")
}

func TestDeepCopyRow_Nil(t *testing.T) {
	assert.Nil(t, deepCopyRow(nil))
}

func TestStringifyValue_Nil(t *testing.T) {
	assert.Equal(t, "NULL", StringifyValue(nil))
}

func TestStringifyValue_String(t *testing.T) {
	assert.Equal(t, "hello", StringifyValue("hello"))
}

func TestStringifyValue_Int(t *testing.T) {
	assert.Equal(t, "42", StringifyValue(int64(42)))
}

func TestStringifyValue_Bool(t *testing.T) {
	assert.Equal(t, "true", StringifyValue(true))
	assert.Equal(t, "false", StringifyValue(false))
}

func TestStringifyValue_Bytes(t *testing.T) {
	assert.Equal(t, "hello", StringifyValue([]byte("hello")))
}

func TestStringifyValue_UUID(t *testing.T) {
	uuid := [16]byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	result := StringifyValue(uuid)
	assert.Equal(t, "01234567-89ab-cdef-0123-456789abcdef", result)
}

func TestStringifyValue_Map(t *testing.T) {
	m := map[string]interface{}{"key": "value"}
	result := StringifyValue(m)
	assert.Contains(t, result, "key")
	assert.Contains(t, result, "value")
}

func TestStringifyValue_Slice(t *testing.T) {
	s := []interface{}{1, "two", true}
	result := StringifyValue(s)
	assert.Contains(t, result, "two")
}

func TestStringifyValue_Time_PreservesOriginal(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	require.NoError(t, err)
	ts := time.Date(2026, 3, 6, 15, 16, 45, 802794000, loc)

	result := StringifyValue(ts)
	// Must not be empty and must contain date components
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "2026")
	assert.Contains(t, result, "15")
}

func TestTableState_SetAndGetPrimaryKey(t *testing.T) {
	ts := NewTableState("public", "users")
	assert.Empty(t, ts.GetPrimaryKey())

	ts.SetPrimaryKey([]string{"id"})
	assert.Equal(t, []string{"id"}, ts.GetPrimaryKey())
}

func TestTableState_GetAllRows_ReturnsDeepCopies(t *testing.T) {
	ts := NewTableState("public", "users")
	ts.PopulateRows([]Row{{"id": int64(1), "name": "Alice"}})

	rows := ts.GetAllRows()
	require.Len(t, rows, 1)

	// Mutating the returned copy must not touch the internal state
	rows[0]["name"] = "Mutated"
	fresh := ts.GetAllRows()
	assert.Equal(t, "Alice", fresh[0]["name"], "internal state must not be mutated via returned rows")
}

func TestTableState_UpdateRow_ChangesCorrectRow(t *testing.T) {
	ts := NewTableState("public", "users")
	ts.SetPrimaryKey([]string{"id"})
	ts.PopulateRows([]Row{
		{"id": int64(1), "name": "Alice"},
		{"id": int64(2), "name": "Bob"},
	})

	pk := PrimaryKey{Columns: map[string]any{"id": int64(1)}}
	ts.UpdateRow(pk, Row{"id": int64(1), "name": "Alice Updated"})

	rows := ts.GetAllRows()
	require.Len(t, rows, 2)

	var alice, bob Row
	for _, r := range rows {
		if r["id"] == int64(1) {
			alice = r
		} else {
			bob = r
		}
	}
	assert.Equal(t, "Alice Updated", alice["name"], "updated row must have new value")
	assert.Equal(t, "Bob", bob["name"], "other rows must be unchanged")
}

func TestTableState_UpdateRow_PreservesTypes(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	require.NoError(t, err)
	ts := time.Date(2026, 3, 6, 15, 16, 45, 802794000, loc)

	state := NewTableState("public", "events")
	state.SetPrimaryKey([]string{"id"})
	state.PopulateRows([]Row{{"id": int64(1), "created_at": ts}})

	pk := PrimaryKey{Columns: map[string]any{"id": int64(1)}}
	updated := Row{"id": int64(1), "created_at": ts}
	state.UpdateRow(pk, updated)

	rows := state.GetAllRows()
	v, ok := rows[0]["created_at"].(time.Time)
	require.True(t, ok, "created_at must remain a time.Time after UpdateRow")
	assert.True(t, ts.Equal(v), "timestamp value must be unchanged")
	assert.Equal(t, loc.String(), v.Location().String(), "timezone must be preserved")
}

// ---------------------------------------------------------------------------
// TableState — DeleteRow
// ---------------------------------------------------------------------------

func TestTableState_DeleteRow(t *testing.T) {
	state := NewTableState("public", "users")
	state.SetPrimaryKey([]string{"id"})
	state.PopulateRows([]Row{
		{"id": int64(1), "name": "Alice"},
		{"id": int64(2), "name": "Bob"},
	})
	state.Count = 2

	pk := PrimaryKey{Columns: map[string]any{"id": int64(1)}}
	state.DeleteRow(pk)

	rows := state.GetAllRows()
	assert.Len(t, rows, 1)
	assert.Equal(t, int64(1), state.Count)
	assert.Equal(t, "Bob", rows[0]["name"])
}

func TestTableState_Pagination(t *testing.T) {
	state := NewTableState("public", "t")
	state.Limit = 10
	state.Count = 25

	assert.Equal(t, int64(1), state.GetCurrentPage())
	assert.Equal(t, int64(3), state.GetTotalPages())

	state.SetOffset(10)
	assert.Equal(t, int64(2), state.GetCurrentPage())

	// Negative offset is clamped to 0
	state.SetOffset(-5)
	assert.Equal(t, int64(0), state.Offset)
}

func TestStateMap_SetAndGet(t *testing.T) {
	sm := NewStateMap()
	state := NewTableState("public", "users")
	sm.Set(sm.Key("public", "users"), state)

	got, ok := sm.Get(sm.Key("public", "users"))
	assert.True(t, ok)
	assert.Same(t, state, got)
}

func TestStateMap_HiddenColumns(t *testing.T) {
	sm := NewStateMap()

	assert.Empty(t, sm.GetHiddenColumns("public", "users"))

	sm.AddHiddenColumn("public", "users", "secret")
	sm.AddHiddenColumn("public", "users", "internal")
	cols := sm.GetHiddenColumns("public", "users")
	assert.ElementsMatch(t, []string{"secret", "internal"}, cols)

	// Other tables are unaffected
	assert.Empty(t, sm.GetHiddenColumns("public", "orders"))

	sm.ResetHiddenColumns("public", "users")
	assert.Empty(t, sm.GetHiddenColumns("public", "users"))
}
