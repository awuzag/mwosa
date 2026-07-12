package krxmonthly

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWriteAndReadArchive(t *testing.T) {
	rows := json.RawMessage(`[{"BAS_DD":"20260625","ISU_CD":"005930"}]`)
	snapshots := []RawSnapshot{
		{
			Provider:         "krx",
			ProviderGroup:    "stockDailyTrade",
			APIID:            "stk_bydd_trd",
			BaseDate:         "2026-06-25",
			RowCount:         1,
			CanonicalSupport: "daily_bar",
			Rows:             rows,
		},
	}
	path := filepath.Join(t.TempDir(), "fixture.zip")
	manifest, err := WriteArchive(path, BuildOptions{
		FixtureID:   "krx-stock-daily-2026-06",
		From:        "2026-06-01",
		To:          "2026-06-30",
		CollectedAt: time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC),
	}, snapshots)
	require.NoError(t, err)
	require.Equal(t, 1, manifest.SnapshotCount)
	require.Equal(t, 1, manifest.TotalRows)
	require.NotEmpty(t, manifest.DatasetSHA256)

	dataset, err := ReadArchive(path)
	require.NoError(t, err)
	require.Equal(t, manifest, dataset.Manifest)
	require.Len(t, dataset.Snapshots, 1)
	require.Equal(t, snapshots[0].Provider, dataset.Snapshots[0].Provider)
	require.Equal(t, snapshots[0].ProviderGroup, dataset.Snapshots[0].ProviderGroup)
	require.Equal(t, snapshots[0].APIID, dataset.Snapshots[0].APIID)
	require.Equal(t, snapshots[0].BaseDate, dataset.Snapshots[0].BaseDate)
	require.Equal(t, snapshots[0].RowCount, dataset.Snapshots[0].RowCount)
	require.JSONEq(t, string(snapshots[0].Rows), string(dataset.Snapshots[0].Rows))
}
