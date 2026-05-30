package valuation

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyfact"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialstatement"
	"github.com/stretchr/testify/require"
)

func TestRepositoryCalculatesAndListsValuationSnapshot(t *testing.T) {
	ctx := context.Background()
	database := storage.NewDatabase(filepath.Join(t.TempDir(), "mwosa.db"))
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	companyRepository, err := companyidentity.NewRepository(database)
	require.NoError(t, err)
	_, err = companyRepository.UpsertCompanies(ctx, []companyidentity.CompanyInput{
		{
			Name:        "삼성전자",
			LegalName:   "삼성전자",
			CountryCode: "KR",
			Identifiers: []companyidentity.IdentifierInput{
				{
					Provider:        provider.ProviderOpenDART,
					Group:           provider.GroupOpenDARTDisclosure,
					Operation:       provider.OperationOpenDARTCorpCode,
					IdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
					IdentifierValue: "00126380",
					Primary:         true,
					Confidence:      1,
				},
				{
					Provider:        provider.ProviderOpenDART,
					Group:           provider.GroupOpenDARTDisclosure,
					Operation:       provider.OperationOpenDARTCorpCode,
					IdentifierType:  companyidentity.IdentifierTypeKRXStockCode,
					IdentifierValue: "005930",
					Confidence:      1,
				},
			},
			InstrumentRef: companyidentity.InstrumentRef{
				Market:       provider.MarketKRX,
				SecurityType: provider.SecurityTypeStock,
				Symbol:       "005930",
				Name:         "삼성전자",
				RelationType: companyidentity.RelationTypeIssuer,
			},
		},
	})
	require.NoError(t, err)
	company, err := companyRepository.Inspect(ctx, "005930")
	require.NoError(t, err)

	statementRepository, err := financialstatement.NewRepository(database)
	require.NoError(t, err)
	_, err = statementRepository.UpsertStatements(ctx, company, []financialsrole.Statement{
		{
			Statement:    financialsrole.StatementTypeSummary,
			Symbol:       "005930",
			Name:         "삼성전자",
			FiscalYear:   "2025",
			FiscalPeriod: "11011",
			Period:       financialsrole.PeriodTypeAnnual,
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTFinancials,
			Operation:    provider.OperationOpenDARTSinglAcntAll,
			Extensions: map[string]string{
				"reprt_code": "11011",
				"fs_div":     "CFS",
			},
			Lines: []financialsrole.LineItem{
				valuationLine("ifrs-full_Revenue", "매출액", "2000000", "IS"),
				valuationLine("ifrs-full_ProfitLoss", "당기순이익", "100000", "IS"),
				valuationLine("ifrs-full_Equity", "자본총계", "500000", "BS"),
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, insertPriceFixture(ctx, database, company.Instruments[0].InstrumentID))
	require.NoError(t, insertDividendFixture(ctx, database, company))

	repository, err := NewRepository(database)
	require.NoError(t, err)
	snapshot, err := repository.CalculateAndUpsert(ctx, company, CalculateOptions{AsOf: "latest"})
	require.NoError(t, err)
	require.Equal(t, "2026-05-16", snapshot.AsOfDate)
	require.Equal(t, int64(100000), *snapshot.PerBP)
	require.Equal(t, int64(20000), *snapshot.PbrBP)
	require.Equal(t, int64(5000), *snapshot.PsrBP)
	require.Equal(t, int64(10), *snapshot.EpsMinor)
	require.Equal(t, int64(50), *snapshot.BpsMinor)
	require.Equal(t, int64(500), *snapshot.DividendYieldBP)
	require.NotContains(t, snapshot.Uncomputable, "dividend_yield")

	snapshots, err := repository.ListSnapshots(ctx, company, Query{AsOf: "latest"})
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, snapshot.AsOfDate, snapshots[0].AsOfDate)
	require.Equal(t, int64(500), *snapshots[0].DividendYieldBP)
}

func valuationLine(id string, name string, value string, sjDiv string) financialsrole.LineItem {
	return financialsrole.LineItem{
		AccountID:   id,
		AccountName: name,
		Value:       value,
		Currency:    "KRW",
		Extensions: map[string]string{
			"sj_div":     sjDiv,
			"reprt_code": "11011",
			"fs_div":     "CFS",
			"rcept_no":   "rcept",
			"thstrm_nm":  "제",
		},
	}
}

func insertPriceFixture(ctx context.Context, database *storage.Database, instrumentID int64) error {
	client, err := database.Client(ctx)
	if err != nil {
		return err
	}
	nowMS := time.Now().UTC().UnixMilli()
	source := storage.ProviderSourceV2Row{
		Provider:      string(provider.ProviderDataGo),
		ProviderGroup: string(provider.GroupStockPrice),
		Operation:     string(provider.OperationGetStockPriceInfo),
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
	}
	if _, err := client.NewInsert().
		Model(&source).
		On("CONFLICT (provider, provider_group, operation) DO UPDATE").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return err
	}
	if err := client.NewSelect().
		Model(&source).
		Where("provider = ?", source.Provider).
		Where("provider_group = ?", source.ProviderGroup).
		Where("operation = ?", source.Operation).
		Limit(1).
		Scan(ctx); err != nil {
		return err
	}
	bar := storage.DailyBarV2Row{
		SchemaVersion:  storage.DailyBarV2SchemaVersion,
		InstrumentID:   instrumentID,
		SourceID:       source.ID,
		TradingDate:    20260516,
		ClosePrice:     sql.NullInt64{Int64: 1000000, Valid: true},
		MarketCapMinor: sql.NullInt64{Int64: 1000000, Valid: true},
		CreatedAtMS:    nowMS,
		UpdatedAtMS:    nowMS,
	}
	_, err = client.NewInsert().
		Model(&bar).
		On("CONFLICT (instrument_id, source_id, trading_date) DO UPDATE").
		Set("close_price = EXCLUDED.close_price").
		Set("market_cap_minor = EXCLUDED.market_cap_minor").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx)
	return err
}

func insertDividendFixture(ctx context.Context, database *storage.Database, company companyidentity.InspectResult) error {
	repository, err := companyfact.NewRepository(database)
	if err != nil {
		return err
	}
	_, err = repository.UpsertFacts(ctx, company, []companyfact.FactInput{
		{
			Provider:     provider.ProviderOpenDART,
			Group:        provider.GroupOpenDARTPeriodicReport,
			Operation:    provider.OperationOpenDARTAlotMatter,
			FactType:     companyfact.FactTypeDividend,
			FiscalYear:   "2025",
			ReportCode:   "11011",
			RceptNo:      "20260330000001",
			FactDate:     "2025-12-31",
			Key:          "thstrm:현금배당금총액:보통주",
			ValueText:    "50,000",
			ValueNumber:  "50000",
			CurrencyCode: "KRW",
			Raw:          map[string]string{"corp_code": "00126380"},
		},
	})
	return err
}
