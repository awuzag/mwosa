package opendart

import (
	"context"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	opendartsdk "github.com/awuzag/opendart"
	"github.com/samber/oops"
)

type FilingRequest struct {
	Identifier string
	CorpCode   string
	From       string
	To         string
	LastReport bool
	PageNo     string
	PageCount  string
}

type Filing struct {
	CorpCode  string `json:"corp_code" csv:"corp_code"`
	CorpName  string `json:"corp_name" csv:"corp_name"`
	StockCode string `json:"stock_code,omitempty" csv:"stock_code"`
	CorpClass string `json:"corp_cls,omitempty" csv:"corp_cls"`
	Report    string `json:"report_nm" csv:"report_nm"`
	ReceiptNo string `json:"rcept_no" csv:"rcept_no"`
	ReceiptAt string `json:"rcept_dt" csv:"rcept_dt"`
	FilerName string `json:"flr_nm,omitempty" csv:"flr_nm"`
	Remark    string `json:"rm,omitempty" csv:"rm"`
}

type FilingResult struct {
	Provider   provider.ProviderID  `json:"provider"`
	Group      provider.GroupID     `json:"provider_group"`
	Operation  provider.OperationID `json:"operation"`
	CorpCode   string               `json:"corp_code,omitempty"`
	StockCode  string               `json:"stock_code,omitempty"`
	Items      []Filing             `json:"items"`
	TotalCount string               `json:"total_count,omitempty"`
	TotalPage  string               `json:"total_page,omitempty"`
	PageNo     string               `json:"page_no,omitempty"`
	PageCount  string               `json:"page_count,omitempty"`
}

func (p *Provider) FetchFilings(ctx context.Context, req FilingRequest) (FilingResult, error) {
	errb := oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART, "operation", provider.OperationOpenDARTList, "identifier", req.Identifier, "corp_code", req.CorpCode)
	if err := p.requireClient(); err != nil {
		return FilingResult{}, errb.Wrap(err)
	}
	corpCode := strings.TrimSpace(req.CorpCode)
	stockCode := ""
	if corpCode == "" && strings.TrimSpace(req.Identifier) != "" {
		company, err := p.ResolveCompany(ctx, req.Identifier)
		if err != nil {
			return FilingResult{}, errb.Wrap(err)
		}
		corpCode = company.CorpCode
		stockCode = company.StockCode
	}
	from, err := normalizeOpenDARTDate(req.From)
	if err != nil {
		return FilingResult{}, errb.Wrap(err)
	}
	to, err := normalizeOpenDARTDate(req.To)
	if err != nil {
		return FilingResult{}, errb.Wrap(err)
	}
	params := opendartsdk.ListParams{
		CorpCode:  corpCode,
		BgnDe:     from,
		EndDe:     to,
		PageNo:    strings.TrimSpace(req.PageNo),
		PageCount: strings.TrimSpace(req.PageCount),
		Sort:      "date",
		SortMth:   "desc",
	}
	if req.LastReport {
		params.LastReprtAt = "Y"
	}
	response, err := p.client.List(ctx, params)
	if err != nil {
		return FilingResult{}, errb.Wrapf(err, "fetch OpenDART filings")
	}
	if err := ensureOpenDARTStatus(response.Status, response.Message, provider.OperationOpenDARTList); err != nil {
		return FilingResult{}, err
	}
	items := make([]Filing, 0, len(response.List))
	for _, item := range response.List {
		items = append(items, Filing{
			CorpCode:  item.CorpCode,
			CorpName:  item.CorpName,
			StockCode: item.StockCode,
			CorpClass: item.CorpCls,
			Report:    item.ReportNm,
			ReceiptNo: item.RceptNo,
			ReceiptAt: item.RceptDt,
			FilerName: item.FlrNm,
			Remark:    item.Rm,
		})
	}
	return FilingResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTDisclosure,
		Operation:  provider.OperationOpenDARTList,
		CorpCode:   corpCode,
		StockCode:  stockCode,
		Items:      items,
		TotalCount: response.TotalCount,
		TotalPage:  response.TotalPage,
		PageNo:     response.PageNo,
		PageCount:  response.PageCount,
	}, nil
}
