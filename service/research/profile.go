package research

type SourceRef struct {
	Provider     string `json:"provider,omitempty"`
	Group        string `json:"provider_group,omitempty"`
	Operation    string `json:"operation,omitempty"`
	ReportCode   string `json:"report_code,omitempty"`
	ReceiptNo    string `json:"receipt_no,omitempty"`
	SourceTable  string `json:"source_table,omitempty"`
	SourceRecord string `json:"source_record,omitempty"`
}

type CompanyIdentifier struct {
	Type       string    `json:"type"`
	Value      string    `json:"value"`
	Primary    bool      `json:"primary,omitempty"`
	Confidence float64   `json:"confidence,omitempty"`
	Source     SourceRef `json:"source,omitempty"`
}

type InstrumentIdentity struct {
	InstrumentID int64  `json:"instrument_id,omitempty"`
	Market       string `json:"market,omitempty"`
	SecurityType string `json:"security_type,omitempty"`
	Symbol       string `json:"symbol,omitempty"`
	Name         string `json:"name,omitempty"`
}

type CompanyIdentity struct {
	CompanyID   int64               `json:"company_id"`
	Name        string              `json:"name"`
	LegalName   string              `json:"legal_name,omitempty"`
	CountryCode string              `json:"country_code,omitempty"`
	Instrument  *InstrumentIdentity `json:"instrument,omitempty"`
	Identifiers []CompanyIdentifier `json:"identifiers,omitempty"`
}

type FinancialMetric struct {
	Metric             string `json:"metric,omitempty"`
	FiscalYear         string `json:"fiscal_year,omitempty"`
	FiscalPeriod       string `json:"fiscal_period,omitempty"`
	AsOfDate           string `json:"as_of_date,omitempty"`
	ValueDecimal       string `json:"value_decimal,omitempty"`
	ValueBP            *int64 `json:"value_bp,omitempty"`
	ValueMinor         *int64 `json:"value_minor,omitempty"`
	FormulaVersion     string `json:"formula_version,omitempty"`
	UncomputableReason string `json:"uncomputable_reason,omitempty"`
}

type FinancialProfile struct {
	Metrics []FinancialMetric `json:"metrics,omitempty"`
}

type ValuationSnapshot struct {
	AsOfDate            string            `json:"as_of_date,omitempty"`
	SourcePriceDate     string            `json:"source_price_date,omitempty"`
	MarketCapMinor      *int64            `json:"market_cap_minor,omitempty"`
	ClosePriceMinor     *int64            `json:"close_price_minor,omitempty"`
	SharesOutstanding   *int64            `json:"shares_outstanding,omitempty"`
	PerBP               *int64            `json:"per_bp,omitempty"`
	PbrBP               *int64            `json:"pbr_bp,omitempty"`
	PsrBP               *int64            `json:"psr_bp,omitempty"`
	EpsMinor            *int64            `json:"eps_minor,omitempty"`
	BpsMinor            *int64            `json:"bps_minor,omitempty"`
	DividendYieldBP     *int64            `json:"dividend_yield_bp,omitempty"`
	MetricSourceVersion string            `json:"metric_source_version,omitempty"`
	Uncomputable        map[string]string `json:"uncomputable,omitempty"`
}

type ValuationProfile struct {
	Snapshots []ValuationSnapshot `json:"snapshots,omitempty"`
}

type CompanyFact struct {
	FactType     string    `json:"fact_type"`
	FiscalYear   string    `json:"fiscal_year,omitempty"`
	ReportCode   string    `json:"report_code,omitempty"`
	RceptNo      string    `json:"rcept_no,omitempty"`
	FactDate     string    `json:"fact_date,omitempty"`
	Key          string    `json:"key"`
	ValueText    string    `json:"value_text,omitempty"`
	ValueNumber  string    `json:"value_number,omitempty"`
	CurrencyCode string    `json:"currency_code,omitempty"`
	Provider     string    `json:"provider,omitempty"`
	Group        string    `json:"provider_group,omitempty"`
	Operation    string    `json:"operation,omitempty"`
	Source       SourceRef `json:"source,omitempty"`
}

type CapitalPolicyProfile struct {
	Dividends []CompanyFact `json:"dividends,omitempty"`
}

type GovernanceProfile struct {
	Facts []CompanyFact `json:"facts,omitempty"`
}

type DisclosureEvent struct {
	EventType   string    `json:"event_type"`
	EventDate   string    `json:"event_date,omitempty"`
	RceptDt     string    `json:"rcept_dt,omitempty"`
	RceptNo     string    `json:"rcept_no,omitempty"`
	Title       string    `json:"title,omitempty"`
	AmountMinor *int64    `json:"amount_minor,omitempty"`
	ValueText   string    `json:"value_text,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Group       string    `json:"provider_group,omitempty"`
	Operation   string    `json:"operation,omitempty"`
	Source      SourceRef `json:"source,omitempty"`
}

type DisclosureEventTimeline struct {
	Events []DisclosureEvent `json:"events,omitempty"`
}

type ScreenCandidate struct {
	Symbol    string                     `json:"symbol"`
	Metrics   map[string]FinancialMetric `json:"financial_metrics,omitempty"`
	Valuation *ValuationSnapshot         `json:"valuation,omitempty"`
	Facts     map[string]CompanyFact     `json:"company_facts,omitempty"`
	Events    []DisclosureEvent          `json:"company_events,omitempty"`
}

type MissingSection struct {
	Section string `json:"section" csv:"section"`
	Reason  string `json:"reason" csv:"reason"`
}

type StockResearchProfile struct {
	Sections                []string                 `json:"sections"`
	Company                 CompanyIdentity          `json:"company"`
	FinancialProfile        *FinancialProfile        `json:"financial_profile,omitempty"`
	ValuationProfile        *ValuationProfile        `json:"valuation_profile,omitempty"`
	CapitalPolicyProfile    *CapitalPolicyProfile    `json:"capital_policy_profile,omitempty"`
	GovernanceProfile       *GovernanceProfile       `json:"governance_profile,omitempty"`
	DisclosureEventTimeline *DisclosureEventTimeline `json:"disclosure_event_timeline,omitempty"`
	ScreenCandidate         *ScreenCandidate         `json:"screen_candidate,omitempty"`
	Missing                 []MissingSection         `json:"missing,omitempty"`
}
