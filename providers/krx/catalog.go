package krx

import provider "github.com/ev3rlit/mwosa/providers/core"

type Service struct {
	Category  string               `json:"category"`
	Group     provider.GroupID     `json:"group"`
	Operation provider.OperationID `json:"api_id"`
	RawOnly   bool                 `json:"raw_only"`
}

func ServiceCatalog() []Service {
	return []Service{
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationKRXDDTrd, RawOnly: true},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationKOSPIDDTrd, RawOnly: true},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationKOSDAQDDTrd, RawOnly: true},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationBondDDTrd, RawOnly: true},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationDerivativesDDTrd, RawOnly: true},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationStockByddTrd},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationKOSDAQByddTrd},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationKONEXByddTrd},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationSWByddTrd, RawOnly: true},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationSRByddTrd, RawOnly: true},
		{Category: "stock", Group: provider.GroupKRXStockInstrument, Operation: provider.OperationStockIssueBaseInfo},
		{Category: "stock", Group: provider.GroupKRXStockInstrument, Operation: provider.OperationKOSDAQIssueBaseInfo},
		{Category: "stock", Group: provider.GroupKRXStockInstrument, Operation: provider.OperationKONEXIssueBaseInfo},
		{Category: "etp", Group: provider.GroupKRXETPDailyTrade, Operation: provider.OperationETFByddTrd},
		{Category: "etp", Group: provider.GroupKRXETPDailyTrade, Operation: provider.OperationETNByddTrd},
		{Category: "etp", Group: provider.GroupKRXETPDailyTrade, Operation: provider.OperationELWByddTrd},
		{Category: "bond", Group: provider.GroupKRXBondDailyTrade, Operation: provider.OperationKTSByddTrd, RawOnly: true},
		{Category: "bond", Group: provider.GroupKRXBondDailyTrade, Operation: provider.OperationGeneralBondByddTrd, RawOnly: true},
		{Category: "bond", Group: provider.GroupKRXBondDailyTrade, Operation: provider.OperationSmallBondByddTrd, RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationFuturesByddTrd, RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSPIStockFutures, RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSDAQStockFutures, RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationOptionsByddTrd, RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSPIStockOptions, RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSDAQStockOptions, RawOnly: true},
		{Category: "commodity", Group: provider.GroupKRXCommodityDailyTrade, Operation: provider.OperationOilByddTrd, RawOnly: true},
		{Category: "commodity", Group: provider.GroupKRXCommodityDailyTrade, Operation: provider.OperationGoldByddTrd, RawOnly: true},
		{Category: "commodity", Group: provider.GroupKRXCommodityDailyTrade, Operation: provider.OperationETSByddTrd, RawOnly: true},
		{Category: "esg", Group: provider.GroupKRXESGReference, Operation: provider.OperationESGETPInfo, RawOnly: true},
		{Category: "esg", Group: provider.GroupKRXESGReference, Operation: provider.OperationSRIBondInfo, RawOnly: true},
		{Category: "esg", Group: provider.GroupKRXESGReference, Operation: provider.OperationESGIndexInfo, RawOnly: true},
	}
}
