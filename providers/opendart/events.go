package opendart

import (
	"context"
	"strconv"
	"strings"

	provider "github.com/ev3rlit/mwosa/providers/core"
	opendartsdk "github.com/ev3rlit/opendart"
	"github.com/samber/oops"
)

const (
	eventTypeConvertibleBondIssuance   = "convertible_bond_issuance"
	eventTypeBondWithWarrantIssuance   = "bond_with_warrant_issuance"
	eventTypeExchangeableBondIssuance  = "exchangeable_bond_issuance"
	eventTypePaidInCapitalIncrease     = "paid_in_capital_increase"
	eventTypeFreeCapitalIncrease       = "free_capital_increase"
	eventTypePaidInFreeCapitalIncrease = "paid_in_free_capital_increase"
	eventTypeCapitalReduction          = "capital_reduction"
	eventTypeDefaultOccurrence         = "default_occurrence"
	eventTypeBankManagementProcedure   = "bank_management_procedure_start"
	eventTypeLawsuitFiling             = "lawsuit_filing"
	eventTypeBusinessTransferIn        = "business_transfer_in"
	eventTypeBusinessTransferOut       = "business_transfer_out"
	eventTypeTangibleAssetTransferIn   = "tangible_asset_transfer_in"
	eventTypeTangibleAssetTransferOut  = "tangible_asset_transfer_out"
	eventTypeCompanyMerger             = "company_merger"
	eventTypeCompanyDivision           = "company_division"
	eventTypeCompanyDivisionMerger     = "company_division_merger"
	eventTypeStockExchangeTransfer     = "stock_exchange_transfer"
	eventTypeTreasuryStockAcquisition  = "treasury_stock_acquisition"
	eventTypeTreasuryStockDisposal     = "treasury_stock_disposal"
)

type EventRequest struct {
	CorpCode string
	From     string
	To       string
}

type CompanyEvent struct {
	EventType   string
	EventDate   string
	RceptDt     string
	RceptNo     string
	Provider    provider.ProviderID
	Group       provider.GroupID
	Operation   provider.OperationID
	Title       string
	AmountMinor *int64
	ValueText   string
	Raw         map[string]any
}

type CompanyEventResult struct {
	Provider   provider.ProviderID  `json:"provider" csv:"provider"`
	Group      provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Operation  provider.OperationID `json:"operation" csv:"operation"`
	Status     string               `json:"status,omitempty" csv:"status"`
	Message    string               `json:"message,omitempty" csv:"message"`
	Events     []CompanyEvent       `json:"events" csv:"-"`
	TotalCount int                  `json:"total_count" csv:"total_count"`
}

type CompanyEventBatchResult struct {
	Provider   provider.ProviderID  `json:"provider" csv:"provider"`
	Group      provider.GroupID     `json:"provider_group" csv:"provider_group"`
	Sources    []CompanyEventResult `json:"sources" csv:"-"`
	Events     []CompanyEvent       `json:"events" csv:"-"`
	TotalCount int                  `json:"total_count" csv:"total_count"`
}

func (p *Provider) FetchMaterialEvents(ctx context.Context, req EventRequest) (CompanyEventBatchResult, error) {
	fetchers := []func(context.Context, EventRequest) (CompanyEventResult, error){
		p.FetchDefaultOccurrenceEvents,
		p.FetchPaidInCapitalIncreaseEvents,
		p.FetchFreeCapitalIncreaseEvents,
		p.FetchPaidInFreeCapitalIncreaseEvents,
		p.FetchCapitalReductionEvents,
		p.FetchBankManagementProcedureEvents,
		p.FetchLawsuitFilingEvents,
		p.FetchBusinessTransferInEvents,
		p.FetchBusinessTransferOutEvents,
		p.FetchTangibleAssetTransferInEvents,
		p.FetchTangibleAssetTransferOutEvents,
		p.FetchConvertibleBondEvents,
		p.FetchBondWithWarrantEvents,
		p.FetchExchangeableBondEvents,
		p.FetchCompanyMergerEvents,
		p.FetchCompanyDivisionEvents,
		p.FetchCompanyDivisionMergerEvents,
		p.FetchStockExchangeTransferEvents,
		p.FetchTreasuryStockAcquisitionEvents,
		p.FetchTreasuryStockDisposalEvents,
	}
	result := CompanyEventBatchResult{
		Provider: provider.ProviderOpenDART,
		Group:    provider.GroupOpenDARTMaterialEvents,
		Sources:  make([]CompanyEventResult, 0, len(fetchers)),
	}
	for _, fetch := range fetchers {
		source, err := fetch(ctx, req)
		if err != nil {
			return CompanyEventBatchResult{}, err
		}
		result.Sources = append(result.Sources, source)
		result.Events = append(result.Events, source.Events...)
	}
	result.TotalCount = len(result.Events)
	return result, nil
}

func validateEventRequest(req EventRequest, operation provider.OperationID) (string, string, string, error) {
	corpCode := strings.TrimSpace(req.CorpCode)
	from, err := normalizeOpenDARTDate(req.From)
	if err != nil {
		return "", "", "", err
	}
	to, err := normalizeOpenDARTDate(req.To)
	if err != nil {
		return "", "", "", err
	}
	errb := oops.In("opendart_adapter").With(
		"provider", provider.ProviderOpenDART,
		"group", provider.GroupOpenDARTMaterialEvents,
		"operation", operation,
		"corp_code", corpCode,
		"from", from,
		"to", to,
	)
	if corpCode == "" {
		return "", "", "", errb.New("OpenDART material events require corp_code")
	}
	if from == "" || to == "" {
		return "", "", "", errb.New("OpenDART material events require --from and --to")
	}
	return corpCode, from, to, nil
}

func (p *Provider) FetchConvertibleBondEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTCvbdIsDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTCvbdIsDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.CvbdIsDecsn(ctx, opendartsdk.CvbdIsDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART convertible bond events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTCvbdIsDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, convertibleBondEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTCvbdIsDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchDefaultOccurrenceEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTDfOcr)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTDfOcr, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.DfOcr(ctx, opendartsdk.DfOcrParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART default occurrence events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTDfOcr); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, defaultOccurrenceEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTDfOcr,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchPaidInCapitalIncreaseEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTPiicDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTPiicDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.PiicDecsn(ctx, opendartsdk.PiicDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART paid-in capital increase events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTPiicDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, paidInCapitalIncreaseEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTPiicDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchFreeCapitalIncreaseEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTFricDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTFricDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.FricDecsn(ctx, opendartsdk.FricDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART free capital increase events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTFricDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, freeCapitalIncreaseEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTFricDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchPaidInFreeCapitalIncreaseEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTPifricDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTPifricDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.PifricDecsn(ctx, opendartsdk.PifricDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART paid-in and free capital increase events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTPifricDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, paidInFreeCapitalIncreaseEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTPifricDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchCapitalReductionEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTCrDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTCrDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.CrDecsn(ctx, opendartsdk.CrDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART capital reduction events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTCrDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, capitalReductionEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTCrDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchBankManagementProcedureEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTBnkMngtPcbg)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTBnkMngtPcbg, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.BnkMngtPcbg(ctx, opendartsdk.BnkMngtPcbgParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART bank management procedure events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTBnkMngtPcbg); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, bankManagementProcedureEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTBnkMngtPcbg,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchLawsuitFilingEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTLwstLg)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTLwstLg, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.LwstLg(ctx, opendartsdk.LwstLgParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART lawsuit filing events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTLwstLg); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, lawsuitFilingEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTLwstLg,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchBusinessTransferInEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTBsnInhDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTBsnInhDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.BsnInhDecsn(ctx, opendartsdk.BsnInhDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART business transfer-in events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTBsnInhDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, businessTransferInEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTBsnInhDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchBusinessTransferOutEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTBsnTrfDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTBsnTrfDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.BsnTrfDecsn(ctx, opendartsdk.BsnTrfDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART business transfer-out events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTBsnTrfDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, businessTransferOutEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTBsnTrfDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchTangibleAssetTransferInEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTTgastInhDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTTgastInhDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.TgastInhDecsn(ctx, opendartsdk.TgastInhDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART tangible asset transfer-in events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTTgastInhDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, tangibleAssetTransferInEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTTgastInhDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchTangibleAssetTransferOutEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTTgastTrfDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTTgastTrfDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.TgastTrfDecsn(ctx, opendartsdk.TgastTrfDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART tangible asset transfer-out events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTTgastTrfDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, tangibleAssetTransferOutEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTTgastTrfDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchBondWithWarrantEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTBdwtIsDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTBdwtIsDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.BdwtIsDecsn(ctx, opendartsdk.BdwtIsDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART bond with warrant events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTBdwtIsDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, bondWithWarrantEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTBdwtIsDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchExchangeableBondEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTExbdIsDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTExbdIsDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.ExbdIsDecsn(ctx, opendartsdk.ExbdIsDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART exchangeable bond events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTExbdIsDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, exchangeableBondEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTExbdIsDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchCompanyMergerEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTCmpMgDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTCmpMgDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.CmpMgDecsn(ctx, opendartsdk.CmpMgDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART company merger events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTCmpMgDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, companyMergerEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTCmpMgDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchCompanyDivisionEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTCmpDvDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTCmpDvDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.CmpDvDecsn(ctx, opendartsdk.CmpDvDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART company division events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTCmpDvDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, companyDivisionEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTCmpDvDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchCompanyDivisionMergerEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTCmpDvmgDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTCmpDvmgDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.CmpDvmgDecsn(ctx, opendartsdk.CmpDvmgDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART company division merger events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTCmpDvmgDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, companyDivisionMergerEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTCmpDvmgDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchStockExchangeTransferEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTStkExtrDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTStkExtrDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.StkExtrDecsn(ctx, opendartsdk.StkExtrDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART stock exchange transfer events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTStkExtrDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, stockExchangeTransferEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTStkExtrDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchTreasuryStockAcquisitionEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTTsstkAqDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTTsstkAqDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.TsstkAqDecsn(ctx, opendartsdk.TsstkAqDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART treasury stock acquisition events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTTsstkAqDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, treasuryStockAcquisitionEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTTsstkAqDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func (p *Provider) FetchTreasuryStockDisposalEvents(ctx context.Context, req EventRequest) (CompanyEventResult, error) {
	corpCode, from, to, err := validateEventRequest(req, provider.OperationOpenDARTTsstkDpDecsn)
	if err != nil {
		return CompanyEventResult{}, err
	}
	errb := materialEventErrorBuilder(provider.OperationOpenDARTTsstkDpDecsn, corpCode, from, to)
	if err := p.requireClient(); err != nil {
		return CompanyEventResult{}, errb.Wrap(err)
	}
	response, err := p.client.TsstkDpDecsn(ctx, opendartsdk.TsstkDpDecsnParams{
		CorpCode: corpCode,
		BgnDe:    from,
		EndDe:    to,
	})
	if err != nil {
		return CompanyEventResult{}, errb.Wrapf(err, "fetch OpenDART treasury stock disposal events")
	}
	if err := ensureOpenDARTEventStatus(response.Status, response.Message, provider.OperationOpenDARTTsstkDpDecsn); err != nil {
		return CompanyEventResult{}, err
	}
	events := make([]CompanyEvent, 0, len(response.List))
	if !isOpenDARTNoDataStatus(response.Status) {
		for _, item := range response.List {
			events = append(events, treasuryStockDisposalEventFromItem(item))
		}
	}
	return CompanyEventResult{
		Provider:   provider.ProviderOpenDART,
		Group:      provider.GroupOpenDARTMaterialEvents,
		Operation:  provider.OperationOpenDARTTsstkDpDecsn,
		Status:     strings.TrimSpace(response.Status),
		Message:    strings.TrimSpace(response.Message),
		Events:     events,
		TotalCount: len(events),
	}, nil
}

func convertibleBondEventFromItem(item opendartsdk.CvbdIsDecsnItem) CompanyEvent {
	raw := rawMap(item)
	return CompanyEvent{
		EventType:   eventTypeConvertibleBondIssuance,
		EventDate:   firstOpenDARTDate(item.Bddd, item.Pymd, item.Sbd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTCvbdIsDecsn,
		Title:       "전환사채권 발행결정",
		AmountMinor: amountMinor(item.BdFta),
		ValueText: eventValueText([]eventValuePart{
			{Label: "권면총액", Value: item.BdFta},
			{Label: "표면이자율", Value: item.BdIntrEx},
			{Label: "만기이자율", Value: item.BdIntrSf},
			{Label: "전환가액", Value: item.CvPrc},
			{Label: "전환비율", Value: item.CvRt},
		}),
		Raw: raw,
	}
}

func defaultOccurrenceEventFromItem(item opendartsdk.DfOcrItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeDefaultOccurrence,
		EventDate:   firstOpenDARTDate(item.Dfd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTDfOcr,
		Title:       "부도발생",
		AmountMinor: amountMinor(item.DfAmt),
		ValueText: eventValueText([]eventValuePart{
			{Label: "부도발생은행", Value: item.DfBnk},
			{Label: "부도내용", Value: item.DfCn},
			{Label: "부도사유", Value: item.DfRs},
			{Label: "최종부도일자", Value: openDARTDateToISO(item.Dfd)},
		}),
		Raw: rawMap(item),
	}
}

func paidInCapitalIncreaseEventFromItem(item opendartsdk.PiicDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypePaidInCapitalIncrease,
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTPiicDecsn,
		Title:       "유상증자 결정",
		AmountMinor: amountMinor(firstNonEmpty(item.FdppBsninh, item.FdppOp, item.FdppFclt, item.FdppOcsa, item.FdppDtrp, item.FdppEtc)),
		ValueText: eventValueText([]eventValuePart{
			{Label: "증자방식", Value: item.IcMthn},
			{Label: "보통주 신주수", Value: item.NstkOstkCnt},
			{Label: "기타주식 신주수", Value: item.NstkEstkCnt},
			{Label: "액면가", Value: item.FvPs},
			{Label: "운영자금", Value: item.FdppOp},
			{Label: "시설자금", Value: item.FdppFclt},
			{Label: "타법인증권취득", Value: item.FdppOcsa},
			{Label: "채무상환", Value: item.FdppDtrp},
			{Label: "공매도시작일", Value: openDARTDateToISO(item.SslBgd)},
			{Label: "공매도종료일", Value: openDARTDateToISO(item.SslEdd)},
		}),
		Raw: rawMap(item),
	}
}

func freeCapitalIncreaseEventFromItem(item opendartsdk.FricDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeFreeCapitalIncrease,
		EventDate: firstOpenDARTDate(item.Bddd, item.NstkAsstd),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTFricDecsn,
		Title:     "무상증자 결정",
		ValueText: eventValueText([]eventValuePart{
			{Label: "보통주 신주수", Value: item.NstkOstkCnt},
			{Label: "기타주식 신주수", Value: item.NstkEstkCnt},
			{Label: "보통주 1주당 배정", Value: item.NstkAscntPsOstk},
			{Label: "기타주식 1주당 배정", Value: item.NstkAscntPsEstk},
			{Label: "신주배정기준일", Value: openDARTDateToISO(item.NstkAsstd)},
			{Label: "상장예정일", Value: openDARTDateToISO(item.NstkLstprd)},
		}),
		Raw: rawMap(item),
	}
}

func paidInFreeCapitalIncreaseEventFromItem(item opendartsdk.PifricDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypePaidInFreeCapitalIncrease,
		EventDate:   firstOpenDARTDate(item.FricBddd, item.FricNstkAsstd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTPifricDecsn,
		Title:       "유무상증자 결정",
		AmountMinor: amountMinor(firstNonEmpty(item.PiicFdppBsninh, item.PiicFdppOp, item.PiicFdppFclt, item.PiicFdppOcsa, item.PiicFdppDtrp, item.PiicFdppEtc)),
		ValueText: eventValueText([]eventValuePart{
			{Label: "유상증자방식", Value: item.PiicIcMthn},
			{Label: "유상 보통주 신주수", Value: item.PiicNstkOstkCnt},
			{Label: "유상 기타주식 신주수", Value: item.PiicNstkEstkCnt},
			{Label: "무상 보통주 신주수", Value: item.FricNstkOstkCnt},
			{Label: "무상 기타주식 신주수", Value: item.FricNstkEstkCnt},
			{Label: "무상 신주배정기준일", Value: openDARTDateToISO(item.FricNstkAsstd)},
			{Label: "무상 상장예정일", Value: openDARTDateToISO(item.FricNstkLstprd)},
		}),
		Raw: rawMap(item),
	}
}

func capitalReductionEventFromItem(item opendartsdk.CrDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeCapitalReduction,
		EventDate:   firstOpenDARTDate(item.Bddd, item.CrStd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTCrDecsn,
		Title:       "감자 결정",
		AmountMinor: amountDifferenceMinor(item.BfcrCpt, item.AtcrCpt),
		ValueText: eventValueText([]eventValuePart{
			{Label: "감자방법", Value: item.CrMth},
			{Label: "감자사유", Value: item.CrRs},
			{Label: "보통주 감자주식수", Value: item.CrstkOstkCnt},
			{Label: "기타주식 감자주식수", Value: item.CrstkEstkCnt},
			{Label: "보통주 감자비율", Value: item.CrRtOstk},
			{Label: "기타주식 감자비율", Value: item.CrRtEstk},
			{Label: "감자기준일", Value: openDARTDateToISO(item.CrStd)},
			{Label: "감자전 자본금", Value: item.BfcrCpt},
			{Label: "감자후 자본금", Value: item.AtcrCpt},
		}),
		Raw: rawMap(item),
	}
}

func bankManagementProcedureEventFromItem(item opendartsdk.BnkMngtPcbgItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeBankManagementProcedure,
		EventDate: firstOpenDARTDate(item.MngtPcbgDd, item.Cfd),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTBnkMngtPcbg,
		Title:     "채권은행 등의 관리절차 개시",
		ValueText: eventValueText([]eventValuePart{
			{Label: "관리기관", Value: item.MngtInt},
			{Label: "관리사유", Value: item.MngtRs},
			{Label: "관리기간", Value: item.MngtPd},
			{Label: "개시결정일자", Value: openDARTDateToISO(item.MngtPcbgDd)},
			{Label: "확인일자", Value: openDARTDateToISO(item.Cfd)},
		}),
		Raw: rawMap(item),
	}
}

func lawsuitFilingEventFromItem(item opendartsdk.LwstLgItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeLawsuitFiling,
		EventDate: firstOpenDARTDate(item.Lgd, item.Cfd),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTLwstLg,
		Title:     firstNonEmpty(item.Icnm, "소송 등의 제기"),
		ValueText: eventValueText([]eventValuePart{
			{Label: "사건명", Value: item.Icnm},
			{Label: "원고/신청인", Value: item.AcAp},
			{Label: "관할법원", Value: item.Cpct},
			{Label: "제기일자", Value: openDARTDateToISO(item.Lgd)},
			{Label: "청구내용", Value: item.RqCn},
			{Label: "향후대책", Value: item.FtCtp},
		}),
		Raw: rawMap(item),
	}
}

func businessTransferInEventFromItem(item opendartsdk.BsnInhDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeBusinessTransferIn,
		EventDate:   firstOpenDARTDate(item.Bddd, item.InhPrdInhStd, item.InhPrdCtrCnsd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTBsnInhDecsn,
		Title:       "영업양수 결정",
		AmountMinor: amountMinor(item.InhPrc),
		ValueText: eventValueText([]eventValuePart{
			{Label: "양수영업", Value: item.InhBsn},
			{Label: "양수영업 주요내용", Value: item.InhBsnMc},
			{Label: "거래상대방", Value: item.DlptnCmpnm},
			{Label: "양수목적", Value: item.InhPp},
			{Label: "양수금액", Value: item.InhPrc},
			{Label: "계약체결일", Value: openDARTDateToISO(item.InhPrdCtrCnsd)},
			{Label: "양수기준일", Value: openDARTDateToISO(item.InhPrdInhStd)},
			{Label: "양수영향", Value: item.InhAf},
		}),
		Raw: rawMap(item),
	}
}

func businessTransferOutEventFromItem(item opendartsdk.BsnTrfDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeBusinessTransferOut,
		EventDate:   firstOpenDARTDate(item.Bddd, item.TrfPrdTrfStd, item.TrfPrdCtrCnsd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTBsnTrfDecsn,
		Title:       "영업양도 결정",
		AmountMinor: amountMinor(item.TrfPrc),
		ValueText: eventValueText([]eventValuePart{
			{Label: "양도영업", Value: item.TrfBsn},
			{Label: "양도영업 주요내용", Value: item.TrfBsnMc},
			{Label: "거래상대방", Value: item.DlptnCmpnm},
			{Label: "양도목적", Value: item.TrfPp},
			{Label: "양도금액", Value: item.TrfPrc},
			{Label: "계약체결일", Value: openDARTDateToISO(item.TrfPrdCtrCnsd)},
			{Label: "양도기준일", Value: openDARTDateToISO(item.TrfPrdTrfStd)},
			{Label: "양도영향", Value: item.TrfAf},
		}),
		Raw: rawMap(item),
	}
}

func tangibleAssetTransferInEventFromItem(item opendartsdk.TgastInhDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeTangibleAssetTransferIn,
		EventDate:   firstOpenDARTDate(item.Bddd, item.InhPrdInhStd, item.InhPrdCtrCnsd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTTgastInhDecsn,
		Title:       "유형자산 양수 결정",
		AmountMinor: amountMinor(item.InhdtlInhprc),
		ValueText: eventValueText([]eventValuePart{
			{Label: "자산명", Value: item.AstNm},
			{Label: "자산구분", Value: item.AstSen},
			{Label: "거래상대방", Value: item.DlptnCmpnm},
			{Label: "양수목적", Value: item.InhPp},
			{Label: "양수금액", Value: item.InhdtlInhprc},
			{Label: "자산총액대비", Value: item.InhdtlTastVs},
			{Label: "계약체결일", Value: openDARTDateToISO(item.InhPrdCtrCnsd)},
			{Label: "양수기준일", Value: openDARTDateToISO(item.InhPrdInhStd)},
		}),
		Raw: rawMap(item),
	}
}

func tangibleAssetTransferOutEventFromItem(item opendartsdk.TgastTrfDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeTangibleAssetTransferOut,
		EventDate:   firstOpenDARTDate(item.Bddd, item.TrfPrdTrfStd, item.TrfPrdCtrCnsd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTTgastTrfDecsn,
		Title:       "유형자산 양도 결정",
		AmountMinor: amountMinor(item.TrfdtlTrfprc),
		ValueText: eventValueText([]eventValuePart{
			{Label: "자산명", Value: item.AstNm},
			{Label: "자산구분", Value: item.AstSen},
			{Label: "거래상대방", Value: item.DlptnCmpnm},
			{Label: "양도목적", Value: item.TrfPp},
			{Label: "양도금액", Value: item.TrfdtlTrfprc},
			{Label: "자산총액대비", Value: item.TrfdtlTastVs},
			{Label: "계약체결일", Value: openDARTDateToISO(item.TrfPrdCtrCnsd)},
			{Label: "양도기준일", Value: openDARTDateToISO(item.TrfPrdTrfStd)},
		}),
		Raw: rawMap(item),
	}
}

func bondWithWarrantEventFromItem(item opendartsdk.BdwtIsDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeBondWithWarrantIssuance,
		EventDate:   firstOpenDARTDate(item.Bddd, item.Pymd, item.Sbd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTBdwtIsDecsn,
		Title:       "신주인수권부사채권 발행결정",
		AmountMinor: amountMinor(item.BdFta),
		ValueText: eventValueText([]eventValuePart{
			{Label: "권면총액", Value: item.BdFta},
			{Label: "표면이자율", Value: item.BdIntrEx},
			{Label: "만기이자율", Value: item.BdIntrSf},
			{Label: "행사가액", Value: item.ExPrc},
			{Label: "행사비율", Value: item.ExRt},
			{Label: "분리여부", Value: item.BdwtDivAtn},
		}),
		Raw: rawMap(item),
	}
}

func exchangeableBondEventFromItem(item opendartsdk.ExbdIsDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeExchangeableBondIssuance,
		EventDate:   firstOpenDARTDate(item.Bddd, item.Pymd, item.Sbd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTExbdIsDecsn,
		Title:       "교환사채권 발행결정",
		AmountMinor: amountMinor(item.BdFta),
		ValueText: eventValueText([]eventValuePart{
			{Label: "권면총액", Value: item.BdFta},
			{Label: "표면이자율", Value: item.BdIntrEx},
			{Label: "만기이자율", Value: item.BdIntrSf},
			{Label: "교환가액", Value: item.ExPrc},
			{Label: "교환비율", Value: item.ExRt},
			{Label: "교환대상", Value: item.Extg},
			{Label: "교환대상주식수", Value: item.ExtgStkcnt},
		}),
		Raw: rawMap(item),
	}
}

func companyMergerEventFromItem(item opendartsdk.CmpMgDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeCompanyMerger,
		EventDate: firstOpenDARTDate(item.Bddd, item.MgscMgdt, item.MgscMgctrd),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTCmpMgDecsn,
		Title:     "회사합병 결정",
		ValueText: eventValueText([]eventValuePart{
			{Label: "합병상대회사", Value: item.MgptncmpCmpnm},
			{Label: "합병방법", Value: item.MgMth},
			{Label: "합병형태", Value: item.MgStn},
			{Label: "합병목적", Value: item.MgPp},
			{Label: "합병비율", Value: item.MgRt},
			{Label: "합병계약일", Value: openDARTDateToISO(item.MgscMgctrd)},
			{Label: "합병기일", Value: openDARTDateToISO(item.MgscMgdt)},
			{Label: "신설합병회사", Value: item.NmgcmpCmpnm},
		}),
		Raw: rawMap(item),
	}
}

func companyDivisionEventFromItem(item opendartsdk.CmpDvDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeCompanyDivision,
		EventDate: firstOpenDARTDate(item.Bddd, item.Dvdt),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTCmpDvDecsn,
		Title:     "회사분할 결정",
		ValueText: eventValueText([]eventValuePart{
			{Label: "분할방법", Value: item.DvMth},
			{Label: "분할비율", Value: item.DvRt},
			{Label: "분할설립회사", Value: item.DvfcmpCmpnm},
			{Label: "분할 후 존속회사", Value: item.AtdvExcmpCmpnm},
			{Label: "이전 사업/재산", Value: item.DvTrfbsnprtCn},
			{Label: "중요영향", Value: item.DvImpef},
			{Label: "분할기일", Value: openDARTDateToISO(item.Dvdt)},
			{Label: "분할등기예정일", Value: openDARTDateToISO(item.Dvrgsprd)},
		}),
		Raw: rawMap(item),
	}
}

func companyDivisionMergerEventFromItem(item opendartsdk.CmpDvmgDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeCompanyDivisionMerger,
		EventDate: firstOpenDARTDate(item.Bddd, item.DvmgscDvmgdt, item.DvmgscDvmgctrd),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTCmpDvmgDecsn,
		Title:     "회사분할합병 결정",
		ValueText: eventValueText([]eventValuePart{
			{Label: "분할합병방법", Value: item.DvmgMth},
			{Label: "분할합병비율", Value: item.DvmgRt},
			{Label: "합병상대회사", Value: item.MgptncmpCmpnm},
			{Label: "분할설립회사", Value: item.DvfcmpCmpnm},
			{Label: "분할 후 존속회사", Value: item.AtdvExcmpCmpnm},
			{Label: "분할합병 영향", Value: item.DvmgImpef},
			{Label: "분할합병계약일", Value: openDARTDateToISO(item.DvmgscDvmgctrd)},
			{Label: "분할합병기일", Value: openDARTDateToISO(item.DvmgscDvmgdt)},
		}),
		Raw: rawMap(item),
	}
}

func stockExchangeTransferEventFromItem(item opendartsdk.StkExtrDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType: eventTypeStockExchangeTransfer,
		EventDate: firstOpenDARTDate(item.Bddd, item.ExtrscExtrdt, item.ExtrscExtrctrd),
		RceptNo:   strings.TrimSpace(item.RceptNo),
		Provider:  provider.ProviderOpenDART,
		Group:     provider.GroupOpenDARTMaterialEvents,
		Operation: provider.OperationOpenDARTStkExtrDecsn,
		Title:     "주식교환ㆍ이전 결정",
		ValueText: eventValueText([]eventValuePart{
			{Label: "구분", Value: item.ExtrSen},
			{Label: "교환ㆍ이전 대상법인", Value: item.ExtrTgcmpCmpnm},
			{Label: "완전모회사", Value: item.AtextrCpcmpnm},
			{Label: "교환ㆍ이전 형태", Value: item.ExtrStn},
			{Label: "교환ㆍ이전 목적", Value: item.ExtrPp},
			{Label: "교환ㆍ이전 비율", Value: item.ExtrRt},
			{Label: "교환ㆍ이전 계약일", Value: openDARTDateToISO(item.ExtrscExtrctrd)},
			{Label: "교환ㆍ이전 일자", Value: openDARTDateToISO(item.ExtrscExtrdt)},
		}),
		Raw: rawMap(item),
	}
}

func treasuryStockAcquisitionEventFromItem(item opendartsdk.TsstkAqDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeTreasuryStockAcquisition,
		EventDate:   firstOpenDARTDate(item.AqDd, item.AqexpdBgd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTTsstkAqDecsn,
		Title:       "자기주식 취득 결정",
		AmountMinor: amountMinor(firstNonEmpty(item.AqplnPrcOstk, item.AqplnPrcEstk)),
		ValueText: eventValueText([]eventValuePart{
			{Label: "취득목적", Value: item.AqPp},
			{Label: "취득방법", Value: item.AqMth},
			{Label: "보통주 취득예정금액", Value: item.AqplnPrcOstk},
			{Label: "기타주식 취득예정금액", Value: item.AqplnPrcEstk},
			{Label: "보통주 취득예정주식수", Value: item.AqplnStkOstk},
			{Label: "기타주식 취득예정주식수", Value: item.AqplnStkEstk},
			{Label: "취득예상시작일", Value: openDARTDateToISO(item.AqexpdBgd)},
			{Label: "취득예상종료일", Value: openDARTDateToISO(item.AqexpdEdd)},
		}),
		Raw: rawMap(item),
	}
}

func treasuryStockDisposalEventFromItem(item opendartsdk.TsstkDpDecsnItem) CompanyEvent {
	return CompanyEvent{
		EventType:   eventTypeTreasuryStockDisposal,
		EventDate:   firstOpenDARTDate(item.DpDd, item.DpprpdBgd),
		RceptNo:     strings.TrimSpace(item.RceptNo),
		Provider:    provider.ProviderOpenDART,
		Group:       provider.GroupOpenDARTMaterialEvents,
		Operation:   provider.OperationOpenDARTTsstkDpDecsn,
		Title:       "자기주식 처분 결정",
		AmountMinor: amountMinor(firstNonEmpty(item.DpplnPrcOstk, item.DpplnPrcEstk)),
		ValueText: eventValueText([]eventValuePart{
			{Label: "처분목적", Value: item.DpPp},
			{Label: "보통주 처분예정금액", Value: item.DpplnPrcOstk},
			{Label: "기타주식 처분예정금액", Value: item.DpplnPrcEstk},
			{Label: "보통주 처분예정주식수", Value: item.DpplnStkOstk},
			{Label: "기타주식 처분예정주식수", Value: item.DpplnStkEstk},
			{Label: "처분예정시작일", Value: openDARTDateToISO(item.DpprpdBgd)},
			{Label: "처분예정종료일", Value: openDARTDateToISO(item.DpprpdEdd)},
		}),
		Raw: rawMap(item),
	}
}

func materialEventErrorBuilder(operation provider.OperationID, corpCode string, from string, to string) oops.OopsErrorBuilder {
	return oops.In("opendart_adapter").With(
		"provider", provider.ProviderOpenDART,
		"group", provider.GroupOpenDARTMaterialEvents,
		"operation", operation,
		"corp_code", corpCode,
		"from", from,
		"to", to,
	)
}

func ensureOpenDARTEventStatus(status string, message string, operation provider.OperationID) error {
	if isOpenDARTNoDataStatus(status) {
		return nil
	}
	return ensureOpenDARTStatus(status, message, operation)
}

func isOpenDARTNoDataStatus(status string) bool {
	return strings.TrimSpace(status) == "013"
}

type eventValuePart struct {
	Label string
	Value string
}

func eventValueText(parts []eventValuePart) string {
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part.Value)
		if value == "" || value == "-" {
			continue
		}
		values = append(values, part.Label+"="+value)
	}
	return strings.Join(values, ", ")
}

func amountMinor(value string) *int64 {
	normalized := numericText(value)
	if normalized == "" {
		return nil
	}
	parsed, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func amountDifferenceMinor(before string, after string) *int64 {
	beforeAmount := amountMinor(before)
	afterAmount := amountMinor(after)
	if beforeAmount == nil || afterAmount == nil {
		return nil
	}
	diff := *beforeAmount - *afterAmount
	if diff < 0 {
		diff = -diff
	}
	return &diff
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" && strings.TrimSpace(value) != "-" {
			return value
		}
	}
	return ""
}

func firstOpenDARTDate(values ...string) string {
	for _, value := range values {
		normalized := openDARTDateToISO(value)
		if normalized != "" {
			return normalized
		}
	}
	return ""
}

func openDARTDateToISO(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "-" {
		return ""
	}
	normalized := strings.ReplaceAll(trimmed, ".", "-")
	normalized = strings.ReplaceAll(normalized, "/", "-")
	if len(normalized) == 8 && normalized[4] != '-' {
		return normalized[:4] + "-" + normalized[4:6] + "-" + normalized[6:8]
	}
	if len(normalized) == 10 && normalized[4] == '-' && normalized[7] == '-' {
		return normalized
	}
	return trimmed
}
