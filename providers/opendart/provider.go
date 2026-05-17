package opendart

import (
	"context"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/ev3rlit/mwosa/providers/core/financials"
	"github.com/ev3rlit/mwosa/providers/spec"
	opendartsdk "github.com/ev3rlit/opendart"
	"github.com/samber/oops"
)

type client interface {
	CorpCode(context.Context) (*opendartsdk.FileResponse, error)
	List(context.Context, opendartsdk.ListParams) (*opendartsdk.ListResponse, error)
	FnlttSinglAcntAll(context.Context, opendartsdk.FnlttSinglAcntAllParams) (*opendartsdk.FnlttSinglAcntAllResponse, error)
	AlotMatter(context.Context, opendartsdk.AlotMatterParams) (*opendartsdk.AlotMatterResponse, error)
	TesstkAcqsDspsSttus(context.Context, opendartsdk.TesstkAcqsDspsSttusParams) (*opendartsdk.TesstkAcqsDspsSttusResponse, error)
	HyslrSttus(context.Context, opendartsdk.HyslrSttusParams) (*opendartsdk.HyslrSttusResponse, error)
	HyslrChgSttus(context.Context, opendartsdk.HyslrChgSttusParams) (*opendartsdk.HyslrChgSttusResponse, error)
	EmpSttus(context.Context, opendartsdk.EmpSttusParams) (*opendartsdk.EmpSttusResponse, error)
	AccnutAdtorNmNdAdtOpinion(context.Context, opendartsdk.AccnutAdtorNmNdAdtOpinionParams) (*opendartsdk.AccnutAdtorNmNdAdtOpinionResponse, error)
	DfOcr(context.Context, opendartsdk.DfOcrParams) (*opendartsdk.DfOcrResponse, error)
	PiicDecsn(context.Context, opendartsdk.PiicDecsnParams) (*opendartsdk.PiicDecsnResponse, error)
	FricDecsn(context.Context, opendartsdk.FricDecsnParams) (*opendartsdk.FricDecsnResponse, error)
	PifricDecsn(context.Context, opendartsdk.PifricDecsnParams) (*opendartsdk.PifricDecsnResponse, error)
	CrDecsn(context.Context, opendartsdk.CrDecsnParams) (*opendartsdk.CrDecsnResponse, error)
	BnkMngtPcbg(context.Context, opendartsdk.BnkMngtPcbgParams) (*opendartsdk.BnkMngtPcbgResponse, error)
	LwstLg(context.Context, opendartsdk.LwstLgParams) (*opendartsdk.LwstLgResponse, error)
	BsnInhDecsn(context.Context, opendartsdk.BsnInhDecsnParams) (*opendartsdk.BsnInhDecsnResponse, error)
	BsnTrfDecsn(context.Context, opendartsdk.BsnTrfDecsnParams) (*opendartsdk.BsnTrfDecsnResponse, error)
	TgastInhDecsn(context.Context, opendartsdk.TgastInhDecsnParams) (*opendartsdk.TgastInhDecsnResponse, error)
	TgastTrfDecsn(context.Context, opendartsdk.TgastTrfDecsnParams) (*opendartsdk.TgastTrfDecsnResponse, error)
	CvbdIsDecsn(context.Context, opendartsdk.CvbdIsDecsnParams) (*opendartsdk.CvbdIsDecsnResponse, error)
	BdwtIsDecsn(context.Context, opendartsdk.BdwtIsDecsnParams) (*opendartsdk.BdwtIsDecsnResponse, error)
	ExbdIsDecsn(context.Context, opendartsdk.ExbdIsDecsnParams) (*opendartsdk.ExbdIsDecsnResponse, error)
	CmpMgDecsn(context.Context, opendartsdk.CmpMgDecsnParams) (*opendartsdk.CmpMgDecsnResponse, error)
	CmpDvDecsn(context.Context, opendartsdk.CmpDvDecsnParams) (*opendartsdk.CmpDvDecsnResponse, error)
	CmpDvmgDecsn(context.Context, opendartsdk.CmpDvmgDecsnParams) (*opendartsdk.CmpDvmgDecsnResponse, error)
	StkExtrDecsn(context.Context, opendartsdk.StkExtrDecsnParams) (*opendartsdk.StkExtrDecsnResponse, error)
	TsstkAqDecsn(context.Context, opendartsdk.TsstkAqDecsnParams) (*opendartsdk.TsstkAqDecsnResponse, error)
	TsstkDpDecsn(context.Context, opendartsdk.TsstkDpDecsnParams) (*opendartsdk.TsstkDpDecsnResponse, error)
}

type Provider struct {
	provider.Identity

	client     client
	financials financials.Fetch
}

func New(config Config) (*Provider, error) {
	errb := oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART)
	c, err := opendartsdk.New(opendartsdk.Config{APIKey: config.APIKey}, clientOptions(config)...)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	return NewWithClient(c), nil
}

func NewWithClient(client client) *Provider {
	p := &Provider{
		Identity: provider.Identity{
			ID:          provider.ProviderOpenDART,
			DisplayName: "OpenDART",
		},
		client: client,
	}
	p.financials = spec.HistoricalFinancials(p.fetchFinancialStatements).
		Markets(provider.MarketKRX).
		SecurityTypes(provider.SecurityTypeStock).
		Group(provider.GroupOpenDARTFinancials).
		Operations(provider.OperationOpenDARTSinglAcntAll).
		RequiresAuth(provider.CredentialScopeOpenDART).
		Priority(60).
		Limitations(
			"OpenDART is filing-derived financial data, not a price provider",
			"stock_code is resolved to corp_code before OpenDART financial API calls",
		).
		MustBuild()
	return p
}

func Register(registry *provider.Registry, p provider.IdentityProvider) error {
	return registry.RegisterProvider(p)
}

func (p *Provider) RoleRegistrations() []provider.RoleRegistration {
	if p == nil {
		return nil
	}
	return []provider.RoleRegistration{
		{
			Profile: provider.RoleProfile{
				Role:       provider.RoleCompanyRegistry,
				Markets:    []provider.Market{provider.MarketKRX},
				Group:      provider.GroupOpenDARTDisclosure,
				Operations: []provider.OperationID{provider.OperationOpenDARTCorpCode},
				AuthScope:  provider.CredentialScopeOpenDART,
				Freshness:  provider.FreshnessFiling,
				Compatibility: provider.Compatibility{
					DataLatency: provider.DataLatencyHistorical,
					Notes: []string{
						"corpCode.xml maps OpenDART corp_code to stock_code for listed companies",
						"corp_code is not a KRX symbol",
					},
				},
				RequiresAuth: true,
				Priority:     60,
				Limitations: []string{
					"company registry is an identifier reference, not market data",
				},
			},
			Impl: p,
		},
		{
			Profile: provider.RoleProfile{
				Role:       provider.RoleFilings,
				Markets:    []provider.Market{provider.MarketKRX},
				Group:      provider.GroupOpenDARTDisclosure,
				Operations: []provider.OperationID{provider.OperationOpenDARTList},
				AuthScope:  provider.CredentialScopeOpenDART,
				Freshness:  provider.FreshnessFiling,
				Compatibility: provider.Compatibility{
					DataLatency: provider.DataLatencyHistorical,
					Notes: []string{
						"filings are accepted disclosure records, not prices or candles",
					},
				},
				RequiresAuth: true,
				Priority:     60,
			},
			Impl: p,
		},
		p.financials.RoleRegistration(),
	}
}

func (p *Provider) FetchFinancialStatements(ctx context.Context, input financials.FetchInput) (financials.FetchResult, error) {
	return p.fetchFinancialStatements(ctx, input)
}

func (p *Provider) FinancialsProfile() financials.Profile {
	return p.financials.FinancialsProfile()
}

func (p *Provider) RoleRegistration() provider.RoleRegistration {
	return p.financials.RoleRegistration()
}

func (p *Provider) requireClient() error {
	if p == nil {
		return oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART).New("opendart provider is nil")
	}
	if p.client == nil {
		return oops.In("opendart_adapter").With("provider", provider.ProviderOpenDART).New("opendart provider client is nil")
	}
	return nil
}

func normalizeOpenDARTDate(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if len(trimmed) == 8 {
		return trimmed, nil
	}
	if len(trimmed) == 10 && trimmed[4] == '-' && trimmed[7] == '-' {
		return trimmed[:4] + trimmed[5:7] + trimmed[8:10], nil
	}
	return "", oops.In("opendart_adapter").With("date", value).New("OpenDART date must be YYYYMMDD or YYYY-MM-DD")
}

func ensureOpenDARTStatus(status string, message string, operation provider.OperationID) error {
	if status == "" || status == "000" {
		return nil
	}
	return oops.In("opendart_adapter").
		With("provider", provider.ProviderOpenDART, "operation", operation, "status", status).
		Errorf("OpenDART API returned status=%s message=%s", status, message)
}
