//go:build integration

package macro

import (
	"context"
	"strings"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	macroservice "github.com/awuzag/mwosa/service/macro"
	"github.com/awuzag/mwosa/storage"
)

func TestPostgresMacroRepositoryStoresIndicatorsDocumentsAndObservations(t *testing.T) {
	ctx := context.Background()
	postgres := integrationtest.StartPostgres(t)
	database := storage.NewDatabaseWithConfig(storage.DatabaseConfig{
		Backend: storage.BackendPostgres,
		URL:     postgres.DSN,
	})
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Fatalf("close database: %v", err)
		}
	})
	reader, writer, err := NewRepository(database)
	if err != nil {
		t.Fatalf("NewRepository error = %v", err)
	}

	indicator := macrorole.Indicator{
		ID:           "ecos.base-rate",
		Preset:       macrorole.PresetKeyStatistics,
		Provider:     provider.ProviderECOS,
		SourceCode:   "722Y001",
		SourceName:   "ECOS key statistics",
		SourceURL:    "https://ecos.bok.or.kr",
		Name:         "Bank of Korea base rate",
		FriendlyName: "base rate",
		Category:     "rates",
		Frequency:    macrorole.FrequencyMonthly,
		Unit:         "%",
		Scale:        "percent",
		Active:       true,
		ProviderDoc: &macrorole.ProviderDocument{
			Provider:      provider.ProviderECOS,
			SchemaVersion: "1.0.0",
			Document: map[string]any{
				"stat_code": "722Y001",
				"item_code": "0101000",
			},
			UpdatedAt: "2024-04-12T00:00:00Z",
		},
	}
	if _, err := writer.UpsertIndicators(ctx, []macrorole.Indicator{indicator}); err != nil {
		t.Fatalf("upsert indicator: %v", err)
	}
	indicator.FriendlyName = "policy rate"
	indicator.ProviderDoc.Document["item_code"] = "0101001"
	if _, err := writer.UpsertIndicators(ctx, []macrorole.Indicator{indicator}); err != nil {
		t.Fatalf("second upsert indicator: %v", err)
	}

	indicators, err := reader.QueryIndicators(ctx, macroservice.IndicatorQuery{
		ProviderID: provider.ProviderECOS,
		Preset:     macrorole.PresetKeyStatistics,
	})
	if err != nil {
		t.Fatalf("query indicators: %v", err)
	}
	if len(indicators) != 1 {
		t.Fatalf("indicators len = %d, want 1", len(indicators))
	}
	if indicators[0].FriendlyName != "policy rate" || indicators[0].SourceName != "ECOS key statistics" || !indicators[0].Active {
		t.Fatalf("indicator = %+v", indicators[0])
	}

	if _, err := writer.UpsertObservations(ctx, []macrorole.Observation{
		{
			IndicatorID: "ecos.base-rate",
			Provider:    provider.ProviderECOS,
			SourceCode:  "722Y001",
			Period:      "2024-04",
			Value:       "3.50",
			PublishedAt: "2024-04-11",
			CollectedAt: "2024-04-12T00:00:00Z",
			Revision:    0,
		},
	}); err != nil {
		t.Fatalf("upsert observations: %v", err)
	}

	observations, err := reader.QueryObservations(ctx, macroservice.ObservationQuery{
		IndicatorID: "ecos.base-rate",
		From:        "2024-04",
		To:          "2024-04",
	})
	if err != nil {
		t.Fatalf("query observations: %v", err)
	}
	if len(observations) != 1 || observations[0].Period != "2024-04" || observations[0].Provider != provider.ProviderECOS {
		t.Fatalf("observations = %+v", observations)
	}

	client, err := database.Client(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	var documentJSON string
	if err := client.QueryRowContext(ctx, `SELECT document_json FROM macro_indicator_provider_doc WHERE indicator_id = 'ecos.base-rate'`).Scan(&documentJSON); err != nil {
		t.Fatalf("query provider document: %v", err)
	}
	if !strings.Contains(documentJSON, `"item_code": "0101001"`) && !strings.Contains(documentJSON, `"item_code":"0101001"`) {
		t.Fatalf("document_json = %s", documentJSON)
	}
}
