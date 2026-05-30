package financialmetric

import (
	"context"
	"path/filepath"
	"testing"

	provider "github.com/awuzag/mwosa/providers/core"
	financialsrole "github.com/awuzag/mwosa/providers/core/financials"
	"github.com/awuzag/mwosa/storage"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/awuzag/mwosa/storage/financialstatement"
	"github.com/stretchr/testify/require"
)

func TestRepositoryCalculatesAndListsMetrics(t *testing.T) {
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
		statement("2024", []financialsrole.LineItem{
			line("ifrs-full_Revenue", "매출액", "1,000", "IS"),
			line("ifrs-full_OperatingIncomeLoss", "영업이익", "100", "IS"),
			line("ifrs-full_ProfitLoss", "당기순이익", "50", "IS"),
			line("ifrs-full_Assets", "자산총계", "500", "BS"),
			line("ifrs-full_Liabilities", "부채총계", "200", "BS"),
			line("ifrs-full_Equity", "자본총계", "300", "BS"),
		}),
		statement("2025", []financialsrole.LineItem{
			line("ifrs-full_Revenue", "매출액", "1,200", "IS"),
			line("ifrs-full_OperatingIncomeLoss", "영업이익", "180", "IS"),
			line("ifrs-full_ProfitLoss", "당기순이익", "90", "IS"),
			line("ifrs-full_Assets", "자산총계", "600", "BS"),
			line("ifrs-full_Liabilities", "부채총계", "240", "BS"),
			line("ifrs-full_Equity", "자본총계", "360", "BS"),
		}),
	})
	require.NoError(t, err)

	repository, err := NewRepository(database)
	require.NoError(t, err)
	result, err := repository.CalculateAndUpsert(ctx, company, CalculateOptions{WindowYears: 2, Period: financialsrole.PeriodTypeAnnual})
	require.NoError(t, err)
	require.Equal(t, 20, result.MetricsCalculated)
	require.Equal(t, 20, result.MetricsWritten)
	require.Equal(t, 7, result.Uncomputable)

	metrics, err := repository.ListMetrics(ctx, company, Query{WindowYears: 2, Period: financialsrole.PeriodTypeAnnual})
	require.NoError(t, err)
	require.NotEmpty(t, metrics)
	require.Contains(t, metricDecimals(metrics), "revenue_growth_yoy=0.20000000")
	require.Contains(t, metricDecimals(metrics), "operating_margin=0.15000000")
}

func statement(year string, lines []financialsrole.LineItem) financialsrole.Statement {
	return financialsrole.Statement{
		Statement:    financialsrole.StatementTypeSummary,
		Symbol:       "005930",
		Name:         "삼성전자",
		FiscalYear:   year,
		FiscalPeriod: "11011",
		Period:       financialsrole.PeriodTypeAnnual,
		Provider:     provider.ProviderOpenDART,
		Group:        provider.GroupOpenDARTFinancials,
		Operation:    provider.OperationOpenDARTSinglAcntAll,
		Extensions: map[string]string{
			"reprt_code": "11011",
			"fs_div":     "CFS",
		},
		Lines: lines,
	}
}

func line(id string, name string, value string, sjDiv string) financialsrole.LineItem {
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

func metricDecimals(metrics []Metric) []string {
	out := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		if metric.ValueDecimal == "" {
			continue
		}
		out = append(out, metric.Metric+"="+metric.ValueDecimal)
	}
	return out
}
