package krx

import provider "github.com/awuzag/mwosa/providers/core"

type Service struct {
	Category    string               `json:"category"`
	Group       provider.GroupID     `json:"group"`
	Operation   provider.OperationID `json:"api_id"`
	Description string               `json:"description"`
	RawOnly     bool                 `json:"raw_only"`
}

func ServiceCatalog() []Service {
	return []Service{
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationKRXDDTrd, Description: "KRX index daily trade rows"},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationKOSPIDDTrd, Description: "KOSPI index daily trade rows"},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationKOSDAQDDTrd, Description: "KOSDAQ index daily trade rows"},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationBondDDTrd, Description: "Bond index daily trade rows", RawOnly: true},
		{Category: "index", Group: provider.GroupKRXIndexDailyTrade, Operation: provider.OperationDerivativesDDTrd, Description: "Derivatives product index daily trade rows"},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationStockByddTrd, Description: "KOSPI stock daily trade rows"},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationKOSDAQByddTrd, Description: "KOSDAQ stock daily trade rows"},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationKONEXByddTrd, Description: "KONEX stock daily trade rows"},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationSWByddTrd, Description: "Subscription warrant daily trade rows", RawOnly: true},
		{Category: "stock", Group: provider.GroupKRXStockDailyTrade, Operation: provider.OperationSRByddTrd, Description: "Subscription right daily trade rows", RawOnly: true},
		{Category: "stock", Group: provider.GroupKRXStockInstrument, Operation: provider.OperationStockIssueBaseInfo, Description: "KOSPI listed issue base information"},
		{Category: "stock", Group: provider.GroupKRXStockInstrument, Operation: provider.OperationKOSDAQIssueBaseInfo, Description: "KOSDAQ listed issue base information"},
		{Category: "stock", Group: provider.GroupKRXStockInstrument, Operation: provider.OperationKONEXIssueBaseInfo, Description: "KONEX listed issue base information"},
		{Category: "etp", Group: provider.GroupKRXETPDailyTrade, Operation: provider.OperationETFByddTrd, Description: "ETF daily trade rows"},
		{Category: "etp", Group: provider.GroupKRXETPDailyTrade, Operation: provider.OperationETNByddTrd, Description: "ETN daily trade rows"},
		{Category: "etp", Group: provider.GroupKRXETPDailyTrade, Operation: provider.OperationELWByddTrd, Description: "ELW daily trade rows"},
		{Category: "bond", Group: provider.GroupKRXBondDailyTrade, Operation: provider.OperationKTSByddTrd, Description: "KTS bond daily trade rows", RawOnly: true},
		{Category: "bond", Group: provider.GroupKRXBondDailyTrade, Operation: provider.OperationGeneralBondByddTrd, Description: "General bond daily trade rows", RawOnly: true},
		{Category: "bond", Group: provider.GroupKRXBondDailyTrade, Operation: provider.OperationSmallBondByddTrd, Description: "Small bond daily trade rows", RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationFuturesByddTrd, Description: "Futures daily trade rows", RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSPIStockFutures, Description: "KOSPI stock futures daily trade rows", RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSDAQStockFutures, Description: "KOSDAQ stock futures daily trade rows", RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationOptionsByddTrd, Description: "Options daily trade rows", RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSPIStockOptions, Description: "KOSPI stock options daily trade rows", RawOnly: true},
		{Category: "derivatives", Group: provider.GroupKRXDerivativesDailyTrade, Operation: provider.OperationKOSDAQStockOptions, Description: "KOSDAQ stock options daily trade rows", RawOnly: true},
		{Category: "commodity", Group: provider.GroupKRXCommodityDailyTrade, Operation: provider.OperationOilByddTrd, Description: "Oil market daily trade rows", RawOnly: true},
		{Category: "commodity", Group: provider.GroupKRXCommodityDailyTrade, Operation: provider.OperationGoldByddTrd, Description: "Gold market daily trade rows", RawOnly: true},
		{Category: "commodity", Group: provider.GroupKRXCommodityDailyTrade, Operation: provider.OperationETSByddTrd, Description: "Emission trading scheme daily trade rows", RawOnly: true},
		{Category: "esg", Group: provider.GroupKRXESGReference, Operation: provider.OperationESGETPInfo, Description: "ESG ETP reference information", RawOnly: true},
		{Category: "esg", Group: provider.GroupKRXESGReference, Operation: provider.OperationSRIBondInfo, Description: "SRI bond reference information", RawOnly: true},
		{Category: "esg", Group: provider.GroupKRXESGReference, Operation: provider.OperationESGIndexInfo, Description: "ESG index reference information", RawOnly: true},
	}
}
