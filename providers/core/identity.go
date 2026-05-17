package core

type ProviderID string
type GroupID string
type OperationID string
type Market string
type SecurityType string
type CredentialScope string
type Freshness string
type DataLatency string
type Role string

const (
	ProviderKIS                    ProviderID = "kis"
	ProviderDataGo                 ProviderID = "datago"
	ProviderDataGoCorporateFinance ProviderID = "datago-corpfin"
	ProviderKRX                    ProviderID = "krx"

	GroupKISDomesticStockQuotation  GroupID = "domesticStockQuotation"
	GroupKISDomesticStockInstrument GroupID = "domesticStockInstrument"
	GroupSecuritiesProductPrice     GroupID = "securitiesProductPrice"
	GroupStockPrice                 GroupID = "stockPrice"
	GroupCorporateFinance           GroupID = "corporateFinance"
	GroupKRXListedInfo              GroupID = "krxListedInfo"
	GroupKRXIndexDailyTrade         GroupID = "indexDailyTrade"
	GroupKRXStockDailyTrade         GroupID = "stockDailyTrade"
	GroupKRXStockInstrument         GroupID = "stockInstrument"
	GroupKRXETPDailyTrade           GroupID = "etpDailyTrade"
	GroupKRXBondDailyTrade          GroupID = "bondDailyTrade"
	GroupKRXDerivativesDailyTrade   GroupID = "derivativesDailyTrade"
	GroupKRXCommodityDailyTrade     GroupID = "commodityDailyTrade"
	GroupKRXESGReference            GroupID = "esgReference"

	OperationKISPrice                  OperationID = "price"
	OperationKISDaily                  OperationID = "daily"
	OperationKISIntraday               OperationID = "intraday"
	OperationKISOrderbook              OperationID = "orderbook"
	OperationKISTrades                 OperationID = "trades"
	OperationKISTimeTrades             OperationID = "timeTrades"
	OperationKISProduct                OperationID = "product"
	OperationKISStock                  OperationID = "stock"
	OperationKISETFETNPrice            OperationID = "etfetnPrice"
	OperationKISETFComponentStockPrice OperationID = "etfComponentStockPrice"
	OperationGetETFPriceInfo           OperationID = "getETFPriceInfo"
	OperationGetETNPriceInfo           OperationID = "getETNPriceInfo"
	OperationGetELWPriceInfo           OperationID = "getELWPriceInfo"
	OperationGetStockPriceInfo         OperationID = "getStockPriceInfo"
	OperationGetSummFinaStatV2         OperationID = "getSummFinaStat_V2"
	OperationGetBalanceSheetV2         OperationID = "getBs_V2"
	OperationGetIncomeStatementV2      OperationID = "getIncoStat_V2"
	OperationGetItemInfo               OperationID = "getItemInfo"
	OperationKRXDDTrd                  OperationID = "krx_dd_trd"
	OperationKOSPIDDTrd                OperationID = "kospi_dd_trd"
	OperationKOSDAQDDTrd               OperationID = "kosdaq_dd_trd"
	OperationBondDDTrd                 OperationID = "bon_dd_trd"
	OperationDerivativesDDTrd          OperationID = "drvprod_dd_trd"
	OperationStockByddTrd              OperationID = "stk_bydd_trd"
	OperationKOSDAQByddTrd             OperationID = "ksq_bydd_trd"
	OperationKONEXByddTrd              OperationID = "knx_bydd_trd"
	OperationSWByddTrd                 OperationID = "sw_bydd_trd"
	OperationSRByddTrd                 OperationID = "sr_bydd_trd"
	OperationStockIssueBaseInfo        OperationID = "stk_isu_base_info"
	OperationKOSDAQIssueBaseInfo       OperationID = "ksq_isu_base_info"
	OperationKONEXIssueBaseInfo        OperationID = "knx_isu_base_info"
	OperationETFByddTrd                OperationID = "etf_bydd_trd"
	OperationETNByddTrd                OperationID = "etn_bydd_trd"
	OperationELWByddTrd                OperationID = "elw_bydd_trd"
	OperationKTSByddTrd                OperationID = "kts_bydd_trd"
	OperationGeneralBondByddTrd        OperationID = "bnd_bydd_trd"
	OperationSmallBondByddTrd          OperationID = "smb_bydd_trd"
	OperationFuturesByddTrd            OperationID = "fut_bydd_trd"
	OperationKOSPIStockFutures         OperationID = "eqsfu_stk_bydd_trd"
	OperationKOSDAQStockFutures        OperationID = "eqkfu_ksq_bydd_trd"
	OperationOptionsByddTrd            OperationID = "opt_bydd_trd"
	OperationKOSPIStockOptions         OperationID = "eqsop_bydd_trd"
	OperationKOSDAQStockOptions        OperationID = "eqkop_bydd_trd"
	OperationOilByddTrd                OperationID = "oil_bydd_trd"
	OperationGoldByddTrd               OperationID = "gold_bydd_trd"
	OperationETSByddTrd                OperationID = "ets_bydd_trd"
	OperationESGETPInfo                OperationID = "esg_etp_info"
	OperationSRIBondInfo               OperationID = "sri_bond_info"
	OperationESGIndexInfo              OperationID = "esg_index_info"

	MarketKRX Market = "krx"

	SecurityTypeETF   SecurityType = "etf"
	SecurityTypeETN   SecurityType = "etn"
	SecurityTypeELW   SecurityType = "elw"
	SecurityTypeStock SecurityType = "stock"

	CredentialScopeKIS    CredentialScope = "kis"
	CredentialScopeDataGo CredentialScope = "datago"
	CredentialScopeKRX    CredentialScope = "krx"

	FreshnessDaily    Freshness = "daily"
	FreshnessFiling   Freshness = "filing"
	FreshnessIntraday Freshness = "intraday"

	DataLatencyRealtime            DataLatency = "realtime"
	DataLatencyEndOfDay            DataLatency = "end_of_day"
	DataLatencyPreviousBusinessDay DataLatency = "previous_business_day"
	DataLatencyHistorical          DataLatency = "historical"

	RoleDailyBar    Role = "daily_bar"
	RoleFinancials  Role = "financials"
	RoleInstrument  Role = "instrument"
	RoleIntradayBar Role = "intraday_bar"
	RoleIndexBar    Role = "index_bar"
	RoleOrderbook   Role = "orderbook"
	RoleQuote       Role = "quote_snapshot"
	RoleComposition Role = "composition"
	RoleTrades      Role = "trades"
)

type Compatibility struct {
	DataLatency         DataLatency
	LagBusinessDays     int
	CurrentDaySupported bool
	Notes               []string
}

type Identity struct {
	ID          ProviderID `json:"id"`
	DisplayName string     `json:"display_name"`
}

func (i Identity) ProviderIdentity() Identity {
	return i
}

type IdentityProvider interface {
	ProviderIdentity() Identity
}
