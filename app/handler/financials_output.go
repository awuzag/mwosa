package handler

import "github.com/awuzag/mwosa/providers/core/financials"

type FinancialStatementsOutput []financials.Statement

func (o FinancialStatementsOutput) JSONValue() any {
	return []financials.Statement(o)
}

func (o FinancialStatementsOutput) NDJSONRows() any {
	return []financials.Statement(o)
}

func (o FinancialStatementsOutput) CSVRows() any {
	return flattenFinancialStatementLines([]financials.Statement(o))
}

func (o FinancialStatementsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0)
	for _, statement := range o {
		for _, line := range statement.Lines {
			rows = append(rows, []string{
				string(statement.Statement),
				statement.FiscalYear,
				string(statement.Period),
				line.AccountName,
				line.Value,
				firstNonEmpty(line.Currency, statement.Currency),
				firstNonEmpty(line.Unit, statement.Unit),
			})
		}
	}
	return []string{"statement", "year", "period", "account", "value", "currency", "unit"}, rows
}

type financialStatementLineRow struct {
	Statement    financials.StatementType `csv:"statement"`
	Symbol       string                   `csv:"symbol"`
	FiscalYear   string                   `csv:"fiscal_year"`
	FiscalPeriod string                   `csv:"fiscal_period"`
	Period       financials.PeriodType    `csv:"period"`
	AccountID    string                   `csv:"account_id"`
	AccountName  string                   `csv:"account_name"`
	Value        string                   `csv:"value"`
	Currency     string                   `csv:"currency"`
	Unit         string                   `csv:"unit"`
}

func flattenFinancialStatementLines(statements []financials.Statement) []financialStatementLineRow {
	rows := make([]financialStatementLineRow, 0)
	for _, statement := range statements {
		for _, line := range statement.Lines {
			rows = append(rows, financialStatementLineRow{
				Statement:    statement.Statement,
				Symbol:       statement.Symbol,
				FiscalYear:   statement.FiscalYear,
				FiscalPeriod: statement.FiscalPeriod,
				Period:       statement.Period,
				AccountID:    line.AccountID,
				AccountName:  line.AccountName,
				Value:        line.Value,
				Currency:     firstNonEmpty(line.Currency, statement.Currency),
				Unit:         firstNonEmpty(line.Unit, statement.Unit),
			})
		}
	}
	return rows
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
