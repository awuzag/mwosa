package opendart

import (
	"context"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/providers/core/financials"
	opendartsdk "github.com/awuzag/opendart"
	"github.com/samber/oops"
)

const (
	reportCodeAnnual       = "11011"
	reportCodeThirdQuarter = "11014"
	fsDivConsolidated      = "CFS"
)

func (p *Provider) fetchFinancialStatements(ctx context.Context, input financials.FetchInput) (financials.FetchResult, error) {
	symbol := strings.TrimSpace(input.Symbol)
	errb := oops.In("opendart_adapter").With(
		"provider", provider.ProviderOpenDART,
		"role", provider.RoleFinancials,
		"symbol", symbol,
		"fiscal_year", input.FiscalYear,
		"period", input.Period,
		"statement", input.Statement,
	)
	if err := p.requireClient(); err != nil {
		return financials.FetchResult{}, errb.Wrap(err)
	}
	if symbol == "" {
		return financials.FetchResult{}, errb.New("OpenDART financials require corp_code or stock_code")
	}
	company, err := p.ResolveCompany(ctx, symbol)
	if err != nil {
		return financials.FetchResult{}, errb.Wrap(err)
	}
	fiscalYear := strings.TrimSpace(input.FiscalYear)
	if fiscalYear == "" {
		return financials.FetchResult{}, errb.New("OpenDART financials require --year")
	}
	reportCode := reportCodeForPeriod(input.Period)
	response, err := p.client.FnlttSinglAcntAll(ctx, opendartsdk.FnlttSinglAcntAllParams{
		CorpCode:  company.CorpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
		FsDiv:     fsDivConsolidated,
	})
	if err != nil {
		return financials.FetchResult{}, errb.Wrapf(err, "fetch OpenDART single-company full financial statements")
	}
	if err := ensureOpenDARTStatus(response.Status, response.Message, provider.OperationOpenDARTSinglAcntAll); err != nil {
		return financials.FetchResult{}, err
	}

	lines := make([]financials.LineItem, 0, len(response.List))
	for _, item := range response.List {
		statement := statementType(item.SjDiv)
		if input.Statement != "" && input.Statement != financials.StatementTypeSummary && input.Statement != statement {
			continue
		}
		lines = append(lines, financials.LineItem{
			AccountID:   item.AccountId,
			AccountName: item.AccountNm,
			Value:       item.ThstrmAmount,
			Currency:    item.Currency,
			Extensions: map[string]string{
				"corp_code":        company.CorpCode,
				"stock_code":       company.StockCode,
				"reprt_code":       item.ReprtCode,
				"fs_div":           fsDivConsolidated,
				"sj_div":           item.SjDiv,
				"sj_nm":            item.SjNm,
				"account_detail":   item.AccountDetail,
				"thstrm_nm":        item.ThstrmNm,
				"frmtrm_nm":        item.FrmtrmNm,
				"frmtrm_amount":    item.FrmtrmAmount,
				"bfefrmtrm_nm":     item.BfefrmtrmNm,
				"bfefrmtrm_amount": item.BfefrmtrmAmount,
				"ord":              item.Ord,
				"rcept_no":         item.RceptNo,
			},
		})
		if input.Limit > 0 && len(lines) >= input.Limit {
			break
		}
	}
	if len(lines) == 0 {
		return financials.FetchResult{}, nil
	}

	statement := financials.Statement{
		Statement:    input.Statement,
		Symbol:       company.StockCode,
		Name:         company.CorpName,
		FiscalYear:   fiscalYear,
		FiscalPeriod: reportCode,
		Period:       withDefaultPeriod(input.Period),
		Currency:     firstCurrency(lines),
		Lines:        lines,
		Extensions: map[string]string{
			"corp_code":     company.CorpCode,
			"stock_code":    company.StockCode,
			"corp_eng_name": company.CorpEngName,
			"modify_date":   company.ModifyDate,
			"reprt_code":    reportCode,
			"fs_div":        fsDivConsolidated,
		},
		Provider:     provider.ProviderOpenDART,
		Group:        provider.GroupOpenDARTFinancials,
		Operation:    provider.OperationOpenDARTSinglAcntAll,
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
	}
	if statement.Statement == "" {
		statement.Statement = financials.StatementTypeSummary
	}
	return financials.FetchResult{
		Statements: []financials.Statement{statement},
		Provider:   p.Identity,
		Group:      provider.GroupOpenDARTFinancials,
		Operation:  provider.OperationOpenDARTSinglAcntAll,
		TotalCount: len(lines),
	}, nil
}

func reportCodeForPeriod(period financials.PeriodType) string {
	switch period {
	case financials.PeriodTypeQuarter:
		return reportCodeThirdQuarter
	default:
		return reportCodeAnnual
	}
}

func withDefaultPeriod(period financials.PeriodType) financials.PeriodType {
	if period == "" {
		return financials.PeriodTypeAnnual
	}
	return period
}

func statementType(sjDiv string) financials.StatementType {
	switch strings.ToUpper(strings.TrimSpace(sjDiv)) {
	case "BS":
		return financials.StatementTypeBalanceSheet
	case "IS", "CIS":
		return financials.StatementTypeIncomeStatement
	case "CF":
		return financials.StatementTypeCashFlow
	default:
		return financials.StatementTypeSummary
	}
}

func firstCurrency(lines []financials.LineItem) string {
	for _, line := range lines {
		if strings.TrimSpace(line.Currency) != "" {
			return line.Currency
		}
	}
	return ""
}
