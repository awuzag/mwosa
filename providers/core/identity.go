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

	GroupKISDomesticStockQuotation  GroupID = "domesticStockQuotation"
	GroupKISDomesticStockInstrument GroupID = "domesticStockInstrument"
	GroupSecuritiesProductPrice     GroupID = "securitiesProductPrice"
	GroupStockPrice                 GroupID = "stockPrice"
	GroupCorporateFinance           GroupID = "corporateFinance"
	GroupKRXListedInfo              GroupID = "krxListedInfo"

	OperationKISPrice             OperationID = "price"
	OperationKISDaily             OperationID = "daily"
	OperationKISIntraday          OperationID = "intraday"
	OperationKISOrderbook         OperationID = "orderbook"
	OperationKISTrades            OperationID = "trades"
	OperationKISTimeTrades        OperationID = "timeTrades"
	OperationKISProduct           OperationID = "product"
	OperationKISStock             OperationID = "stock"
	OperationKISETFETNPrice       OperationID = "etfetnPrice"
	OperationGetETFPriceInfo      OperationID = "getETFPriceInfo"
	OperationGetETNPriceInfo      OperationID = "getETNPriceInfo"
	OperationGetELWPriceInfo      OperationID = "getELWPriceInfo"
	OperationGetStockPriceInfo    OperationID = "getStockPriceInfo"
	OperationGetSummFinaStatV2    OperationID = "getSummFinaStat_V2"
	OperationGetBalanceSheetV2    OperationID = "getBs_V2"
	OperationGetIncomeStatementV2 OperationID = "getIncoStat_V2"
	OperationGetItemInfo          OperationID = "getItemInfo"

	MarketKRX Market = "krx"

	SecurityTypeETF   SecurityType = "etf"
	SecurityTypeETN   SecurityType = "etn"
	SecurityTypeELW   SecurityType = "elw"
	SecurityTypeStock SecurityType = "stock"

	CredentialScopeKIS    CredentialScope = "kis"
	CredentialScopeDataGo CredentialScope = "datago"

	FreshnessDaily  Freshness = "daily"
	FreshnessFiling Freshness = "filing"

	DataLatencyRealtime            DataLatency = "realtime"
	DataLatencyEndOfDay            DataLatency = "end_of_day"
	DataLatencyPreviousBusinessDay DataLatency = "previous_business_day"
	DataLatencyHistorical          DataLatency = "historical"

	RoleDailyBar   Role = "daily_bar"
	RoleFinancials Role = "financials"
	RoleInstrument Role = "instrument"
	RoleQuote      Role = "quote_snapshot"
)

type Compatibility struct {
	DataLatency         DataLatency
	LagBusinessDays     int
	CurrentDaySupported bool
	Notes               []string
}

type Identity struct {
	ID          ProviderID
	DisplayName string
}

func (i Identity) ProviderIdentity() Identity {
	return i
}

type IdentityProvider interface {
	ProviderIdentity() Identity
}
