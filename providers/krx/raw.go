package krx

import (
	"context"

	provider "github.com/ev3rlit/mwosa/providers/core"
	"github.com/samber/oops"
)

type RawRequest struct {
	APIID    provider.OperationID
	BaseDate string
}

type RawResult struct {
	Provider  provider.ProviderID  `json:"provider"`
	Group     provider.GroupID     `json:"provider_group"`
	APIID     provider.OperationID `json:"api_id"`
	BaseDate  string               `json:"base_date"`
	Rows      any                  `json:"rows"`
	RowCount  int                  `json:"row_count"`
	Canonical string               `json:"canonical_support"`
}

func (p *Provider) FetchRaw(ctx context.Context, req RawRequest) (RawResult, error) {
	errb := oops.In("krx_adapter").With("provider", provider.ProviderKRX, "api_id", req.APIID, "base_date", req.BaseDate)
	if p == nil {
		return RawResult{}, errb.New("krx provider is nil")
	}
	baseDate := req.BaseDate
	if _, err := parseDate(baseDate); err != nil {
		return RawResult{}, errb.Wrap(err)
	}
	baseDate = mustAPIDate(baseDate)
	if err := p.requireAPI(req.APIID, provider.Role("raw"), "", ""); err != nil {
		return RawResult{}, err
	}
	if p == nil || p.client == nil {
		return RawResult{}, errb.New("krx provider client is nil")
	}

	rows, count, err := p.fetchRawRows(ctx, req.APIID, baseDate)
	if err != nil {
		return RawResult{}, errb.Wrap(err)
	}
	return RawResult{
		Provider:  provider.ProviderKRX,
		Group:     groupForOperation(req.APIID),
		APIID:     req.APIID,
		BaseDate:  formatDate(baseDate),
		Rows:      rows,
		RowCount:  count,
		Canonical: canonicalSupport(req.APIID),
	}, nil
}

func (p *Provider) fetchRawRows(ctx context.Context, apiID provider.OperationID, baseDate string) (any, int, error) {
	switch apiID {
	case provider.OperationKRXDDTrd:
		rows, err := p.client.KRXIndex(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSPIDDTrd:
		rows, err := p.client.KOSPIIndex(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSDAQDDTrd:
		rows, err := p.client.KOSDAQIndex(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationBondDDTrd:
		rows, err := p.client.BondIndex(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationDerivativesDDTrd:
		rows, err := p.client.DerivativesProductIndex(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationStockByddTrd:
		rows, err := p.client.Stock(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSDAQByddTrd:
		rows, err := p.client.KOSDAQStock(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKONEXByddTrd:
		rows, err := p.client.KONEXStock(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationSWByddTrd:
		rows, err := p.client.SubscriptionWarrant(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationSRByddTrd:
		rows, err := p.client.SubscriptionRight(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationStockIssueBaseInfo:
		rows, err := p.client.StockIssueBaseInfo(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSDAQIssueBaseInfo:
		rows, err := p.client.KOSDAQIssueBaseInfo(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKONEXIssueBaseInfo:
		rows, err := p.client.KONEXIssueBaseInfo(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationETFByddTrd:
		rows, err := p.client.ETF(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationETNByddTrd:
		rows, err := p.client.ETN(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationELWByddTrd:
		rows, err := p.client.ELW(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKTSByddTrd:
		rows, err := p.client.KTSBond(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationGeneralBondByddTrd:
		rows, err := p.client.GeneralBond(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationSmallBondByddTrd:
		rows, err := p.client.SmallBond(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationFuturesByddTrd:
		rows, err := p.client.Futures(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSPIStockFutures:
		rows, err := p.client.KOSPIStockFutures(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSDAQStockFutures:
		rows, err := p.client.KOSDAQStockFutures(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationOptionsByddTrd:
		rows, err := p.client.Options(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSPIStockOptions:
		rows, err := p.client.KOSPIStockOptions(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationKOSDAQStockOptions:
		rows, err := p.client.KOSDAQStockOptions(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationOilByddTrd:
		rows, err := p.client.Oil(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationGoldByddTrd:
		rows, err := p.client.Gold(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationETSByddTrd:
		rows, err := p.client.EmissionTradingScheme(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationESGETPInfo:
		rows, err := p.client.ESGETPInfo(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationSRIBondInfo:
		rows, err := p.client.SRIBondInfo(ctx, baseDate)
		return rows, len(rows), err
	case provider.OperationESGIndexInfo:
		rows, err := p.client.ESGIndexInfo(ctx, baseDate)
		return rows, len(rows), err
	default:
		return nil, 0, provider.NewUnsupported(provider.UnsupportedError{
			Capability:  provider.Role("raw"),
			ProviderID:  provider.ProviderKRX,
			OperationID: apiID,
			Reason:      "KRX OPEN API is unsupported",
		})
	}
}

func mustAPIDate(value string) string {
	parsed, err := parseDate(value)
	if err != nil {
		return value
	}
	return parsed.Format("20060102")
}

func canonicalSupport(apiID provider.OperationID) string {
	switch apiID {
	case provider.OperationETFByddTrd, provider.OperationETNByddTrd, provider.OperationELWByddTrd:
		return "daily_bar,instrument"
	case provider.OperationStockByddTrd, provider.OperationKOSDAQByddTrd, provider.OperationKONEXByddTrd:
		return "daily_bar"
	case provider.OperationStockIssueBaseInfo, provider.OperationKOSDAQIssueBaseInfo, provider.OperationKONEXIssueBaseInfo:
		return "instrument"
	default:
		return "raw_only"
	}
}
