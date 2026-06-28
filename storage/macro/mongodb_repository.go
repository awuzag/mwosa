package macro

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	macrorole "github.com/awuzag/mwosa/providers/core/macro"
	macroservice "github.com/awuzag/mwosa/service/macro"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	indicators   *mongo.Collection
	observations *mongo.Collection
}

type macroIndicatorMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	IndicatorID  string                       `bson:"indicator_id"`
	Preset       string                       `bson:"preset,omitempty"`
	Provider     string                       `bson:"provider"`
	SourceCode   string                       `bson:"source_code"`
	Name         string                       `bson:"name"`
	FriendlyName string                       `bson:"friendly_name,omitempty"`
	Category     string                       `bson:"category,omitempty"`
	Frequency    string                       `bson:"frequency,omitempty"`
	Unit         string                       `bson:"unit,omitempty"`
	Scale        string                       `bson:"scale,omitempty"`
	Active       bool                         `bson:"active"`
	Sources      []macroIndicatorMongoSource  `bson:"sources"`
	ProviderDocs []macroProviderMongoDocument `bson:"provider_docs,omitempty"`
}

type macroIndicatorMongoSource struct {
	Provider   string `bson:"provider"`
	SourceCode string `bson:"source_code"`
	SourceName string `bson:"source_name,omitempty"`
	SourceURL  string `bson:"source_url,omitempty"`
}

type macroProviderMongoDocument struct {
	Provider      string         `bson:"provider"`
	SchemaVersion string         `bson:"schema_version"`
	Document      map[string]any `bson:"document"`
	UpdatedAt     string         `bson:"updated_at,omitempty"`
	UpdatedAtMS   int64          `bson:"updated_at_ms"`
}

type macroObservationMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
	CollectedAt   time.Time `bson:"collected_at"`

	IndicatorID         string `bson:"indicator_id"`
	Provider            string `bson:"provider,omitempty"`
	SourceCode          string `bson:"source_code,omitempty"`
	Period              string `bson:"period"`
	Value               string `bson:"value"`
	PublishedAt         string `bson:"published_at,omitempty"`
	CollectedAtText     string `bson:"collected_at_text"`
	ObservationRevision int    `bson:"observation_revision"`
}

var _ macroservice.ReadRepository = (*mongoRepository)(nil)
var _ macroservice.WriteRepository = (*mongoRepository)(nil)

func NewMongoRepositories(database *mongo.Database) (macroservice.ReadRepository, macroservice.WriteRepository, error) {
	if database == nil {
		return nil, nil, oops.In("macro_repository").New("mongodb database is nil")
	}
	repository := &mongoRepository{
		indicators:   database.Collection("macro_indicators"),
		observations: database.Collection("macro_observations"),
	}
	return repository, repository, nil
}

func (r *mongoRepository) QueryIndicators(ctx context.Context, query macroservice.IndicatorQuery) ([]macrorole.Indicator, error) {
	errb := oops.In("macro_repository").With("backend", "mongodb", "provider", query.ProviderID, "preset", query.Preset, "indicator_id", query.IndicatorID)
	cursor, err := r.indicators.Find(ctx, macroIndicatorMongoFilter(query), options.Find().SetSort(bson.D{
		{Key: "provider", Value: 1},
		{Key: "preset", Value: 1},
		{Key: "category", Value: 1},
		{Key: "indicator_id", Value: 1},
	}))
	if err != nil {
		return nil, errb.Wrapf(err, "query macro indicators mongodb")
	}
	defer cursor.Close(ctx)
	var documents []macroIndicatorMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode macro indicators mongodb")
	}
	indicators := make([]macrorole.Indicator, 0, len(documents))
	for _, document := range documents {
		indicators = append(indicators, macroIndicatorMongoDocumentToCanonical(document))
	}
	return indicators, nil
}

func (r *mongoRepository) QueryObservations(ctx context.Context, query macroservice.ObservationQuery) ([]macrorole.Observation, error) {
	errb := oops.In("macro_repository").With("backend", "mongodb", "indicator_id", query.IndicatorID, "from", query.From, "to", query.To)
	cursor, err := r.observations.Find(ctx, macroObservationMongoFilter(query), options.Find().SetSort(bson.D{
		{Key: "period", Value: 1},
		{Key: "observation_revision", Value: 1},
	}))
	if err != nil {
		return nil, errb.Wrapf(err, "query macro observations mongodb")
	}
	defer cursor.Close(ctx)
	var documents []macroObservationMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode macro observations mongodb")
	}
	observations := make([]macrorole.Observation, 0, len(documents))
	for _, document := range documents {
		observations = append(observations, macroObservationMongoDocumentToCanonical(document))
	}
	return observations, nil
}

func (r *mongoRepository) UpsertIndicators(ctx context.Context, indicators []macrorole.Indicator) (macroservice.IndicatorWriteResult, error) {
	errb := oops.In("macro_repository").With("backend", "mongodb", "indicators", len(indicators))
	result := macroservice.IndicatorWriteResult{}
	for _, indicator := range indicators {
		indicator = normalizeIndicator(indicator)
		indicatorErrb := errb.With("provider", indicator.Provider, "source_code", indicator.SourceCode, "indicator_id", indicator.ID)
		if err := validateIndicator(indicator); err != nil {
			return macroservice.IndicatorWriteResult{}, indicatorErrb.Wrap(err)
		}
		document, err := macroIndicatorToMongoDocument(indicator)
		if err != nil {
			return macroservice.IndicatorWriteResult{}, indicatorErrb.Wrap(err)
		}
		if err := r.replaceIndicatorDocument(ctx, document); err != nil {
			return macroservice.IndicatorWriteResult{}, indicatorErrb.With("id", document.ID).Wrap(err)
		}
		result.IndicatorsWritten++
		result.SourcesWritten++
		result.RowsAffected += 2
		if indicator.ProviderDoc != nil {
			result.DocumentsWritten++
			result.RowsAffected++
		}
	}
	return result, nil
}

func (r *mongoRepository) UpsertObservations(ctx context.Context, observations []macrorole.Observation) (macroservice.ObservationWriteResult, error) {
	errb := oops.In("macro_repository").With("backend", "mongodb", "observations", len(observations))
	result := macroservice.ObservationWriteResult{}
	for _, observation := range observations {
		observation = normalizeObservation(observation)
		observationErrb := errb.With("indicator_id", observation.IndicatorID, "period", observation.Period, "revision", observation.Revision)
		if err := validateObservation(observation); err != nil {
			return macroservice.ObservationWriteResult{}, observationErrb.Wrap(err)
		}
		if observation.Provider == "" || observation.SourceCode == "" {
			indicator, err := r.lookupIndicator(ctx, observation.IndicatorID)
			if err != nil {
				return macroservice.ObservationWriteResult{}, observationErrb.Wrap(err)
			}
			if observation.Provider == "" {
				observation.Provider = indicator.Provider
			}
			if observation.SourceCode == "" {
				observation.SourceCode = indicator.SourceCode
			}
		}
		document, err := macroObservationToMongoDocument(observation)
		if err != nil {
			return macroservice.ObservationWriteResult{}, observationErrb.Wrap(err)
		}
		if err := r.upsertObservationDocument(ctx, document); err != nil {
			return macroservice.ObservationWriteResult{}, observationErrb.With("id", document.ID).Wrap(err)
		}
		result.ObservationsWritten++
		result.RowsAffected++
	}
	return result, nil
}

func (r *mongoRepository) replaceIndicatorDocument(ctx context.Context, next macroIndicatorMongoDocument) error {
	var current macroIndicatorMongoDocument
	err := r.indicators.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if _, insertErr := r.indicators.InsertOne(ctx, next); insertErr != nil {
			return oops.In("macro_repository").With("id", next.ID).Wrapf(insertErr, "insert macro indicator mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("macro_repository").With("id", next.ID).Wrapf(err, "read macro indicator mongodb document")
	}
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	next.Sources = replaceMacroSource(current.Sources, next.Sources[0])
	if len(next.ProviderDocs) > 0 {
		next.ProviderDocs = replaceMacroProviderDoc(current.ProviderDocs, next.ProviderDocs[0])
	} else {
		next.ProviderDocs = current.ProviderDocs
	}
	updateResult, err := r.indicators.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("macro_repository").With("id", next.ID, "revision", current.Revision).Wrapf(err, "replace macro indicator mongodb document")
	}
	if updateResult.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("macro_indicators", next.ID, current.Revision)
	}
	return nil
}

func (r *mongoRepository) upsertObservationDocument(ctx context.Context, document macroObservationMongoDocument) error {
	filter := bson.D{{Key: "_id", Value: document.ID}}
	update := bson.D{
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: document.ID},
			{Key: "created_at", Value: document.CreatedAt},
		}},
		{Key: "$set", Value: bson.D{
			{Key: "schema_version", Value: document.SchemaVersion},
			{Key: "updated_at", Value: document.UpdatedAt},
			{Key: "collected_at", Value: document.CollectedAt},
			{Key: "indicator_id", Value: document.IndicatorID},
			{Key: "provider", Value: document.Provider},
			{Key: "source_code", Value: document.SourceCode},
			{Key: "period", Value: document.Period},
			{Key: "value", Value: document.Value},
			{Key: "published_at", Value: document.PublishedAt},
			{Key: "collected_at_text", Value: document.CollectedAtText},
			{Key: "observation_revision", Value: document.ObservationRevision},
		}},
		{Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}},
	}
	if _, err := r.observations.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true)); err != nil {
		return oops.In("macro_repository").With("id", document.ID).Wrapf(err, "upsert macro observation mongodb document")
	}
	return nil
}

func (r *mongoRepository) lookupIndicator(ctx context.Context, indicatorID string) (macrorole.Indicator, error) {
	var document macroIndicatorMongoDocument
	err := r.indicators.FindOne(ctx, bson.D{{Key: "indicator_id", Value: strings.TrimSpace(indicatorID)}}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return macrorole.Indicator{}, oops.In("macro_repository").With("indicator_id", indicatorID).New("macro indicator metadata not found")
	}
	if err != nil {
		return macrorole.Indicator{}, oops.In("macro_repository").With("indicator_id", indicatorID).Wrapf(err, "read macro indicator mongodb document")
	}
	return macroIndicatorMongoDocumentToCanonical(document), nil
}

func macroIndicatorToMongoDocument(indicator macrorole.Indicator) (macroIndicatorMongoDocument, error) {
	now := storagemongodb.ISOTimeNow()
	source := macroIndicatorMongoSource{
		Provider:   string(indicator.Provider),
		SourceCode: indicator.SourceCode,
		SourceName: indicator.SourceName,
		SourceURL:  indicator.SourceURL,
	}
	document := macroIndicatorMongoDocument{
		ID:            macroIndicatorMongoID(indicator.ID),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		IndicatorID:   indicator.ID,
		Preset:        string(indicator.Preset),
		Provider:      string(indicator.Provider),
		SourceCode:    indicator.SourceCode,
		Name:          indicator.Name,
		FriendlyName:  indicator.FriendlyName,
		Category:      indicator.Category,
		Frequency:     string(indicator.Frequency),
		Unit:          indicator.Unit,
		Scale:         indicator.Scale,
		Active:        indicator.Active,
		Sources:       []macroIndicatorMongoSource{source},
	}
	if indicator.ProviderDoc != nil {
		providerDoc, err := macroProviderDocumentToMongo(indicator.ProviderDoc, now)
		if err != nil {
			return macroIndicatorMongoDocument{}, err
		}
		document.ProviderDocs = []macroProviderMongoDocument{providerDoc}
	}
	return document, nil
}

func macroProviderDocumentToMongo(document *macrorole.ProviderDocument, now time.Time) (macroProviderMongoDocument, error) {
	schemaVersion := strings.TrimSpace(document.SchemaVersion)
	if schemaVersion == "" {
		schemaVersion = "1.0.0"
	}
	if _, err := encodeDocument(document.Document); err != nil {
		return macroProviderMongoDocument{}, err
	}
	return macroProviderMongoDocument{
		Provider:      string(document.Provider),
		SchemaVersion: schemaVersion,
		Document:      document.Document,
		UpdatedAt:     strings.TrimSpace(document.UpdatedAt),
		UpdatedAtMS:   now.UTC().UnixMilli(),
	}, nil
}

func macroObservationToMongoDocument(observation macrorole.Observation) (macroObservationMongoDocument, error) {
	collectedAt, err := parseMacroCollectedAt(observation.CollectedAt)
	if err != nil {
		return macroObservationMongoDocument{}, err
	}
	now := storagemongodb.ISOTimeNow()
	return macroObservationMongoDocument{
		ID:                  macroObservationMongoID(observation.IndicatorID, observation.Period, observation.Revision),
		SchemaVersion:       storagemongodb.SchemaVersion1,
		Revision:            1,
		CreatedAt:           now,
		UpdatedAt:           now,
		CollectedAt:         collectedAt,
		IndicatorID:         observation.IndicatorID,
		Provider:            string(observation.Provider),
		SourceCode:          observation.SourceCode,
		Period:              observation.Period,
		Value:               observation.Value,
		PublishedAt:         observation.PublishedAt,
		CollectedAtText:     observation.CollectedAt,
		ObservationRevision: observation.Revision,
	}, nil
}

func macroIndicatorMongoFilter(query macroservice.IndicatorQuery) bson.D {
	filter := bson.D{}
	if query.ProviderID != "" {
		filter = append(filter, bson.E{Key: "provider", Value: string(query.ProviderID)})
	}
	if query.Preset != "" {
		filter = append(filter, bson.E{Key: "preset", Value: string(query.Preset)})
	}
	if strings.TrimSpace(query.IndicatorID) != "" {
		filter = append(filter, bson.E{Key: "indicator_id", Value: strings.TrimSpace(query.IndicatorID)})
	}
	return filter
}

func macroObservationMongoFilter(query macroservice.ObservationQuery) bson.D {
	filter := bson.D{{Key: "indicator_id", Value: strings.TrimSpace(query.IndicatorID)}}
	rangeFilter := bson.D{}
	if strings.TrimSpace(query.From) != "" {
		rangeFilter = append(rangeFilter, bson.E{Key: "$gte", Value: strings.TrimSpace(query.From)})
	}
	if strings.TrimSpace(query.To) != "" {
		rangeFilter = append(rangeFilter, bson.E{Key: "$lte", Value: strings.TrimSpace(query.To)})
	}
	if len(rangeFilter) > 0 {
		filter = append(filter, bson.E{Key: "period", Value: rangeFilter})
	}
	return filter
}

func macroIndicatorMongoDocumentToCanonical(document macroIndicatorMongoDocument) macrorole.Indicator {
	source := firstMacroSource(document)
	return macrorole.Indicator{
		ID:           document.IndicatorID,
		Preset:       macrorole.Preset(document.Preset),
		Provider:     provider.ProviderID(document.Provider),
		SourceCode:   document.SourceCode,
		SourceName:   source.SourceName,
		SourceURL:    source.SourceURL,
		Name:         document.Name,
		FriendlyName: document.FriendlyName,
		Category:     document.Category,
		Frequency:    macrorole.Frequency(document.Frequency),
		Unit:         document.Unit,
		Scale:        document.Scale,
		Active:       document.Active,
		ProviderDoc:  firstMacroProviderDoc(document),
	}
}

func macroObservationMongoDocumentToCanonical(document macroObservationMongoDocument) macrorole.Observation {
	return macrorole.Observation{
		IndicatorID: document.IndicatorID,
		Provider:    provider.ProviderID(document.Provider),
		SourceCode:  document.SourceCode,
		Period:      document.Period,
		Value:       document.Value,
		PublishedAt: document.PublishedAt,
		CollectedAt: document.CollectedAtText,
		Revision:    document.ObservationRevision,
	}
}

func firstMacroSource(document macroIndicatorMongoDocument) macroIndicatorMongoSource {
	for _, source := range document.Sources {
		if source.Provider == document.Provider && source.SourceCode == document.SourceCode {
			return source
		}
	}
	if len(document.Sources) > 0 {
		return document.Sources[0]
	}
	return macroIndicatorMongoSource{}
}

func firstMacroProviderDoc(document macroIndicatorMongoDocument) *macrorole.ProviderDocument {
	if len(document.ProviderDocs) == 0 {
		return nil
	}
	stored := document.ProviderDocs[0]
	return &macrorole.ProviderDocument{
		IndicatorID:   document.IndicatorID,
		Provider:      provider.ProviderID(stored.Provider),
		SchemaVersion: stored.SchemaVersion,
		Document:      stored.Document,
		UpdatedAt:     stored.UpdatedAt,
	}
}

func replaceMacroSource(sources []macroIndicatorMongoSource, next macroIndicatorMongoSource) []macroIndicatorMongoSource {
	replaced := false
	out := make([]macroIndicatorMongoSource, 0, len(sources)+1)
	for _, source := range sources {
		if source.Provider == next.Provider && source.SourceCode == next.SourceCode {
			out = append(out, next)
			replaced = true
			continue
		}
		out = append(out, source)
	}
	if !replaced {
		out = append(out, next)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.Join([]string{out[i].Provider, out[i].SourceCode}, "\x00") < strings.Join([]string{out[j].Provider, out[j].SourceCode}, "\x00")
	})
	return out
}

func replaceMacroProviderDoc(documents []macroProviderMongoDocument, next macroProviderMongoDocument) []macroProviderMongoDocument {
	replaced := false
	out := make([]macroProviderMongoDocument, 0, len(documents)+1)
	for _, document := range documents {
		if document.Provider == next.Provider && document.SchemaVersion == next.SchemaVersion {
			out = append(out, next)
			replaced = true
			continue
		}
		out = append(out, document)
	}
	if !replaced {
		out = append(out, next)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.Join([]string{out[i].Provider, out[i].SchemaVersion}, "\x00") < strings.Join([]string{out[j].Provider, out[j].SchemaVersion}, "\x00")
	})
	return out
}

func parseMacroCollectedAt(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, oops.In("macro_repository").With("collected_at", value).New("macro observation collected_at must be RFC3339 or YYYY-MM-DD")
}

func macroIndicatorMongoID(indicatorID string) string {
	return strings.Join([]string{"macro_indicators", strings.TrimSpace(indicatorID)}, ":")
}

func macroObservationMongoID(indicatorID string, period string, revision int) string {
	return strings.Join([]string{"macro_observations", strings.TrimSpace(indicatorID), strings.TrimSpace(period), strconvItoa(revision)}, ":")
}

func strconvItoa(value int) string {
	return strconv.Itoa(value)
}
