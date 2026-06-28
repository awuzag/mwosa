package composition

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	compositionrole "github.com/awuzag/mwosa/providers/core/composition"
	compositionservice "github.com/awuzag/mwosa/service/composition"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	collection *mongo.Collection
}

type compositionMongoDocument struct {
	ID            string    `bson:"_id"`
	SchemaVersion string    `bson:"schema_version"`
	Revision      int64     `bson:"revision"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`

	SubjectKey   string                     `bson:"subject_key"`
	Subject      compositionMongoInstrument `bson:"subject"`
	Source       compositionMongoSource     `bson:"source"`
	AsOfDate     string                     `bson:"as_of_date"`
	ObservedAtMS int64                      `bson:"observed_at_ms"`
	Members      []compositionMongoMember   `bson:"members"`
}

type compositionMongoSource struct {
	Provider      string `bson:"provider"`
	ProviderGroup string `bson:"provider_group"`
	Operation     string `bson:"operation"`
}

type compositionMongoInstrument struct {
	Market       string `bson:"market"`
	SecurityType string `bson:"security_type"`
	Symbol       string `bson:"symbol"`
	ISIN         string `bson:"isin,omitempty"`
	Name         string `bson:"name,omitempty"`
}

type compositionMongoMember struct {
	Ordinal           int                        `bson:"ordinal"`
	InstrumentKey     string                     `bson:"instrument_key"`
	Market            string                     `bson:"market"`
	SecurityType      string                     `bson:"security_type"`
	Symbol            string                     `bson:"symbol"`
	ISIN              string                     `bson:"isin,omitempty"`
	Name              string                     `bson:"name,omitempty"`
	WeightValue       string                     `bson:"weight_value,omitempty"`
	QuantityValue     string                     `bson:"quantity_value,omitempty"`
	ValuationCurrency string                     `bson:"valuation_currency,omitempty"`
	ValuationValue    string                     `bson:"valuation_value,omitempty"`
	Instrument        compositionMongoInstrument `bson:"instrument"`
}

var _ compositionservice.Repository = (*mongoRepository)(nil)

func NewMongoRepository(database *mongo.Database) (compositionservice.Repository, error) {
	if database == nil {
		return nil, oops.In("composition_repository").New("mongodb database is nil")
	}
	return &mongoRepository{collection: database.Collection("compositions")}, nil
}

func (r *mongoRepository) UpsertComposition(ctx context.Context, aggregate compositionrole.Composition) (compositionservice.WriteResult, error) {
	errb := oops.In("composition_repository").With(
		"backend", "mongodb",
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
	document := compositionToMongoDocument(aggregate)
	if err := r.replaceDocument(ctx, document); err != nil {
		return compositionservice.WriteResult{}, errb.With("id", document.ID).Wrap(err)
	}
	return compositionservice.WriteResult{
		RowsAffected:       1 + len(aggregate.Members),
		CompositionsStored: 1,
		MembersStored:      len(aggregate.Members),
	}, nil
}

func (r *mongoRepository) GetComposition(ctx context.Context, query compositionservice.Query) (compositionrole.Composition, error) {
	errb := oops.In("composition_repository").With("backend", "mongodb", "provider", query.ProviderID, "market", query.Market, "security_type", query.SecurityType, "symbol", query.Symbol, "as_of_date", query.AsOfDate, "observed_at_ms", query.ObservedAtMS)
	if strings.TrimSpace(query.Symbol) == "" {
		return compositionrole.Composition{}, errb.New("get composition requires symbol")
	}
	filter := compositionMongoFilter(query)
	var document compositionMongoDocument
	err := r.collection.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "as_of_date", Value: -1}, {Key: "observed_at_ms", Value: -1}})).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return compositionrole.Composition{}, &compositionservice.NotFoundError{
			Symbol:       strings.TrimSpace(query.Symbol),
			Market:       query.Market,
			SecurityType: query.SecurityType,
			AsOfDate:     query.AsOfDate,
		}
	}
	if err != nil {
		return compositionrole.Composition{}, errb.Wrapf(err, "get composition mongodb document")
	}
	return mongoDocumentToComposition(document), nil
}

func (r *mongoRepository) replaceDocument(ctx context.Context, next compositionMongoDocument) error {
	var current compositionMongoDocument
	err := r.collection.FindOne(ctx, bson.D{{Key: "_id", Value: next.ID}}).Decode(&current)
	if errors.Is(err, mongo.ErrNoDocuments) {
		if _, insertErr := r.collection.InsertOne(ctx, next); insertErr != nil {
			return oops.In("composition_repository").With("id", next.ID).Wrapf(insertErr, "insert composition mongodb document")
		}
		return nil
	}
	if err != nil {
		return oops.In("composition_repository").With("id", next.ID).Wrapf(err, "read composition mongodb document")
	}
	next.CreatedAt = current.CreatedAt
	next.Revision = current.Revision + 1
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: current.Revision}}, next)
	if err != nil {
		return oops.In("composition_repository").With("id", next.ID, "revision", current.Revision).Wrapf(err, "replace composition mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("compositions", next.ID, current.Revision)
	}
	return nil
}

func compositionToMongoDocument(aggregate compositionrole.Composition) compositionMongoDocument {
	now := storagemongodb.ISOTimeNow()
	subject := compositionInstrumentToMongo(aggregate.Subject)
	source := compositionMongoSource{
		Provider:      string(aggregate.Source.Provider),
		ProviderGroup: string(aggregate.Source.Group),
		Operation:     string(aggregate.Source.Operation),
	}
	return compositionMongoDocument{
		ID:            compositionMongoID(subject, source, aggregate.AsOfDate, aggregate.ObservedAtMS),
		SchemaVersion: storagemongodb.SchemaVersion1,
		Revision:      1,
		CreatedAt:     now,
		UpdatedAt:     now,
		SubjectKey:    compositionInstrumentKey(subject.Market, subject.SecurityType, subject.Symbol),
		Subject:       subject,
		Source:        source,
		AsOfDate:      strings.TrimSpace(aggregate.AsOfDate),
		ObservedAtMS:  aggregate.ObservedAtMS,
		Members:       compositionMembersToMongo(aggregate.Members),
	}
}

func compositionMembersToMongo(members []compositionrole.CompositionMember) []compositionMongoMember {
	out := make([]compositionMongoMember, 0, len(members))
	for index, member := range members {
		instrument := compositionInstrumentToMongo(member.Instrument)
		out = append(out, compositionMongoMember{
			Ordinal:           index,
			InstrumentKey:     compositionInstrumentKey(instrument.Market, instrument.SecurityType, instrument.Symbol),
			Market:            instrument.Market,
			SecurityType:      instrument.SecurityType,
			Symbol:            instrument.Symbol,
			ISIN:              instrument.ISIN,
			Name:              instrument.Name,
			WeightValue:       strings.TrimSpace(member.Weight.Value),
			QuantityValue:     strings.TrimSpace(member.Quantity.Value),
			ValuationCurrency: strings.TrimSpace(member.Valuation.Currency),
			ValuationValue:    strings.TrimSpace(member.Valuation.Value),
			Instrument:        instrument,
		})
	}
	return out
}

func mongoDocumentToComposition(document compositionMongoDocument) compositionrole.Composition {
	members := make([]compositionrole.CompositionMember, 0, len(document.Members))
	for _, member := range document.Members {
		members = append(members, compositionrole.CompositionMember{
			Instrument: compositionrole.InstrumentRef{
				Market:       provider.Market(firstNonEmpty(member.Instrument.Market, member.Market)),
				SecurityType: provider.SecurityType(firstNonEmpty(member.Instrument.SecurityType, member.SecurityType)),
				Symbol:       firstNonEmpty(member.Instrument.Symbol, member.Symbol),
				ISIN:         firstNonEmpty(member.Instrument.ISIN, member.ISIN),
				Name:         firstNonEmpty(member.Instrument.Name, member.Name),
			},
			Weight: compositionrole.DecimalValue{
				Value: member.WeightValue,
			},
			Quantity: compositionrole.DecimalValue{
				Value: member.QuantityValue,
			},
			Valuation: compositionrole.MoneyValue{
				Currency: member.ValuationCurrency,
				Value:    member.ValuationValue,
			},
		})
	}
	return compositionrole.Composition{
		Source: compositionrole.SourceRef{
			Provider:  provider.ProviderID(document.Source.Provider),
			Group:     provider.GroupID(document.Source.ProviderGroup),
			Operation: provider.OperationID(document.Source.Operation),
		},
		Subject: compositionrole.InstrumentRef{
			Market:       provider.Market(document.Subject.Market),
			SecurityType: provider.SecurityType(document.Subject.SecurityType),
			Symbol:       document.Subject.Symbol,
			ISIN:         document.Subject.ISIN,
			Name:         document.Subject.Name,
		},
		AsOfDate:     document.AsOfDate,
		ObservedAtMS: document.ObservedAtMS,
		Members:      members,
	}
}

func compositionMongoFilter(query compositionservice.Query) bson.D {
	filter := bson.D{
		{Key: "subject.market", Value: string(marketWithDefault(query.Market))},
		{Key: "subject.symbol", Value: strings.TrimSpace(query.Symbol)},
	}
	if query.SecurityType != "" {
		filter = append(filter, bson.E{Key: "subject.security_type", Value: string(query.SecurityType)})
	}
	if query.ProviderID != "" {
		filter = append(filter, bson.E{Key: "source.provider", Value: string(query.ProviderID)})
	}
	if strings.TrimSpace(query.AsOfDate) != "" {
		filter = append(filter, bson.E{Key: "as_of_date", Value: strings.TrimSpace(query.AsOfDate)})
	}
	if query.ObservedAtMS != 0 {
		filter = append(filter, bson.E{Key: "observed_at_ms", Value: query.ObservedAtMS})
	}
	return filter
}

func compositionInstrumentToMongo(ref compositionrole.InstrumentRef) compositionMongoInstrument {
	market := string(marketWithDefault(ref.Market))
	return compositionMongoInstrument{
		Market:       market,
		SecurityType: string(ref.SecurityType),
		Symbol:       strings.TrimSpace(ref.Symbol),
		ISIN:         strings.TrimSpace(ref.ISIN),
		Name:         strings.TrimSpace(ref.Name),
	}
}

func compositionInstrumentKey(market string, securityType string, symbol string) string {
	return strings.Join([]string{market, securityType, symbol}, ":")
}

func compositionMongoID(subject compositionMongoInstrument, source compositionMongoSource, asOfDate string, observedAtMS int64) string {
	return strings.Join([]string{
		"compositions",
		subject.Market,
		subject.SecurityType,
		subject.Symbol,
		source.Provider,
		source.ProviderGroup,
		source.Operation,
		strings.TrimSpace(asOfDate),
		strconvFormatInt(observedAtMS),
	}, ":")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
