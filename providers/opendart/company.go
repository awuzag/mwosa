package opendart

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

type Company struct {
	CorpCode    string `xml:"corp_code" json:"corp_code" csv:"corp_code"`
	CorpName    string `xml:"corp_name" json:"corp_name" csv:"corp_name"`
	CorpEngName string `xml:"corp_eng_name" json:"corp_eng_name" csv:"corp_eng_name"`
	StockCode   string `xml:"stock_code" json:"stock_code,omitempty" csv:"stock_code"`
	ModifyDate  string `xml:"modify_date" json:"modify_date" csv:"modify_date"`
}

type CompanyRegistryResult struct {
	Provider   provider.ProviderID  `json:"provider" csv:"provider"`
	Group      provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation  provider.OperationID `json:"operation" csv:"operation"`
	Companies  []Company            `json:"companies" csv:"-"`
	TotalCount int                  `json:"total_count" csv:"total_count"`
}

func (p *Provider) FetchCompanies(ctx context.Context, listedOnly bool) (CompanyRegistryResult, error) {
	errb := oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART, "operation", provider.OperationOpenDARTCorpCode)
	if err := p.requireClient(); err != nil {
		return CompanyRegistryResult{}, errb.Wrap(err)
	}
	file, err := p.client.CorpCode(ctx)
	if err != nil {
		return CompanyRegistryResult{}, errb.Wrapf(err, "fetch OpenDART corpCode.xml")
	}
	companies, err := DecodeCorpCodeZIP(file.Body)
	if err != nil {
		return CompanyRegistryResult{}, errb.Wrap(err)
	}
	if listedOnly {
		listed := make([]Company, 0, len(companies))
		for _, company := range companies {
			if strings.TrimSpace(company.StockCode) != "" {
				listed = append(listed, company)
			}
		}
		companies = listed
	}
	return CompanyRegistryResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTDisclosure,
		Operation:  provider.OperationOpenDARTCorpCode,
		Companies:  companies,
		TotalCount: len(companies),
	}, nil
}

func (p *Provider) ResolveCompany(ctx context.Context, identifier string) (Company, error) {
	query := strings.TrimSpace(identifier)
	errb := oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART, "identifier", query)
	if query == "" {
		return Company{}, errb.New("OpenDART company resolution requires corp_code or stock_code")
	}
	result, err := p.FetchCompanies(ctx, false)
	if err != nil {
		return Company{}, errb.Wrap(err)
	}
	return resolveCompany(result.Companies, query)
}

func resolveCompany(companies []Company, query string) (Company, error) {
	errb := oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART, "identifier", query)
	matches := make([]Company, 0, 1)
	for _, company := range companies {
		switch {
		case company.CorpCode == query:
			matches = append(matches, company)
		case company.StockCode != "" && company.StockCode == query:
			matches = append(matches, company)
		case strings.EqualFold(company.CorpName, query):
			matches = append(matches, company)
		}
	}
	if len(matches) == 0 {
		return Company{}, errb.New("OpenDART company not found by corp_code, stock_code, or exact company name")
	}
	if len(matches) > 1 {
		return Company{}, errb.With("matches", len(matches)).New("OpenDART company resolution returned multiple matches")
	}
	return matches[0], nil
}

type corpCodeXML struct {
	Companies []Company `xml:"list"`
}

func DecodeCorpCodeZIP(body []byte) ([]Company, error) {
	errb := oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART, "operation", provider.OperationOpenDARTCorpCode, "format", "zip")
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, errb.Wrapf(err, "decode OpenDART corpCode.zip")
	}
	for _, file := range reader.File {
		if !strings.EqualFold(file.Name, "CORPCODE.xml") {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return nil, errb.With("file", file.Name).Wrapf(err, "open corp code XML inside zip")
		}
		defer handle.Close()
		var decoded corpCodeXML
		if err := xml.NewDecoder(handle).Decode(&decoded); err != nil {
			return nil, errb.With("file", file.Name).Wrapf(err, "decode corp code XML")
		}
		return decoded.Companies, nil
	}
	return nil, errb.New("OpenDART corpCode.zip missing CORPCODE.xml")
}
