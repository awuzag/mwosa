//go:build integration

package instrument

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/awuzag/mwosa/internal/integrationtest"
	provider "github.com/awuzag/mwosa/providers/core"
	coreinstrument "github.com/awuzag/mwosa/providers/core/instrument"
	instrumentservice "github.com/awuzag/mwosa/service/instrument"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestMongoInstrumentRepositoryMatchesSQLiteContract(t *testing.T) {
	ctx := context.Background()
	server := integrationtest.StartMongoDB(t)
	runtime, err := storagemongodb.NewRuntime(ctx, storagemongodb.Config{
		URI:      server.URI,
		Database: "mwosa_instrument_contract_test",
	})
	if err != nil {
		t.Fatalf("new mongodb runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Fatalf("close mongodb runtime: %v", err)
		}
	})
	if err := runtime.Init(ctx); err != nil {
		t.Fatalf("init mongodb runtime: %v", err)
	}

	sqliteDatabase := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		if err := sqliteDatabase.Close(); err != nil {
			t.Fatalf("close sqlite database: %v", err)
		}
	})
	sqliteRepository, err := NewRepository(sqliteDatabase)
	if err != nil {
		t.Fatalf("new sqlite repository: %v", err)
	}
	mongoRepository, err := NewMongoRepository(runtime.Database())
	if err != nil {
		t.Fatalf("new mongodb repository: %v", err)
	}

	assertInstrumentRepositoryContract(t, sqliteRepository)
	assertInstrumentRepositoryContract(t, mongoRepository)
	assertMongoInstrumentDocumentShape(t, runtime)
}

func assertInstrumentRepositoryContract(t *testing.T, repository instrumentservice.Repository) {
	t.Helper()

	ctx := context.Background()
	item := samsungInstrument("Samsung Electronics")
	if _, err := repository.UpsertInstruments(ctx, []coreinstrument.Instrument{item}); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	item.Extensions["issueEnglishName"] = "Samsung Electronics Co Ltd"
	delete(item.Extensions, "parValue")
	result, err := repository.UpsertInstruments(ctx, []coreinstrument.Instrument{item})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if result.InstrumentsWritten != 1 || result.RowsAffected != 1 {
		t.Fatalf("write result = %+v, want one instrument", result)
	}

	for _, query := range []string{"Samsung", "삼성", "005930", "KR7005930003"} {
		found, err := repository.SearchInstruments(ctx, instrumentservice.Query{
			ProviderID:   provider.ProviderKRX,
			Market:       provider.MarketKRX,
			SecurityType: provider.SecurityTypeStock,
			Query:        query,
			Limit:        10,
		})
		if err != nil {
			t.Fatalf("search %q: %v", query, err)
		}
		if len(found.Instruments) != 1 {
			t.Fatalf("search %q len = %d, want 1", query, len(found.Instruments))
		}
		got := found.Instruments[0]
		if got.SecurityCode != "005930" || got.Extensions["issueEnglishName"] != "Samsung Electronics Co Ltd" {
			t.Fatalf("search %q result = %+v, want Samsung Electronics Co Ltd", query, got)
		}
	}

	inspected, err := repository.InspectInstrument(ctx, instrumentservice.Query{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       "KR7005930003",
	})
	if err != nil {
		t.Fatalf("inspect by isin: %v", err)
	}
	if inspected.SecurityCode != "005930" || inspected.ISIN != "KR7005930003" {
		t.Fatalf("inspect result = %+v, want 005930/KR7005930003", inspected)
	}
	if inspected.Extensions["parValue"] != "" {
		t.Fatalf("stale extension parValue = %q, want removed", inspected.Extensions["parValue"])
	}
}

func assertMongoInstrumentDocumentShape(t *testing.T, runtime *storagemongodb.Runtime) {
	t.Helper()

	var stored struct {
		ID            bson.ObjectID  `bson:"_id"`
		InstrumentKey string         `bson:"instrument_key"`
		SchemaVersion string         `bson:"schema_version"`
		Revision      int64          `bson:"revision"`
		Sources       []bson.M       `bson:"sources"`
		Extensions    map[string]any `bson:"extensions"`
	}
	if err := runtime.Database().
		Collection("instruments").
		FindOne(context.Background(), bson.D{{Key: "instrument_key", Value: "instruments:krx:stock:005930"}}).
		Decode(&stored); err != nil {
		t.Fatalf("find mongodb instrument document: %v", err)
	}
	if stored.ID.IsZero() {
		t.Fatalf("mongodb _id is zero, want ObjectID")
	}
	if stored.InstrumentKey != "instruments:krx:stock:005930" {
		t.Fatalf("instrument_key = %q, want stable natural key", stored.InstrumentKey)
	}
	if stored.SchemaVersion == "" || stored.Revision < 2 {
		t.Fatalf("common fields = schema %q revision %d, want revision incremented", stored.SchemaVersion, stored.Revision)
	}
	if len(stored.Sources) != 1 {
		t.Fatalf("sources len = %d, want embedded source", len(stored.Sources))
	}
	if stored.Extensions["issueEnglishName"] != "Samsung Electronics Co Ltd" {
		t.Fatalf("extensions = %#v, want embedded latest extensions", stored.Extensions)
	}
	if _, ok := stored.Extensions["parValue"]; ok {
		t.Fatalf("extensions = %#v, want stale extension removed", stored.Extensions)
	}
}
