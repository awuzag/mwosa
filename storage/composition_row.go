package storage

import "github.com/uptrace/bun"

type CompositionObservationV1Row struct {
	bun.BaseModel `bun:"table:composition_observation_v1,alias:composition_observation_v1"`

	ID                  int64  `bun:"id,pk,autoincrement"`
	SourceID            int64  `bun:"source_id,notnull"`
	SubjectInstrumentID int64  `bun:"subject_instrument_id,notnull"`
	AsOfDate            string `bun:"as_of_date,notnull"`
	ObservedAtMS        int64  `bun:"observed_at_ms,notnull"`
	CreatedAtMS         int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS         int64  `bun:"updated_at_ms,notnull"`
}

type CompositionMemberV1Row struct {
	bun.BaseModel `bun:"table:composition_member_v1,alias:composition_member_v1"`

	CompositionID      int64  `bun:"composition_id,pk,notnull"`
	MemberInstrumentID int64  `bun:"member_instrument_id,pk,notnull"`
	Ordinal            int    `bun:"ordinal,notnull"`
	WeightValue        string `bun:"weight_value,notnull,default:''"`
	QuantityValue      string `bun:"quantity_value,notnull,default:''"`
	ValuationCurrency  string `bun:"valuation_currency,notnull,default:''"`
	ValuationValue     string `bun:"valuation_value,notnull,default:''"`
	CreatedAtMS        int64  `bun:"created_at_ms,notnull"`
	UpdatedAtMS        int64  `bun:"updated_at_ms,notnull"`
}
