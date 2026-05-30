package opendart

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	opendartsdk "github.com/awuzag/opendart"
	"github.com/samber/oops"
)

type PeriodicFactRequest struct {
	CorpCode   string
	FiscalYear string
	ReportCode string
}

type DividendFactRequest = PeriodicFactRequest

type CompanyFact struct {
	Provider                       provider.ProviderID
	Group                          provider.GroupID
	Operation                      provider.OperationID
	ProviderCompanyIdentifierType  string
	ProviderCompanyIdentifierValue string
	FactType                       string
	FiscalYear                     string
	ReportCode                     string
	RceptNo                        string
	FactDate                       string
	Key                            string
	ValueText                      string
	ValueNumber                    string
	CurrencyCode                   string
	Raw                            map[string]any
}

type CompanyFactResult struct {
	Provider   provider.ProviderID  `json:"provider" csv:"provider"`
	Group      provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation  provider.OperationID `json:"operation" csv:"operation"`
	Status     string               `json:"status,omitempty" csv:"status"`
	Message    string               `json:"message,omitempty" csv:"message"`
	Facts      []CompanyFact        `json:"facts" csv:"-"`
	TotalCount int                  `json:"total_count" csv:"total_count"`
}

type CompanyFactBatchResult struct {
	Provider   provider.ProviderID `json:"provider" csv:"provider"`
	Group      provider.GroupID    `json:"provider_group" csv:"provider_group"`
	Sources    []CompanyFactResult `json:"sources" csv:"-"`
	Facts      []CompanyFact       `json:"facts" csv:"-"`
	TotalCount int                 `json:"total_count" csv:"total_count"`
}

func (p *Provider) FetchPeriodicFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactBatchResult, error) {
	fetchers := []func(context.Context, PeriodicFactRequest) (CompanyFactResult, error){
		p.FetchDividendFacts,
		p.FetchTreasuryStockFacts,
		p.FetchMajorShareholderFacts,
		p.FetchMajorShareholderChangeFacts,
		p.FetchEmployeeFacts,
		p.FetchAuditOpinionFacts,
	}
	result := CompanyFactBatchResult{
		Provider: provider.ProviderOpenDART,
		Group:    provider.GroupOpenDARTPeriodicReport,
		Sources:  make([]CompanyFactResult, 0, len(fetchers)),
	}
	for _, fetch := range fetchers {
		source, err := fetch(ctx, req)
		if err != nil {
			return CompanyFactBatchResult{}, err
		}
		result.Sources = append(result.Sources, source)
		result.Facts = append(result.Facts, source.Facts...)
	}
	result.TotalCount = len(result.Facts)
	return result, nil
}

func (p *Provider) FetchDividendFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactResult, error) {
	corpCode, fiscalYear, reportCode, err := validatePeriodicFactRequest(req, provider.OperationOpenDARTAlotMatter, "dividend facts")
	if err != nil {
		return CompanyFactResult{}, err
	}
	errb := periodicFactErrorBuilder(provider.OperationOpenDARTAlotMatter, corpCode, fiscalYear, reportCode)
	if err := p.requireClient(); err != nil {
		return CompanyFactResult{}, errb.Wrap(err)
	}
	response, err := p.client.AlotMatter(ctx, opendartsdk.AlotMatterParams{
		CorpCode:  corpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
	})
	if err != nil {
		return CompanyFactResult{}, errb.Wrapf(err, "fetch OpenDART dividend matters")
	}
	if err := ensureOpenDARTFactStatus(response.Status, response.Message, provider.OperationOpenDARTAlotMatter); err != nil {
		return CompanyFactResult{}, err
	}
	facts := make([]CompanyFact, 0, len(response.List)*3)
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			facts = append(facts, dividendFactsFromItem(item, fiscalYear, reportCode)...)
		}
	}
	return CompanyFactResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTPeriodicReport,
		Operation:  provider.OperationOpenDARTAlotMatter,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Facts:      facts,
		TotalCount: len(facts),
	}, nil
}

func (p *Provider) FetchTreasuryStockFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactResult, error) {
	corpCode, fiscalYear, reportCode, err := validatePeriodicFactRequest(req, provider.OperationOpenDARTTesstkAcqs, "treasury stock facts")
	if err != nil {
		return CompanyFactResult{}, err
	}
	errb := periodicFactErrorBuilder(provider.OperationOpenDARTTesstkAcqs, corpCode, fiscalYear, reportCode)
	if err := p.requireClient(); err != nil {
		return CompanyFactResult{}, errb.Wrap(err)
	}
	response, err := p.client.TesstkAcqsDspsSttus(ctx, opendartsdk.TesstkAcqsDspsSttusParams{
		CorpCode:  corpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
	})
	if err != nil {
		return CompanyFactResult{}, errb.Wrapf(err, "fetch OpenDART treasury stock status")
	}
	if err := ensureOpenDARTFactStatus(response.Status, response.Message, provider.OperationOpenDARTTesstkAcqs); err != nil {
		return CompanyFactResult{}, err
	}
	facts := make([]CompanyFact, 0, len(response.List)*5)
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			facts = append(facts, treasuryStockFactsFromItem(item, fiscalYear, reportCode)...)
		}
	}
	return CompanyFactResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTPeriodicReport,
		Operation:  provider.OperationOpenDARTTesstkAcqs,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Facts:      facts,
		TotalCount: len(facts),
	}, nil
}

func (p *Provider) FetchMajorShareholderFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactResult, error) {
	corpCode, fiscalYear, reportCode, err := validatePeriodicFactRequest(req, provider.OperationOpenDARTHyslrSttus, "major shareholder facts")
	if err != nil {
		return CompanyFactResult{}, err
	}
	errb := periodicFactErrorBuilder(provider.OperationOpenDARTHyslrSttus, corpCode, fiscalYear, reportCode)
	if err := p.requireClient(); err != nil {
		return CompanyFactResult{}, errb.Wrap(err)
	}
	response, err := p.client.HyslrSttus(ctx, opendartsdk.HyslrSttusParams{
		CorpCode:  corpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
	})
	if err != nil {
		return CompanyFactResult{}, errb.Wrapf(err, "fetch OpenDART major shareholder status")
	}
	if err := ensureOpenDARTFactStatus(response.Status, response.Message, provider.OperationOpenDARTHyslrSttus); err != nil {
		return CompanyFactResult{}, err
	}
	facts := make([]CompanyFact, 0, len(response.List)*4)
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			facts = append(facts, majorShareholderFactsFromItem(item, fiscalYear, reportCode)...)
		}
	}
	return CompanyFactResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTPeriodicReport,
		Operation:  provider.OperationOpenDARTHyslrSttus,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Facts:      facts,
		TotalCount: len(facts),
	}, nil
}

func (p *Provider) FetchMajorShareholderChangeFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactResult, error) {
	corpCode, fiscalYear, reportCode, err := validatePeriodicFactRequest(req, provider.OperationOpenDARTHyslrChg, "major shareholder change facts")
	if err != nil {
		return CompanyFactResult{}, err
	}
	errb := periodicFactErrorBuilder(provider.OperationOpenDARTHyslrChg, corpCode, fiscalYear, reportCode)
	if err := p.requireClient(); err != nil {
		return CompanyFactResult{}, errb.Wrap(err)
	}
	response, err := p.client.HyslrChgSttus(ctx, opendartsdk.HyslrChgSttusParams{
		CorpCode:  corpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
	})
	if err != nil {
		return CompanyFactResult{}, errb.Wrapf(err, "fetch OpenDART major shareholder changes")
	}
	if err := ensureOpenDARTFactStatus(response.Status, response.Message, provider.OperationOpenDARTHyslrChg); err != nil {
		return CompanyFactResult{}, err
	}
	facts := make([]CompanyFact, 0, len(response.List)*2)
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			facts = append(facts, majorShareholderChangeFactsFromItem(item, fiscalYear, reportCode)...)
		}
	}
	return CompanyFactResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTPeriodicReport,
		Operation:  provider.OperationOpenDARTHyslrChg,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Facts:      facts,
		TotalCount: len(facts),
	}, nil
}

func (p *Provider) FetchEmployeeFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactResult, error) {
	corpCode, fiscalYear, reportCode, err := validatePeriodicFactRequest(req, provider.OperationOpenDARTEmpSttus, "employee facts")
	if err != nil {
		return CompanyFactResult{}, err
	}
	errb := periodicFactErrorBuilder(provider.OperationOpenDARTEmpSttus, corpCode, fiscalYear, reportCode)
	if err := p.requireClient(); err != nil {
		return CompanyFactResult{}, errb.Wrap(err)
	}
	response, err := p.client.EmpSttus(ctx, opendartsdk.EmpSttusParams{
		CorpCode:  corpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
	})
	if err != nil {
		return CompanyFactResult{}, errb.Wrapf(err, "fetch OpenDART employee status")
	}
	if err := ensureOpenDARTFactStatus(response.Status, response.Message, provider.OperationOpenDARTEmpSttus); err != nil {
		return CompanyFactResult{}, err
	}
	facts := make([]CompanyFact, 0, len(response.List)*6)
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			facts = append(facts, employeeFactsFromItem(item, fiscalYear, reportCode)...)
		}
	}
	return CompanyFactResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTPeriodicReport,
		Operation:  provider.OperationOpenDARTEmpSttus,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Facts:      facts,
		TotalCount: len(facts),
	}, nil
}

func (p *Provider) FetchAuditOpinionFacts(ctx context.Context, req PeriodicFactRequest) (CompanyFactResult, error) {
	corpCode, fiscalYear, reportCode, err := validatePeriodicFactRequest(req, provider.OperationOpenDARTAuditOpinion, "audit opinion facts")
	if err != nil {
		return CompanyFactResult{}, err
	}
	errb := periodicFactErrorBuilder(provider.OperationOpenDARTAuditOpinion, corpCode, fiscalYear, reportCode)
	if err := p.requireClient(); err != nil {
		return CompanyFactResult{}, errb.Wrap(err)
	}
	response, err := p.client.AccnutAdtorNmNdAdtOpinion(ctx, opendartsdk.AccnutAdtorNmNdAdtOpinionParams{
		CorpCode:  corpCode,
		BsnsYear:  fiscalYear,
		ReprtCode: reportCode,
	})
	if err != nil {
		return CompanyFactResult{}, errb.Wrapf(err, "fetch OpenDART audit opinion")
	}
	if err := ensureOpenDARTFactStatus(response.Status, response.Message, provider.OperationOpenDARTAuditOpinion); err != nil {
		return CompanyFactResult{}, err
	}
	facts := make([]CompanyFact, 0, len(response.List)*4)
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			facts = append(facts, auditOpinionFactsFromItem(item, fiscalYear, reportCode)...)
		}
	}
	return CompanyFactResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTPeriodicReport,
		Operation:  provider.OperationOpenDARTAuditOpinion,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Facts:      facts,
		TotalCount: len(facts),
	}, nil
}

func validatePeriodicFactRequest(req PeriodicFactRequest, operation provider.OperationID, label string) (string, string, string, error) {
	corpCode := strings.TrimSpace(req.CorpCode)
	fiscalYear := strings.TrimSpace(req.FiscalYear)
	reportCode := strings.TrimSpace(req.ReportCode)
	if reportCode == "" {
		reportCode = reportCodeAnnual
	}
	errb := oops.In("opendart_adapter").With(
		"provider", provider.ProviderOpenDART,
		"operation", operation,
		"corp_code", corpCode,
		"fiscal_year", fiscalYear,
		"report_code", reportCode,
	)
	if corpCode == "" {
		return "", "", "", errb.Errorf("OpenDART %s require corp_code", label)
	}
	if fiscalYear == "" {
		return "", "", "", errb.Errorf("OpenDART %s require fiscal year", label)
	}
	return corpCode, fiscalYear, reportCode, nil
}

func periodicFactErrorBuilder(operation provider.OperationID, corpCode string, fiscalYear string, reportCode string) oops.OopsErrorBuilder {
	return oops.In("opendart_adapter").With(
		"provider", provider.ProviderOpenDART,
		"group", provider.GroupOpenDARTPeriodicReport,
		"operation", operation,
		"corp_code", corpCode,
		"fiscal_year", fiscalYear,
		"report_code", reportCode,
	)
}

func ensureOpenDARTFactStatus(status string, message string, operation provider.OperationID) error {
	if isOpenDARTNoDataStatus(status) {
		return nil
	}
	return ensureOpenDARTStatus(status, message, operation)
}

func dividendFactsFromItem(item opendartsdk.AlotMatterItem, fiscalYear string, reportCode string) []CompanyFact {
	raw := rawMap(item)
	periods := []struct {
		key   string
		value string
	}{
		{key: "thstrm", value: item.Thstrm},
		{key: "frmtrm", value: item.Frmtrm},
		{key: "lwfr", value: item.Lwfr},
	}
	facts := make([]CompanyFact, 0, len(periods))
	for _, period := range periods {
		value := strings.TrimSpace(period.value)
		if value == "" || value == "-" {
			continue
		}
		keyParts := []string{period.key}
		if strings.TrimSpace(item.Se) != "" {
			keyParts = append(keyParts, strings.TrimSpace(item.Se))
		}
		if strings.TrimSpace(item.StockKnd) != "" {
			keyParts = append(keyParts, strings.TrimSpace(item.StockKnd))
		}
		facts = append(facts, CompanyFact{
			Provider:                       provider.ProviderOpenDART,
			Group:                          provider.GroupOpenDARTPeriodicReport,
			Operation:                      provider.OperationOpenDARTAlotMatter,
			ProviderCompanyIdentifierType:  "dart_corp_code",
			ProviderCompanyIdentifierValue: strings.TrimSpace(item.CorpCode),
			FactType:                       "dividend",
			FiscalYear:                     fiscalYear,
			ReportCode:                     reportCode,
			RceptNo:                        strings.TrimSpace(item.RceptNo),
			FactDate:                       strings.TrimSpace(item.StlmDt),
			Key:                            strings.Join(keyParts, ":"),
			ValueText:                      value,
			ValueNumber:                    numericText(value),
			CurrencyCode:                   "KRW",
			Raw:                            raw,
		})
	}
	return facts
}

func treasuryStockFactsFromItem(item opendartsdk.TesstkAcqsDspsSttusItem, fiscalYear string, reportCode string) []CompanyFact {
	return numericFactsFromParts("treasury_stock", provider.OperationOpenDARTTesstkAcqs, item.CorpCode, fiscalYear, reportCode, item.RceptNo, item.StlmDt, rawMap(item),
		[]string{item.StockKnd, item.AcqsMth1, item.AcqsMth2, item.AcqsMth3},
		[]factMetric{
			{Key: "beginning_quantity", Value: item.BsisQy},
			{Key: "acquired_quantity", Value: item.ChangeQyAcqs},
			{Key: "disposed_quantity", Value: item.ChangeQyDsps},
			{Key: "cancelled_quantity", Value: item.ChangeQyIncnr},
			{Key: "ending_quantity", Value: item.TrmendQy},
		},
	)
}

func majorShareholderFactsFromItem(item opendartsdk.HyslrSttusItem, fiscalYear string, reportCode string) []CompanyFact {
	return numericFactsFromParts("major_shareholder", provider.OperationOpenDARTHyslrSttus, item.CorpCode, fiscalYear, reportCode, item.RceptNo, item.StlmDt, rawMap(item),
		[]string{item.Nm, item.Relate, item.StockKnd},
		[]factMetric{
			{Key: "beginning_shares", Value: item.BsisPosesnStockCo},
			{Key: "beginning_ownership_ratio", Value: item.BsisPosesnStockQotaRt},
			{Key: "ending_shares", Value: item.TrmendPosesnStockCo},
			{Key: "ending_ownership_ratio", Value: item.TrmendPosesnStockQotaRt},
		},
	)
}

func majorShareholderChangeFactsFromItem(item opendartsdk.HyslrChgSttusItem, fiscalYear string, reportCode string) []CompanyFact {
	return numericFactsFromParts("major_shareholder_change", provider.OperationOpenDARTHyslrChg, item.CorpCode, fiscalYear, reportCode, item.RceptNo, firstOpenDARTDate(item.ChangeOn, item.StlmDt), rawMap(item),
		[]string{item.MxmmShrholdrNm, item.ChangeCause},
		[]factMetric{
			{Key: "shares", Value: item.PosesnStockCo},
			{Key: "ownership_ratio", Value: item.QotaRt},
		},
	)
}

func employeeFactsFromItem(item opendartsdk.EmpSttusItem, fiscalYear string, reportCode string) []CompanyFact {
	return numericFactsFromParts("employee", provider.OperationOpenDARTEmpSttus, item.CorpCode, fiscalYear, reportCode, item.RceptNo, item.StlmDt, rawMap(item),
		[]string{item.FoBbm, item.Sexdstn},
		[]factMetric{
			{Key: "total_count", Value: item.Sm},
			{Key: "regular_count", Value: item.RgllbrCo},
			{Key: "regular_avg_service_years", Value: item.RgllbrAbacptLabrrCo},
			{Key: "contract_count", Value: item.CnttkCo},
			{Key: "contract_avg_service_years", Value: item.CnttkAbacptLabrrCo},
			{Key: "annual_salary_total", Value: item.FyerSalaryTotamt, Currency: "KRW"},
			{Key: "average_salary", Value: item.JanSalaryAm, Currency: "KRW"},
		},
	)
}

func auditOpinionFactsFromItem(item opendartsdk.AccnutAdtorNmNdAdtOpinionItem, fiscalYear string, reportCode string) []CompanyFact {
	raw := rawMap(item)
	period := firstNonEmpty(item.BsnsYear, fiscalYear)
	return textFactsFromParts("audit_opinion", provider.OperationOpenDARTAuditOpinion, item.CorpCode, fiscalYear, reportCode, item.RceptNo, item.StlmDt, raw,
		[]string{period},
		[]factMetric{
			{Key: "auditor", Value: item.Adtor},
			{Key: "opinion", Value: item.AdtOpinion},
			{Key: "emphasis_matter", Value: item.EmphsMatter},
			{Key: "core_audit_matter", Value: item.CoreAdtMatter},
			{Key: "special_matter", Value: item.AdtReprtSpcmntMatter},
		},
	)
}

type factMetric struct {
	Key      string
	Value    string
	Currency string
}

func numericFactsFromParts(factType string, operation provider.OperationID, corpCode string, fiscalYear string, reportCode string, rceptNo string, factDate string, raw map[string]any, keyParts []string, metrics []factMetric) []CompanyFact {
	return factsFromParts(factType, operation, corpCode, fiscalYear, reportCode, rceptNo, factDate, raw, keyParts, metrics, true)
}

func textFactsFromParts(factType string, operation provider.OperationID, corpCode string, fiscalYear string, reportCode string, rceptNo string, factDate string, raw map[string]any, keyParts []string, metrics []factMetric) []CompanyFact {
	return factsFromParts(factType, operation, corpCode, fiscalYear, reportCode, rceptNo, factDate, raw, keyParts, metrics, false)
}

func factsFromParts(factType string, operation provider.OperationID, corpCode string, fiscalYear string, reportCode string, rceptNo string, factDate string, raw map[string]any, keyParts []string, metrics []factMetric, numeric bool) []CompanyFact {
	facts := make([]CompanyFact, 0, len(metrics))
	for _, metric := range metrics {
		value := strings.TrimSpace(metric.Value)
		if value == "" || value == "-" {
			continue
		}
		number := ""
		if numeric {
			number = numericText(value)
			if number == "" {
				continue
			}
		}
		key := factKey(append(keyParts, metric.Key)...)
		facts = append(facts, CompanyFact{
			Provider:                       provider.ProviderOpenDART,
			Group:                          provider.GroupOpenDARTPeriodicReport,
			Operation:                      operation,
			ProviderCompanyIdentifierType:  "dart_corp_code",
			ProviderCompanyIdentifierValue: strings.TrimSpace(corpCode),
			FactType:                       factType,
			FiscalYear:                     fiscalYear,
			ReportCode:                     reportCode,
			RceptNo:                        strings.TrimSpace(rceptNo),
			FactDate:                       firstOpenDARTDate(factDate),
			Key:                            key,
			ValueText:                      value,
			ValueNumber:                    number,
			CurrencyCode:                   metric.Currency,
			Raw:                            raw,
		})
	}
	return facts
}

func factKey(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" || value == "-" {
			continue
		}
		out = append(out, value)
	}
	return strings.Join(out, ":")
}

func rawMap(value any) map[string]any {
	raw := map[string]any{}
	body, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	_ = json.Unmarshal(body, &raw)
	return raw
}

func numericText(value string) string {
	normalized := strings.NewReplacer(",", "", " ", "").Replace(strings.TrimSpace(value))
	if normalized == "" || normalized == "-" {
		return ""
	}
	if _, err := strconv.ParseFloat(normalized, 64); err != nil {
		return ""
	}
	return normalized
}
