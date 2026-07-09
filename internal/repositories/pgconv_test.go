package repositories

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestPgConv_TextRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "empty string", in: ""},
		{name: "non-empty string", in: "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := toPgText(tt.in)
			assert.Equal(t, tt.in != "", pg.Valid)
			assert.Equal(t, tt.in, pg.String)
			assert.Equal(t, tt.in, fromPgText(pg))
		})
	}

	t.Run("fromPgText invalid returns empty", func(t *testing.T) {
		assert.Equal(t, "", fromPgText(pgtype.Text{Valid: false, String: "ignored"}))
	})
}

func TestPgConv_DateRoundTrip(t *testing.T) {
	zero := time.Time{}
	nonZero := time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)

	t.Run("zero time", func(t *testing.T) {
		pg := toPgDate(zero)
		assert.False(t, pg.Valid)
		assert.True(t, fromPgDate(pg).IsZero())
	})

	t.Run("non-zero time", func(t *testing.T) {
		pg := toPgDate(nonZero)
		assert.True(t, pg.Valid)
		assert.True(t, nonZero.Equal(pg.Time))
		assert.True(t, nonZero.Equal(fromPgDate(pg)))
	})

	t.Run("fromPgDate invalid returns zero", func(t *testing.T) {
		assert.True(t, fromPgDate(pgtype.Date{Valid: false, Time: nonZero}).IsZero())
	})
}

func TestPgConv_DatePtrRoundTrip(t *testing.T) {
	nonZero := time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)

	t.Run("nil pointer", func(t *testing.T) {
		pg := toPgDatePtr(nil)
		assert.False(t, pg.Valid)
		assert.Nil(t, fromPgDatePtr(pg))
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		pg := toPgDatePtr(&nonZero)
		assert.True(t, pg.Valid)
		got := fromPgDatePtr(pg)
		assert.NotNil(t, got)
		assert.True(t, nonZero.Equal(*got))
	})

	t.Run("fromPgDatePtr invalid returns nil", func(t *testing.T) {
		assert.Nil(t, fromPgDatePtr(pgtype.Date{Valid: false, Time: nonZero}))
	})
}
