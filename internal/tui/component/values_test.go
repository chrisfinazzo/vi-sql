package component

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// sqlLiteral — used to generate the UPDATE template opened in the editor
// ---------------------------------------------------------------------------

func TestSQLLiteral_Nil(t *testing.T) {
	assert.Equal(t, "NULL", sqlLiteral(nil))
}

func TestSQLLiteral_String(t *testing.T) {
	assert.Equal(t, "'hello'", sqlLiteral("hello"))
}

func TestSQLLiteral_StringWithSingleQuote(t *testing.T) {
	assert.Equal(t, "'it''s'", sqlLiteral("it's"), "single quotes must be escaped")
}

func TestSQLLiteral_Integer(t *testing.T) {
	assert.Equal(t, "'42'", sqlLiteral(int64(42)))
}

func TestSQLLiteral_Bool(t *testing.T) {
	assert.Equal(t, "'true'", sqlLiteral(true))
}

func TestSQLLiteral_Time_UsesRFC3339Nano(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	require.NoError(t, err)
	ts := time.Date(2026, 3, 6, 15, 16, 45, 802794000, loc)

	result := sqlLiteral(ts)

	// Must produce a quoted RFC3339Nano string, NOT Go's default format
	expected := "'" + ts.Format(time.RFC3339Nano) + "'"
	assert.Equal(t, expected, result)

	// Must NOT contain the "CET" abbreviation that PostgreSQL rejects
	assert.NotContains(t, result, "CET", "timezone abbreviation must not appear in SQL literal")
	// Must use numeric offset (+02:00 style)
	assert.Contains(t, result, "+", "numeric offset must be present")
}

func TestSQLLiteral_Time_UTC(t *testing.T) {
	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result := sqlLiteral(ts)
	assert.Equal(t, "'"+ts.Format(time.RFC3339Nano)+"'", result)
}

func TestSQLLiteral_NullString(t *testing.T) {
	// StringifyValue returns "NULL" for nil; sqlLiteral must propagate that
	assert.Equal(t, "NULL", sqlLiteral(nil))
}

// ---------------------------------------------------------------------------
// editableString — pre-fills the inline-edit input field
// ---------------------------------------------------------------------------

func TestEditableString_Time_UsesRFC3339Nano(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Warsaw")
	require.NoError(t, err)
	ts := time.Date(2026, 3, 6, 15, 16, 45, 802794000, loc)

	result := editableString(ts)

	assert.Equal(t, ts.Format(time.RFC3339Nano), result)
	assert.NotContains(t, result, "CET", "CET abbreviation must not appear — PostgreSQL rejects it")
}

func TestEditableString_NonTime_FallsBackToStringify(t *testing.T) {
	assert.Equal(t, "NULL", editableString(nil))
	assert.Equal(t, "hello", editableString("hello"))
	assert.Equal(t, "42", editableString(int64(42)))
}

// ---------------------------------------------------------------------------
// Round-trip invariant:
// editableString(v) fed back through sqlLiteral must be PostgreSQL-parseable
// (i.e. no timezone abbreviation, quoted, offset-based).
// ---------------------------------------------------------------------------

func TestRoundTrip_TimeValue_NoTimezoneAbbreviation(t *testing.T) {
	locs := []string{"Europe/Warsaw", "America/New_York", "Asia/Tokyo", "UTC"}
	for _, locName := range locs {
		t.Run(locName, func(t *testing.T) {
			loc, err := time.LoadLocation(locName)
			require.NoError(t, err)
			ts := time.Date(2026, 6, 15, 12, 30, 0, 123456000, loc)

			editable := editableString(ts)
			literal := sqlLiteral(ts)

			// Both must NOT contain named timezone abbreviations
			for _, abbr := range []string{"CET", "CEST", "EST", "EDT", "JST"} {
				assert.NotContains(t, editable, abbr, "editable string must not contain %s", abbr)
				assert.NotContains(t, literal, abbr, "SQL literal must not contain %s", abbr)
			}

			// editable string must be parseable back as RFC3339Nano
			parsed, err := time.Parse(time.RFC3339Nano, editable)
			require.NoError(t, err, "editableString output must be parseable as RFC3339Nano")
			assert.True(t, ts.Equal(parsed), "parsed time must equal original")
		})
	}
}
