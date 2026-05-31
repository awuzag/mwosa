package macro

import (
	"context"
	"fmt"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	"github.com/samber/oops"
)

type Router interface {
	RouteMacro(ctx context.Context, input macrorole.RouteInput) (macrorole.Fetcher, error)
}

type ReadRepository interface {
	QueryIndicators(ctx context.Context, query IndicatorQuery) ([]macrorole.Indicator, error)
	QueryObservations(ctx context.Context, query ObservationQuery) ([]macrorole.Observation, error)
}

type WriteRepository interface {
	UpsertIndicators(ctx context.Context, indicators []macrorole.Indicator) (IndicatorWriteResult, error)
	UpsertObservations(ctx context.Context, observations []macrorole.Observation) (ObservationWriteResult, error)
}

type IndicatorQuery struct {
	ProviderID  provider.ProviderID
	Preset      macrorole.Preset
	IndicatorID string
}

type ObservationQuery struct {
	IndicatorID string
	From        string
	To          string
}

type IndicatorWriteResult struct {
	RowsAffected      int `json:"rows_affected" csv:"rows_affected"`
	IndicatorsWritten int `json:"indicators_written" csv:"indicators_written"`
	SourcesWritten    int `json:"sources_written" csv:"sources_written"`
	DocumentsWritten  int `json:"documents_written" csv:"documents_written"`
}

type ObservationWriteResult struct {
	RowsAffected        int `json:"rows_affected" csv:"rows_affected"`
	ObservationsWritten int `json:"observations_written" csv:"observations_written"`
}

type ListIndicatorsRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Preset         macrorole.Preset
}

type GetObservationsRequest struct {
	IndicatorID string
	From        string
	To          string
}

type SyncPresetRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	Preset         macrorole.Preset
}

type SyncObservationsRequest struct {
	ProviderID     provider.ProviderID
	PreferProvider provider.ProviderID
	IndicatorID    string
	From           string
	To             string
}

type IndicatorsResult struct {
	Indicators []macrorole.Indicator `json:"indicators"`
}

type ObservationsResult struct {
	IndicatorID  string                  `json:"indicator_id"`
	Observations []macrorole.Observation `json:"observations"`
}

type SyncPresetResult struct {
	Preset            macrorole.Preset     `json:"preset" csv:"preset"`
	ProviderID        provider.ProviderID  `json:"provider" csv:"provider"`
	Group             provider.GroupID     `json:"provider_group" csv:"group"`
	Operation         provider.OperationID `json:"operation" csv:"operation"`
	IndicatorsFetched int                  `json:"indicators_fetched" csv:"fetched"`
	IndicatorsStored  int                  `json:"indicators_stored" csv:"stored"`
	SourcesStored     int                  `json:"sources_stored" csv:"sources"`
	DocumentsStored   int                  `json:"documents_stored" csv:"documents"`
	RowsAffected      int                  `json:"rows_affected" csv:"rows_affected"`
}

type SyncObservationsResult struct {
	IndicatorID         string               `json:"indicator_id" csv:"indicator_id"`
	ProviderID          provider.ProviderID  `json:"provider" csv:"provider"`
	Group               provider.GroupID     `json:"provider_group" csv:"group"`
	Operation           provider.OperationID `json:"operation" csv:"operation"`
	From                string               `json:"from,omitempty" csv:"from"`
	To                  string               `json:"to,omitempty" csv:"to"`
	ObservationsFetched int                  `json:"observations_fetched" csv:"fetched"`
	ObservationsStored  int                  `json:"observations_stored" csv:"stored"`
	RowsAffected        int                  `json:"rows_affected" csv:"rows_affected"`
}

type ReadService struct {
	reader ReadRepository
}

type Service struct {
	router Router
	reader ReadRepository
	writer WriteRepository
}

func NewReadService(reader ReadRepository) (ReadService, error) {
	if reader == nil {
		return ReadService{}, oops.In("macro_service").New("macro read repository is nil")
	}
	return ReadService{reader: reader}, nil
}

func NewService(reader ReadRepository, writer WriteRepository, router Router) (Service, error) {
	errb := oops.In("macro_service")
	if reader == nil {
		return Service{}, errb.New("macro read repository is nil")
	}
	if writer == nil {
		return Service{}, errb.New("macro write repository is nil")
	}
	if router == nil {
		return Service{}, errb.New("macro router is nil")
	}
	return Service{router: router, reader: reader, writer: writer}, nil
}

func (s ReadService) ListIndicators(ctx context.Context, req ListIndicatorsRequest) (IndicatorsResult, error) {
	preset, err := normalizePreset(req.Preset)
	if err != nil {
		return IndicatorsResult{}, err
	}
	indicators, err := s.reader.QueryIndicators(ctx, IndicatorQuery{
		ProviderID: req.ProviderID,
		Preset:     preset,
	})
	if err != nil {
		return IndicatorsResult{}, oops.In("macro_service").With("provider", req.ProviderID, "preset", preset).Wrapf(err, "query macro indicators")
	}
	return IndicatorsResult{Indicators: indicators}, nil
}

func (s ReadService) GetObservations(ctx context.Context, req GetObservationsRequest) (ObservationsResult, error) {
	query, err := observationQuery(req.IndicatorID, req.From, req.To)
	if err != nil {
		return ObservationsResult{}, err
	}
	observations, err := s.reader.QueryObservations(ctx, query)
	if err != nil {
		return ObservationsResult{}, oops.In("macro_service").With("indicator_id", query.IndicatorID, "from", query.From, "to", query.To).Wrapf(err, "query macro observations")
	}
	if len(observations) == 0 {
		return ObservationsResult{}, notFound(query)
	}
	return ObservationsResult{IndicatorID: query.IndicatorID, Observations: observations}, nil
}

func (s Service) ListIndicators(ctx context.Context, req ListIndicatorsRequest) (IndicatorsResult, error) {
	preset, err := normalizePreset(req.Preset)
	if err != nil {
		return IndicatorsResult{}, err
	}
	if req.ProviderID == "" && req.PreferProvider == "" {
		return ReadService{reader: s.reader}.ListIndicators(ctx, req)
	}
	result, err := s.fetchIndicators(ctx, req.ProviderID, req.PreferProvider, preset)
	if err != nil {
		return IndicatorsResult{}, err
	}
	return IndicatorsResult{Indicators: result.Indicators}, nil
}

func (s Service) SyncPreset(ctx context.Context, req SyncPresetRequest) (SyncPresetResult, error) {
	preset, err := normalizePreset(req.Preset)
	if err != nil {
		return SyncPresetResult{}, err
	}
	result, err := s.fetchIndicators(ctx, req.ProviderID, req.PreferProvider, preset)
	if err != nil {
		return SyncPresetResult{}, err
	}
	writeResult, err := s.writer.UpsertIndicators(ctx, result.Indicators)
	if err != nil {
		return SyncPresetResult{}, oops.In("macro_service").With("provider", result.Provider.ID, "preset", preset).Wrapf(err, "store macro indicators")
	}
	return SyncPresetResult{
		Preset:            preset,
		ProviderID:        result.Provider.ID,
		Group:             result.Group,
		Operation:         result.Operation,
		IndicatorsFetched: len(result.Indicators),
		IndicatorsStored:  writeResult.IndicatorsWritten,
		SourcesStored:     writeResult.SourcesWritten,
		DocumentsStored:   writeResult.DocumentsWritten,
		RowsAffected:      writeResult.RowsAffected,
	}, nil
}

func (s Service) SyncObservations(ctx context.Context, req SyncObservationsRequest) (SyncObservationsResult, error) {
	query, err := observationQuery(req.IndicatorID, req.From, req.To)
	if err != nil {
		return SyncObservationsResult{}, err
	}
	if query.From == "" || query.To == "" {
		return SyncObservationsResult{}, oops.In("macro_service").With("indicator_id", query.IndicatorID).New("sync macro observations requires --from and --to")
	}
	indicator, err := s.lookupIndicator(ctx, query.IndicatorID)
	if err != nil {
		return SyncObservationsResult{}, err
	}
	fetcher, err := s.route(ctx, req.ProviderID, req.PreferProvider, query.IndicatorID)
	if err != nil {
		return SyncObservationsResult{}, err
	}
	result, err := fetcher.FetchMacroObservations(ctx, macrorole.ObservationInput{
		IndicatorID: query.IndicatorID,
		SourceCode:  indicator.SourceCode,
		From:        query.From,
		To:          query.To,
	})
	if err != nil {
		return SyncObservationsResult{}, oops.In("macro_service").With("indicator_id", query.IndicatorID, "from", query.From, "to", query.To).Wrapf(err, "fetch macro observations")
	}
	writeResult, err := s.writer.UpsertObservations(ctx, result.Observations)
	if err != nil {
		return SyncObservationsResult{}, oops.In("macro_service").With("indicator_id", query.IndicatorID).Wrapf(err, "store macro observations")
	}
	return SyncObservationsResult{
		IndicatorID:         query.IndicatorID,
		ProviderID:          result.Provider.ID,
		Group:               result.Group,
		Operation:           result.Operation,
		From:                query.From,
		To:                  query.To,
		ObservationsFetched: len(result.Observations),
		ObservationsStored:  writeResult.ObservationsWritten,
		RowsAffected:        writeResult.RowsAffected,
	}, nil
}

func (s Service) fetchIndicators(ctx context.Context, providerID provider.ProviderID, preferProvider provider.ProviderID, preset macrorole.Preset) (macrorole.IndicatorResult, error) {
	fetcher, err := s.route(ctx, providerID, preferProvider, string(preset))
	if err != nil {
		return macrorole.IndicatorResult{}, err
	}
	result, err := fetcher.FetchMacroIndicators(ctx, macrorole.IndicatorInput{Preset: preset})
	if err != nil {
		return macrorole.IndicatorResult{}, oops.In("macro_service").With("provider", providerID, "prefer_provider", preferProvider, "preset", preset).Wrapf(err, "fetch macro indicators")
	}
	return result, nil
}

func (s Service) route(ctx context.Context, providerID provider.ProviderID, preferProvider provider.ProviderID, symbol string) (macrorole.Fetcher, error) {
	if s.router == nil {
		return nil, oops.In("macro_service").New("macro router is nil")
	}
	fetcher, err := s.router.RouteMacro(ctx, macrorole.RouteInput{
		ProviderID:     providerID,
		PreferProvider: preferProvider,
		Group:          provider.GroupECOSKeyStatistics,
		Operation:      provider.OperationECOSKeyStatistics,
		IndicatorID:    symbol,
	})
	if err != nil {
		return nil, oops.In("macro_service").With("provider", providerID, "prefer_provider", preferProvider, "symbol", symbol).Wrapf(err, "route macro")
	}
	return fetcher, nil
}

func (s Service) lookupIndicator(ctx context.Context, indicatorID string) (macrorole.Indicator, error) {
	indicators, err := s.reader.QueryIndicators(ctx, IndicatorQuery{IndicatorID: indicatorID})
	if err != nil {
		return macrorole.Indicator{}, oops.In("macro_service").With("indicator_id", indicatorID).Wrapf(err, "query macro indicator")
	}
	if len(indicators) == 0 {
		return macrorole.Indicator{}, oops.In("macro_service").With("indicator_id", indicatorID).New("macro indicator metadata not found; run `mwosa sync macro key-statistics --provider ecos` first")
	}
	return indicators[0], nil
}

func normalizePreset(value macrorole.Preset) (macrorole.Preset, error) {
	trimmed := macrorole.Preset(strings.TrimSpace(string(value)))
	if trimmed == "" {
		return "", nil
	}
	if trimmed != macrorole.PresetKeyStatistics {
		return "", oops.In("macro_service").With("preset", trimmed).Errorf("unsupported macro preset: %s", trimmed)
	}
	return trimmed, nil
}

func observationQuery(indicatorID string, from string, to string) (ObservationQuery, error) {
	query := ObservationQuery{
		IndicatorID: strings.TrimSpace(indicatorID),
		From:        strings.TrimSpace(from),
		To:          strings.TrimSpace(to),
	}
	errb := oops.In("macro_service").With("indicator_id", query.IndicatorID, "from", query.From, "to", query.To)
	if query.IndicatorID == "" {
		return ObservationQuery{}, errb.New("macro observation request requires indicator id")
	}
	if query.From != "" && query.To != "" && query.From > query.To {
		return ObservationQuery{}, errb.New("--from must be on or before --to")
	}
	return query, nil
}

func notFound(query ObservationQuery) error {
	hint := fmt.Sprintf("run `mwosa sync macro %s --provider <provider> --from <period> --to <period>`", query.IndicatorID)
	return oops.In("macro_service").With(
		"indicator_id", query.IndicatorID,
		"from", query.From,
		"to", query.To,
	).New("macro observations not found " + hint)
}
