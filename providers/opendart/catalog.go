package opendart

import provider "github.com/ev3rlit/mwosa/providers/core"

type CatalogService struct {
	Category         string
	Group            provider.GroupID
	Operation        provider.OperationID
	Description      string
	CanonicalSupport string
}

func ServiceCatalog() []CatalogService {
	return []CatalogService{
		{
			Category:         "disclosure",
			Group:            provider.GroupOpenDARTDisclosure,
			Operation:        provider.OperationOpenDARTCorpCode,
			Description:      "OpenDART company registry: corp_code, corp_name, corp_eng_name, stock_code, modify_date",
			CanonicalSupport: "company_registry",
		},
		{
			Category:         "disclosure",
			Group:            provider.GroupOpenDARTDisclosure,
			Operation:        provider.OperationOpenDARTList,
			Description:      "OpenDART disclosure search by corp_code and date range",
			CanonicalSupport: "filings",
		},
		{
			Category:         "financial",
			Group:            provider.GroupOpenDARTFinancials,
			Operation:        provider.OperationOpenDARTSinglAcntAll,
			Description:      "OpenDART single-company full financial statements; stock_code is resolved to corp_code",
			CanonicalSupport: "financials",
		},
		{
			Category:         "periodic_report",
			Group:            provider.GroupOpenDARTPeriodicReport,
			Operation:        provider.OperationOpenDARTAlotMatter,
			Description:      "OpenDART dividend matters from periodic reports",
			CanonicalSupport: "company_facts/dividends",
		},
		{
			Category:         "periodic_report",
			Group:            provider.GroupOpenDARTPeriodicReport,
			Operation:        provider.OperationOpenDARTTesstkAcqs,
			Description:      "OpenDART treasury stock acquisition and disposal status from periodic reports",
			CanonicalSupport: "company_facts/treasury_stock",
		},
		{
			Category:         "periodic_report",
			Group:            provider.GroupOpenDARTPeriodicReport,
			Operation:        provider.OperationOpenDARTHyslrSttus,
			Description:      "OpenDART major shareholder status from periodic reports",
			CanonicalSupport: "company_facts/major_shareholder",
		},
		{
			Category:         "periodic_report",
			Group:            provider.GroupOpenDARTPeriodicReport,
			Operation:        provider.OperationOpenDARTHyslrChg,
			Description:      "OpenDART major shareholder changes from periodic reports",
			CanonicalSupport: "company_facts/major_shareholder_change",
		},
		{
			Category:         "periodic_report",
			Group:            provider.GroupOpenDARTPeriodicReport,
			Operation:        provider.OperationOpenDARTEmpSttus,
			Description:      "OpenDART employee status from periodic reports",
			CanonicalSupport: "company_facts/employee",
		},
		{
			Category:         "periodic_report",
			Group:            provider.GroupOpenDARTPeriodicReport,
			Operation:        provider.OperationOpenDARTAuditOpinion,
			Description:      "OpenDART auditor name and audit opinion from periodic reports",
			CanonicalSupport: "company_facts/audit_opinion",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTDfOcr,
			Description:      "OpenDART default occurrence reports",
			CanonicalSupport: "company_events/default_occurrence",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTPiicDecsn,
			Description:      "OpenDART paid-in capital increase decisions from material event reports",
			CanonicalSupport: "company_events/paid_in_capital_increase",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTFricDecsn,
			Description:      "OpenDART free capital increase decisions from material event reports",
			CanonicalSupport: "company_events/free_capital_increase",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTPifricDecsn,
			Description:      "OpenDART paid-in and free capital increase decisions from material event reports",
			CanonicalSupport: "company_events/paid_in_free_capital_increase",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTCrDecsn,
			Description:      "OpenDART capital reduction decisions from material event reports",
			CanonicalSupport: "company_events/capital_reduction",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTBnkMngtPcbg,
			Description:      "OpenDART bank management procedure start reports",
			CanonicalSupport: "company_events/bank_management_procedure_start",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTLwstLg,
			Description:      "OpenDART lawsuit filing reports",
			CanonicalSupport: "company_events/lawsuit_filing",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTBsnInhDecsn,
			Description:      "OpenDART business transfer-in decisions from material event reports",
			CanonicalSupport: "company_events/business_transfer_in",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTBsnTrfDecsn,
			Description:      "OpenDART business transfer-out decisions from material event reports",
			CanonicalSupport: "company_events/business_transfer_out",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTTgastInhDecsn,
			Description:      "OpenDART tangible asset transfer-in decisions from material event reports",
			CanonicalSupport: "company_events/tangible_asset_transfer_in",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTTgastTrfDecsn,
			Description:      "OpenDART tangible asset transfer-out decisions from material event reports",
			CanonicalSupport: "company_events/tangible_asset_transfer_out",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTCvbdIsDecsn,
			Description:      "OpenDART convertible bond issuance decisions from material event reports",
			CanonicalSupport: "company_events/convertible_bond_issuance",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTBdwtIsDecsn,
			Description:      "OpenDART bond with warrant issuance decisions from material event reports",
			CanonicalSupport: "company_events/bond_with_warrant_issuance",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTExbdIsDecsn,
			Description:      "OpenDART exchangeable bond issuance decisions from material event reports",
			CanonicalSupport: "company_events/exchangeable_bond_issuance",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTCmpMgDecsn,
			Description:      "OpenDART company merger decisions from material event reports",
			CanonicalSupport: "company_events/company_merger",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTCmpDvDecsn,
			Description:      "OpenDART company division decisions from material event reports",
			CanonicalSupport: "company_events/company_division",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTCmpDvmgDecsn,
			Description:      "OpenDART company division merger decisions from material event reports",
			CanonicalSupport: "company_events/company_division_merger",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTStkExtrDecsn,
			Description:      "OpenDART stock exchange or transfer decisions from material event reports",
			CanonicalSupport: "company_events/stock_exchange_transfer",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTTsstkAqDecsn,
			Description:      "OpenDART treasury stock acquisition decisions from material event reports",
			CanonicalSupport: "company_events/treasury_stock_acquisition",
		},
		{
			Category:         "material_event",
			Group:            provider.GroupOpenDARTMaterialEvents,
			Operation:        provider.OperationOpenDARTTsstkDpDecsn,
			Description:      "OpenDART treasury stock disposal decisions from material event reports",
			CanonicalSupport: "company_events/treasury_stock_disposal",
		},
	}
}
