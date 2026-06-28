package mongodb

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestISOTimeFormatsUTCWithMilliseconds(t *testing.T) {
	value := ISOTime(time.Date(2026, 6, 28, 12, 34, 56, 789123456, time.FixedZone("KST", 9*60*60)))

	data, err := json.Marshal(value)
	require.NoError(t, err)
	require.Equal(t, `"2026-06-28T03:34:56.789Z"`, string(data))
}

func TestNewDocumentFieldsSetsCommonFields(t *testing.T) {
	now := time.Date(2026, 6, 28, 3, 34, 56, 789000000, time.UTC)

	fields, err := NewDocumentFields("daily_bars:krx:etf:069500:2026-06-28", "1.0.0", now)
	require.NoError(t, err)

	require.Equal(t, "daily_bars:krx:etf:069500:2026-06-28", fields.ID)
	require.Equal(t, "1.0.0", fields.SchemaVersion)
	require.Equal(t, int64(1), fields.Revision)
	require.True(t, fields.CreatedAt.Equal(now))
	require.True(t, fields.UpdatedAt.Equal(now))
}

func TestNewDocumentFieldsRejectsMissingRequiredValues(t *testing.T) {
	_, err := NewDocumentFields("", "1.0.0", time.Now())
	require.Error(t, err)

	_, err = NewDocumentFields("id", "", time.Now())
	require.Error(t, err)
}

func TestRevisionConflictErrorIncludesCollectionAndID(t *testing.T) {
	err := NewRevisionConflictError("daily_bars", "bar-1", 2)

	require.Error(t, err)
	require.Contains(t, err.Error(), "daily_bars")
	require.Contains(t, err.Error(), "bar-1")
	require.Contains(t, err.Error(), "revision")
}
