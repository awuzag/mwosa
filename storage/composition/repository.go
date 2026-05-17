package composition

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	provider "github.com/ev3rlit/mwosa/providers/core"
	compositionrole "github.com/ev3rlit/mwosa/providers/core/composition"
	compositionservice "github.com/ev3rlit/mwosa/service/composition"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
)

const defaultPriceScale = 4

type repository struct {
	database *storage.Database
}

var _ compositionservice.Repository = (*repository)(nil)

func NewRepository(database *storage.Database) (compositionservice.Repository, error) {
	if database == nil {
		return nil, oops.In("composition_repository").New("composition repository database is nil")
	}
	return &repository{database: database}, nil
}

func (r *repository) UpsertComposition(ctx context.Context, aggregate compositionrole.Composition) (compositionservice.WriteResult, error) {
	errb := oops.In("composition_repository").With(
		"provider", aggregate.Source.Provider,
		"group", aggregate.Source.Group,
		"operation", aggregate.Source.Operation,
		"market", aggregate.Subject.Market,
		"security_type", aggregate.Subject.SecurityType,
		"symbol", aggregate.Subject.Symbol,
		"as_of_date", aggregate.AsOfDate,
		"observed_at_ms", aggregate.ObservedAtMS,
	)
	if err := validateAggregateKey(aggregate); err != nil {
		return compositionservice.WriteResult{}, errb.Wrap(err)
	}
	client, err := r.database.Client(ctx)
	if err != nil {
		return compositionservice.WriteResult{}, errb.Wrap(err)
	}
	tx, err := client.BeginTx(ctx, nil)
	if err != nil {
		return compositionservice.WriteResult{}, errb.Wrapf(err, "begin composition sqlite transaction")
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	nowMS := time.Now().UTC().UnixMilli()
	cache := upsertCache{
		markets:     make(map[string]storage.MarketV2Row),
		sources:     make(map[string]int64),
		instruments: make(map[string]storage.InstrumentV2Row),
	}
	sourceID, err := ensureProviderSource(ctx, tx, cache, aggregate.Source, nowMS)
	if err != nil {
		return compositionservice.WriteResult{}, errb.Wrap(err)
	}
	subject, err := ensureInstrument(ctx, tx, cache, aggregate.Subject, nowMS)
	if err != nil {
		return compositionservice.WriteResult{}, errb.Wrap(err)
	}
	root, err := upsertObservation(ctx, tx, sourceID, subject.ID, aggregate, nowMS)
	if err != nil {
		return compositionservice.WriteResult{}, errb.Wrap(err)
	}
	if err := replaceMembers(ctx, tx, cache, root.ID, aggregate.Members, nowMS); err != nil {
		return compositionservice.WriteResult{}, errb.Wrap(err)
	}

	if err := tx.Commit(); err != nil {
		return compositionservice.WriteResult{}, errb.Wrapf(err, "commit composition sqlite transaction")
	}
	committed = true
	return compositionservice.WriteResult{
		RowsAffected:       1 + len(aggregate.Members),
		CompositionsStored: 1,
		MembersStored:      len(aggregate.Members),
	}, nil
}

func (r *repository) GetComposition(ctx context.Context, query compositionservice.Query) (compositionrole.Composition, error) {
	errb := oops.In("composition_repository").With("provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "symbol", query.Symbol, "as_of_date", query.AsOfDate, "observed_at_ms", query.ObservedAtMS)
	if strings.TrimSpace(query.Symbol) == "" {
		return compositionrole.Composition{}, errb.New("get composition requires symbol")
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return compositionrole.Composition{}, errb.Wrap(err)
	}
	record, err := queryObservation(ctx, client, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return compositionrole.Composition{}, &compositionservice.NotFoundError{
				Symbol:       strings.TrimSpace(query.Symbol),
				Market:       query.Market,
				SecurityType: query.SecurityType,
				AsOfDate:     query.AsOfDate,
			}
		}
		return compositionrole.Composition{}, errb.Wrap(err)
	}
	members, err := queryMembers(ctx, client, record.ID)
	if err != nil {
		return compositionrole.Composition{}, errb.Wrap(err)
	}
	return compositionrole.Composition{
		Source: compositionrole.SourceRef{
			Provider:  provider.ProviderID(record.Provider),
			Group:     provider.GroupID(record.ProviderGroup),
			Operation: provider.OperationID(record.Operation),
		},
		Subject: compositionrole.InstrumentRef{
			Market:       provider.Market(record.Market),
			SecurityType: provider.SecurityType(record.SecurityType),
			Symbol:       record.Symbol,
			ISIN:         record.ISIN,
			Name:         record.Name,
		},
		AsOfDate:     record.AsOfDate,
		ObservedAtMS: record.ObservedAtMS,
		Members:      members,
	}, nil
}

type upsertCache struct {
	markets     map[string]storage.MarketV2Row
	sources     map[string]int64
	instruments map[string]storage.InstrumentV2Row
}

func validateAggregateKey(aggregate compositionrole.Composition) error {
	if aggregate.Source.Provider == "" || aggregate.Source.Group == "" || aggregate.Source.Operation == "" ||
		aggregate.Subject.SecurityType == "" || strings.TrimSpace(aggregate.Subject.Symbol) == "" ||
		strings.TrimSpace(aggregate.AsOfDate) == "" || aggregate.ObservedAtMS == 0 {
		return oops.In("composition_repository").With(
			"provider", aggregate.Source.Provider,
			"group", aggregate.Source.Group,
			"operation", aggregate.Source.Operation,
			"market", aggregate.Subject.Market,
			"security_type", aggregate.Subject.SecurityType,
			"symbol", aggregate.Subject.Symbol,
			"as_of_date", aggregate.AsOfDate,
			"observed_at_ms", aggregate.ObservedAtMS,
		).New("composition missing sqlite key")
	}
	for _, member := range aggregate.Members {
		if err := validateInstrumentRef(member.Instrument); err != nil {
			return err
		}
	}
	return nil
}

func validateInstrumentRef(ref compositionrole.InstrumentRef) error {
	if ref.SecurityType == "" || strings.TrimSpace(ref.Symbol) == "" {
		return oops.In("composition_repository").With("market", ref.Market, "security_type", ref.SecurityType, "symbol", ref.Symbol).New("composition instrument missing sqlite key")
	}
	return nil
}

func ensureMarket(ctx context.Context, tx bun.Tx, cache upsertCache, market provider.Market, nowMS int64) (storage.MarketV2Row, error) {
	code := string(marketWithDefault(market))
	if row, ok := cache.markets[code]; ok {
		return row, nil
	}
	row := storage.MarketV2Row{
		Code:               code,
		Timezone:           marketTimezone(provider.Market(code)),
		RegularOpenMinute:  marketRegularOpenMinute(provider.Market(code)),
		RegularCloseMinute: marketRegularCloseMinute(provider.Market(code)),
		CreatedAtMS:        nowMS,
		UpdatedAtMS:        nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (code) DO UPDATE").
		Set("timezone = EXCLUDED.timezone").
		Set("regular_open_minute = EXCLUDED.regular_open_minute").
		Set("regular_close_minute = EXCLUDED.regular_close_minute").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("composition_repository").With("market", code).Wrapf(err, "upsert market v2")
	}
	var stored storage.MarketV2Row
	if err := tx.NewSelect().Model(&stored).Where("code = ?", code).Limit(1).Scan(ctx); err != nil {
		return storage.MarketV2Row{}, oops.In("composition_repository").With("market", code).Wrapf(err, "select market v2")
	}
	cache.markets[code] = stored
	return stored, nil
}

func ensureProviderSource(ctx context.Context, tx bun.Tx, cache upsertCache, source compositionrole.SourceRef, nowMS int64) (int64, error) {
	key := strings.Join([]string{string(source.Provider), string(source.Group), string(source.Operation)}, "\x00")
	if id, ok := cache.sources[key]; ok {
		return id, nil
	}
	row := storage.ProviderSourceV2Row{
		Provider:      string(source.Provider),
		ProviderGroup: string(source.Group),
		Operation:     string(source.Operation),
		CreatedAtMS:   nowMS,
		UpdatedAtMS:   nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (provider, provider_group, operation) DO UPDATE").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return 0, oops.In("composition_repository").With("provider", source.Provider, "group", source.Group, "operation", source.Operation).Wrapf(err, "upsert provider source v2")
	}
	var stored storage.ProviderSourceV2Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("provider = ?", string(source.Provider)).
		Where("provider_group = ?", string(source.Group)).
		Where("operation = ?", string(source.Operation)).
		Limit(1).
		Scan(ctx); err != nil {
		return 0, oops.In("composition_repository").With("provider", source.Provider, "group", source.Group, "operation", source.Operation).Wrapf(err, "select provider source v2")
	}
	cache.sources[key] = stored.ID
	return stored.ID, nil
}

func ensureInstrument(ctx context.Context, tx bun.Tx, cache upsertCache, ref compositionrole.InstrumentRef, nowMS int64) (storage.InstrumentV2Row, error) {
	if err := validateInstrumentRef(ref); err != nil {
		return storage.InstrumentV2Row{}, err
	}
	market, err := ensureMarket(ctx, tx, cache, ref.Market, nowMS)
	if err != nil {
		return storage.InstrumentV2Row{}, err
	}
	symbol := strings.TrimSpace(ref.Symbol)
	key := strconv.FormatInt(market.ID, 10) + "\x00" + string(ref.SecurityType) + "\x00" + symbol
	if row, ok := cache.instruments[key]; ok {
		return row, nil
	}
	row := storage.InstrumentV2Row{
		MarketID:     market.ID,
		SecurityType: string(ref.SecurityType),
		Symbol:       symbol,
		ISIN:         strings.TrimSpace(ref.ISIN),
		Name:         strings.TrimSpace(ref.Name),
		CurrencyCode: "KRW",
		PriceScale:   defaultPriceScale,
		CreatedAtMS:  nowMS,
		UpdatedAtMS:  nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (market_id, security_type, symbol) DO UPDATE").
		Set("isin = CASE WHEN EXCLUDED.isin <> '' THEN EXCLUDED.isin ELSE instrument_v2.isin END").
		Set("name = CASE WHEN EXCLUDED.name <> '' THEN EXCLUDED.name ELSE instrument_v2.name END").
		Set("currency_code = EXCLUDED.currency_code").
		Set("price_scale = EXCLUDED.price_scale").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("composition_repository").With("market_id", market.ID, "security_type", ref.SecurityType, "symbol", symbol).Wrapf(err, "upsert instrument v2")
	}
	var stored storage.InstrumentV2Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("market_id = ?", market.ID).
		Where("security_type = ?", string(ref.SecurityType)).
		Where("symbol = ?", symbol).
		Limit(1).
		Scan(ctx); err != nil {
		return storage.InstrumentV2Row{}, oops.In("composition_repository").With("market_id", market.ID, "security_type", ref.SecurityType, "symbol", symbol).Wrapf(err, "select instrument v2")
	}
	cache.instruments[key] = stored
	return stored, nil
}

func upsertObservation(ctx context.Context, tx bun.Tx, sourceID int64, subjectID int64, aggregate compositionrole.Composition, nowMS int64) (storage.CompositionObservationV1Row, error) {
	row := storage.CompositionObservationV1Row{
		SourceID:            sourceID,
		SubjectInstrumentID: subjectID,
		AsOfDate:            strings.TrimSpace(aggregate.AsOfDate),
		ObservedAtMS:        aggregate.ObservedAtMS,
		CreatedAtMS:         nowMS,
		UpdatedAtMS:         nowMS,
	}
	if _, err := tx.NewInsert().
		Model(&row).
		On("CONFLICT (source_id, subject_instrument_id, as_of_date, observed_at_ms) DO UPDATE").
		Set("updated_at_ms = EXCLUDED.updated_at_ms").
		Exec(ctx); err != nil {
		return storage.CompositionObservationV1Row{}, oops.In("composition_repository").With("source_id", sourceID, "subject_instrument_id", subjectID, "as_of_date", aggregate.AsOfDate, "observed_at_ms", aggregate.ObservedAtMS).Wrapf(err, "upsert composition observation")
	}
	var stored storage.CompositionObservationV1Row
	if err := tx.NewSelect().
		Model(&stored).
		Where("source_id = ?", sourceID).
		Where("subject_instrument_id = ?", subjectID).
		Where("as_of_date = ?", strings.TrimSpace(aggregate.AsOfDate)).
		Where("observed_at_ms = ?", aggregate.ObservedAtMS).
		Limit(1).
		Scan(ctx); err != nil {
		return storage.CompositionObservationV1Row{}, oops.In("composition_repository").With("source_id", sourceID, "subject_instrument_id", subjectID, "as_of_date", aggregate.AsOfDate, "observed_at_ms", aggregate.ObservedAtMS).Wrapf(err, "select composition observation")
	}
	return stored, nil
}

func replaceMembers(ctx context.Context, tx bun.Tx, cache upsertCache, compositionID int64, members []compositionrole.CompositionMember, nowMS int64) error {
	errb := oops.In("composition_repository").With("composition_id", compositionID)
	if _, err := tx.NewDelete().
		Model((*storage.CompositionMemberV1Row)(nil)).
		Where("composition_id = ?", compositionID).
		Exec(ctx); err != nil {
		return errb.Wrapf(err, "delete composition members")
	}
	for ordinal, member := range members {
		instrument, err := ensureInstrument(ctx, tx, cache, member.Instrument, nowMS)
		if err != nil {
			return errb.Wrap(err)
		}
		row := storage.CompositionMemberV1Row{
			CompositionID:      compositionID,
			MemberInstrumentID: instrument.ID,
			Ordinal:            ordinal,
			WeightValue:        strings.TrimSpace(member.Weight.Value),
			QuantityValue:      strings.TrimSpace(member.Quantity.Value),
			ValuationCurrency:  strings.TrimSpace(member.Valuation.Currency),
			ValuationValue:     strings.TrimSpace(member.Valuation.Value),
			CreatedAtMS:        nowMS,
			UpdatedAtMS:        nowMS,
		}
		if _, err := tx.NewInsert().Model(&row).Exec(ctx); err != nil {
			return errb.With("member_symbol", member.Instrument.Symbol).Wrapf(err, "insert composition member")
		}
	}
	return nil
}

type observationRecord struct {
	ID            int64
	Provider      string
	ProviderGroup string
	Operation     string
	AsOfDate      string
	ObservedAtMS  int64
	Market        string
	SecurityType  string
	Symbol        string
	ISIN          string
	Name          string
}

type queryDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func queryObservation(ctx context.Context, db queryDB, query compositionservice.Query) (observationRecord, error) {
	sqlQuery, args := observationSelectSQL(query)
	var row observationRecord
	err := db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&row.ID,
		&row.Provider,
		&row.ProviderGroup,
		&row.Operation,
		&row.AsOfDate,
		&row.ObservedAtMS,
		&row.Market,
		&row.SecurityType,
		&row.Symbol,
		&row.ISIN,
		&row.Name,
	)
	if err != nil {
		return observationRecord{}, err
	}
	return row, nil
}

func observationSelectSQL(query compositionservice.Query) (string, []any) {
	var builder strings.Builder
	builder.WriteString(`SELECT
	c.id,
	s.provider,
	s.provider_group,
	s.operation,
	c.as_of_date,
	c.observed_at_ms,
	m.code,
	i.security_type,
	i.symbol,
	i.isin,
	i.name
FROM composition_observation_v1 AS c
JOIN provider_source_v2 AS s ON s.id = c.source_id
JOIN instrument_v2 AS i ON i.id = c.subject_instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE m.code = ? AND i.symbol = ?`)
	args := []any{string(marketWithDefault(query.Market)), strings.TrimSpace(query.Symbol)}
	if query.SecurityType != "" {
		builder.WriteString(" AND i.security_type = ?")
		args = append(args, string(query.SecurityType))
	}
	if query.ProviderID != "" {
		builder.WriteString(" AND s.provider = ?")
		args = append(args, string(query.ProviderID))
	}
	if strings.TrimSpace(query.AsOfDate) != "" {
		builder.WriteString(" AND c.as_of_date = ?")
		args = append(args, strings.TrimSpace(query.AsOfDate))
	}
	if query.ObservedAtMS != 0 {
		builder.WriteString(" AND c.observed_at_ms = ?")
		args = append(args, query.ObservedAtMS)
	}
	builder.WriteString(" ORDER BY c.as_of_date DESC, c.observed_at_ms DESC LIMIT 1")
	return builder.String(), args
}

func queryMembers(ctx context.Context, db queryDB, compositionID int64) ([]compositionrole.CompositionMember, error) {
	rows, err := db.QueryContext(ctx, `SELECT
	m.code,
	i.security_type,
	i.symbol,
	i.isin,
	i.name,
	cm.weight_value,
	cm.quantity_value,
	cm.valuation_currency,
	cm.valuation_value
FROM composition_member_v1 AS cm
JOIN instrument_v2 AS i ON i.id = cm.member_instrument_id
JOIN market_v2 AS m ON m.id = i.market_id
WHERE cm.composition_id = ?
ORDER BY cm.ordinal ASC`, compositionID)
	if err != nil {
		return nil, oops.In("composition_repository").With("composition_id", compositionID).Wrapf(err, "query composition members")
	}
	defer rows.Close()

	members := make([]compositionrole.CompositionMember, 0)
	for rows.Next() {
		var member compositionrole.CompositionMember
		var market string
		var securityType string
		if err := rows.Scan(
			&market,
			&securityType,
			&member.Instrument.Symbol,
			&member.Instrument.ISIN,
			&member.Instrument.Name,
			&member.Weight.Value,
			&member.Quantity.Value,
			&member.Valuation.Currency,
			&member.Valuation.Value,
		); err != nil {
			return nil, oops.In("composition_repository").With("composition_id", compositionID).Wrapf(err, "scan composition member")
		}
		member.Instrument.Market = provider.Market(market)
		member.Instrument.SecurityType = provider.SecurityType(securityType)
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, oops.In("composition_repository").With("composition_id", compositionID).Wrapf(err, "iterate composition members")
	}
	return members, nil
}

func marketWithDefault(market provider.Market) provider.Market {
	if market == "" {
		return provider.MarketKRX
	}
	return market
}

func marketTimezone(market provider.Market) string {
	if market == provider.MarketKRX {
		return "Asia/Seoul"
	}
	return "UTC"
}

func marketRegularOpenMinute(market provider.Market) int {
	if market == provider.MarketKRX {
		return 9 * 60
	}
	return 0
}

func marketRegularCloseMinute(market provider.Market) int {
	if market == provider.MarketKRX {
		return 15*60 + 30
	}
	return 0
}
