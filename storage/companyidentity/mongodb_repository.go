package companyidentity

import (
	"context"
	"encoding/binary"
	"hash/fnv"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	provider "github.com/awuzag/mwosa/providers/core"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoRepository struct {
	collection *mongo.Collection
}

type companyMongoDocument struct {
	ID            string                    `bson:"_id"`
	SchemaVersion string                    `bson:"schema_version"`
	Revision      int64                     `bson:"revision"`
	CreatedAt     time.Time                 `bson:"created_at"`
	UpdatedAt     time.Time                 `bson:"updated_at"`
	CompanyID     int64                     `bson:"company_id"`
	Name          string                    `bson:"name"`
	LegalName     string                    `bson:"legal_name,omitempty"`
	EnglishName   string                    `bson:"english_name,omitempty"`
	CountryCode   string                    `bson:"country_code,omitempty"`
	Identifiers   []identifierMongoDocument `bson:"identifiers"`
	Instruments   []instrumentMongoDocument `bson:"instruments"`
}

type identifierMongoDocument struct {
	Provider        string    `bson:"provider"`
	ProviderGroup   string    `bson:"provider_group"`
	Operation       string    `bson:"operation"`
	IdentifierType  string    `bson:"identifier_type"`
	IdentifierValue string    `bson:"identifier_value"`
	ValidFrom       string    `bson:"valid_from"`
	ValidTo         string    `bson:"valid_to,omitempty"`
	Primary         bool      `bson:"primary"`
	Confidence      float64   `bson:"confidence"`
	SourceUpdatedAt string    `bson:"source_updated_at,omitempty"`
	CreatedAt       time.Time `bson:"created_at"`
	UpdatedAt       time.Time `bson:"updated_at"`
}

type instrumentMongoDocument struct {
	InstrumentID  int64     `bson:"instrument_id"`
	InstrumentKey string    `bson:"instrument_key"`
	Market        string    `bson:"market"`
	SecurityType  string    `bson:"security_type"`
	Symbol        string    `bson:"symbol"`
	Name          string    `bson:"name,omitempty"`
	RelationType  string    `bson:"relation_type"`
	ValidFrom     string    `bson:"valid_from"`
	ValidTo       string    `bson:"valid_to,omitempty"`
	CreatedAt     time.Time `bson:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at"`
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("company_identity_repository").New("mongodb database is nil")
	}
	return MongoRepository{collection: database.Collection("companies")}, nil
}

func (r MongoRepository) UpsertCompanies(ctx context.Context, companies []CompanyInput) (UpsertResult, error) {
	errb := oops.In("company_identity_repository").With("backend", "mongodb", "companies", len(companies))
	var result UpsertResult
	for _, input := range companies {
		document, err := r.resolveCompanyDocument(ctx, input)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		next, identifiersWritten, instrumentsLinked, err := mergeCompanyInput(document, input)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		if err := r.replaceDocument(ctx, next, document != nil); err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		result.CompaniesWritten++
		result.IdentifiersWritten += identifiersWritten
		result.InstrumentsLinked += instrumentsLinked
	}
	return result, nil
}

func (r MongoRepository) Inspect(ctx context.Context, query string) (InspectResult, error) {
	trimmed := strings.TrimSpace(query)
	errb := oops.In("company_identity_repository").With("backend", "mongodb", "query", trimmed)
	if trimmed == "" {
		return InspectResult{}, errb.New("inspect company requires query")
	}
	documents, err := r.findCompanyDocuments(ctx, trimmed)
	if err != nil {
		return InspectResult{}, errb.Wrap(err)
	}
	if len(documents) == 0 {
		return InspectResult{}, errb.New("company not found")
	}
	if len(documents) > 1 {
		return InspectResult{}, errb.With("matches", len(documents)).New("company query matched multiple companies")
	}
	return inspectResultFromMongoDocument(documents[0]), nil
}

func (r MongoRepository) resolveCompanyDocument(ctx context.Context, input CompanyInput) (*companyMongoDocument, error) {
	for _, identifier := range input.Identifiers {
		identifierType := strings.TrimSpace(identifier.IdentifierType)
		identifierValue := strings.TrimSpace(identifier.IdentifierValue)
		if identifierType == "" || identifierValue == "" {
			continue
		}
		var document companyMongoDocument
		err := r.collection.FindOne(ctx, bson.D{
			{Key: "identifiers.identifier_type", Value: identifierType},
			{Key: "identifiers.identifier_value", Value: identifierValue},
		}).Decode(&document)
		if err == nil {
			return &document, nil
		}
		if err != mongo.ErrNoDocuments {
			return nil, oops.In("company_identity_repository").With("backend", "mongodb", "identifier_type", identifierType, "identifier_value", identifierValue).Wrapf(err, "select company identifier")
		}
	}
	return nil, nil
}

func (r MongoRepository) replaceDocument(ctx context.Context, next companyMongoDocument, existing bool) error {
	if !existing {
		if _, err := r.collection.InsertOne(ctx, next); err != nil {
			return oops.In("company_identity_repository").With("backend", "mongodb", "company_id", next.CompanyID).Wrapf(err, "insert company mongodb document")
		}
		return nil
	}
	result, err := r.collection.ReplaceOne(ctx, bson.D{{Key: "_id", Value: next.ID}, {Key: "revision", Value: next.Revision - 1}}, next)
	if err != nil {
		return oops.In("company_identity_repository").With("backend", "mongodb", "company_id", next.CompanyID, "revision", next.Revision-1).Wrapf(err, "replace company mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("companies", next.ID, next.Revision-1)
	}
	return nil
}

func (r MongoRepository) findCompanyDocuments(ctx context.Context, query string) ([]companyMongoDocument, error) {
	filter := bson.D{{Key: "$or", Value: bson.A{
		bson.D{{Key: "identifiers.identifier_value", Value: query}},
		bson.D{{Key: "name", Value: containsRegex(query)}},
		bson.D{{Key: "legal_name", Value: containsRegex(query)}},
		bson.D{{Key: "english_name", Value: containsRegex(query)}},
	}}}
	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}).SetLimit(2))
	if err != nil {
		return nil, oops.In("company_identity_repository").With("backend", "mongodb", "query", query).Wrapf(err, "query company")
	}
	defer cursor.Close(ctx)
	var documents []companyMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, oops.In("company_identity_repository").With("backend", "mongodb", "query", query).Wrapf(err, "decode company")
	}
	sort.SliceStable(documents, func(i, j int) bool {
		return companyRank(documents[i], query) < companyRank(documents[j], query)
	})
	return documents, nil
}

func mergeCompanyInput(current *companyMongoDocument, input CompanyInput) (companyMongoDocument, int, int, error) {
	name := firstNonEmpty(input.Name, input.LegalName, input.EnglishName)
	if name == "" {
		return companyMongoDocument{}, 0, 0, oops.In("company_identity_repository").New("company identity row missing name")
	}
	now := storagemongodb.ISOTimeNow()
	var document companyMongoDocument
	if current == nil {
		companyID := stableInt64(companyNaturalKey(input))
		document = companyMongoDocument{
			ID:            companyMongoID(companyID),
			SchemaVersion: storagemongodb.SchemaVersion1,
			Revision:      1,
			CreatedAt:     now,
			UpdatedAt:     now,
			CompanyID:     companyID,
			Identifiers:   []identifierMongoDocument{},
			Instruments:   []instrumentMongoDocument{},
		}
	} else {
		document = *current
		document.Revision++
		document.UpdatedAt = now
	}
	document.Name = name
	document.LegalName = strings.TrimSpace(input.LegalName)
	document.EnglishName = strings.TrimSpace(input.EnglishName)
	document.CountryCode = firstNonEmpty(input.CountryCode, "KR")

	identifiersWritten := 0
	for _, identifier := range input.Identifiers {
		next, err := identifierToMongoDocument(identifier, now)
		if err != nil {
			return companyMongoDocument{}, 0, 0, err
		}
		document.Identifiers = upsertIdentifierDocument(document.Identifiers, next)
		identifiersWritten++
	}

	instrumentsLinked := 0
	if strings.TrimSpace(input.InstrumentRef.Symbol) != "" {
		next := instrumentToMongoDocument(input.InstrumentRef, now)
		document.Instruments = upsertInstrumentDocument(document.Instruments, next)
		instrumentsLinked = 1
	}
	sortCompanyDocument(&document)
	return document, identifiersWritten, instrumentsLinked, nil
}

func identifierToMongoDocument(input IdentifierInput, now time.Time) (identifierMongoDocument, error) {
	identifierType := strings.TrimSpace(input.IdentifierType)
	identifierValue := strings.TrimSpace(input.IdentifierValue)
	if identifierType == "" || identifierValue == "" {
		return identifierMongoDocument{}, oops.In("company_identity_repository").New("company identifier missing type or value")
	}
	confidence := input.Confidence
	if confidence == 0 {
		confidence = 1
	}
	return identifierMongoDocument{
		Provider:        string(input.Provider),
		ProviderGroup:   string(input.Group),
		Operation:       string(input.Operation),
		IdentifierType:  identifierType,
		IdentifierValue: identifierValue,
		Primary:         input.Primary,
		Confidence:      confidence,
		SourceUpdatedAt: strings.TrimSpace(input.SourceUpdatedAt),
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func instrumentToMongoDocument(input InstrumentRef, now time.Time) instrumentMongoDocument {
	market := string(withDefaultMarket(input.Market))
	securityType := input.SecurityType
	if securityType == "" {
		securityType = provider.SecurityTypeStock
	}
	symbol := strings.TrimSpace(input.Symbol)
	relationType := firstNonEmpty(input.RelationType, RelationTypeIssuer)
	key := instrumentKey(market, string(securityType), symbol)
	return instrumentMongoDocument{
		InstrumentID:  stableInt64(key),
		InstrumentKey: key,
		Market:        market,
		SecurityType:  string(securityType),
		Symbol:        symbol,
		Name:          strings.TrimSpace(input.Name),
		RelationType:  relationType,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func upsertIdentifierDocument(items []identifierMongoDocument, next identifierMongoDocument) []identifierMongoDocument {
	for i := range items {
		if sameIdentifier(items[i], next) {
			next.CreatedAt = items[i].CreatedAt
			items[i] = next
			return items
		}
	}
	return append(items, next)
}

func upsertInstrumentDocument(items []instrumentMongoDocument, next instrumentMongoDocument) []instrumentMongoDocument {
	for i := range items {
		if items[i].InstrumentKey == next.InstrumentKey && items[i].RelationType == next.RelationType && items[i].ValidFrom == next.ValidFrom {
			next.CreatedAt = items[i].CreatedAt
			items[i] = next
			return items
		}
	}
	return append(items, next)
}

func inspectResultFromMongoDocument(document companyMongoDocument) InspectResult {
	identifiers := make([]Identifier, 0, len(document.Identifiers))
	for _, item := range document.Identifiers {
		identifiers = append(identifiers, Identifier{
			CompanyID:       document.CompanyID,
			Provider:        provider.ProviderID(item.Provider),
			Group:           provider.GroupID(item.ProviderGroup),
			Operation:       provider.OperationID(item.Operation),
			IdentifierType:  item.IdentifierType,
			IdentifierValue: item.IdentifierValue,
			Primary:         item.Primary,
			Confidence:      item.Confidence,
			SourceUpdatedAt: item.SourceUpdatedAt,
		})
	}
	sort.SliceStable(identifiers, func(i, j int) bool {
		if identifiers[i].Primary != identifiers[j].Primary {
			return identifiers[i].Primary
		}
		if identifiers[i].IdentifierType != identifiers[j].IdentifierType {
			return identifiers[i].IdentifierType < identifiers[j].IdentifierType
		}
		return identifiers[i].Provider < identifiers[j].Provider
	})

	instruments := make([]InstrumentLink, 0, len(document.Instruments))
	for _, item := range document.Instruments {
		instruments = append(instruments, InstrumentLink{
			CompanyID:    document.CompanyID,
			InstrumentID: item.InstrumentID,
			Market:       provider.Market(item.Market),
			SecurityType: provider.SecurityType(item.SecurityType),
			Symbol:       item.Symbol,
			Name:         item.Name,
			RelationType: item.RelationType,
		})
	}
	sort.SliceStable(instruments, func(i, j int) bool {
		if instruments[i].RelationType != instruments[j].RelationType {
			return instruments[i].RelationType < instruments[j].RelationType
		}
		if instruments[i].Market != instruments[j].Market {
			return instruments[i].Market < instruments[j].Market
		}
		return instruments[i].Symbol < instruments[j].Symbol
	})

	return InspectResult{
		Company: Company{
			ID:          document.CompanyID,
			Name:        document.Name,
			LegalName:   document.LegalName,
			EnglishName: document.EnglishName,
			CountryCode: document.CountryCode,
		},
		Identifiers: identifiers,
		Instruments: instruments,
	}
}

func sortCompanyDocument(document *companyMongoDocument) {
	sort.SliceStable(document.Identifiers, func(i, j int) bool {
		if document.Identifiers[i].Primary != document.Identifiers[j].Primary {
			return document.Identifiers[i].Primary
		}
		if document.Identifiers[i].IdentifierType != document.Identifiers[j].IdentifierType {
			return document.Identifiers[i].IdentifierType < document.Identifiers[j].IdentifierType
		}
		return document.Identifiers[i].Provider < document.Identifiers[j].Provider
	})
	sort.SliceStable(document.Instruments, func(i, j int) bool {
		if document.Instruments[i].RelationType != document.Instruments[j].RelationType {
			return document.Instruments[i].RelationType < document.Instruments[j].RelationType
		}
		if document.Instruments[i].Market != document.Instruments[j].Market {
			return document.Instruments[i].Market < document.Instruments[j].Market
		}
		return document.Instruments[i].Symbol < document.Instruments[j].Symbol
	})
}

func sameIdentifier(left identifierMongoDocument, right identifierMongoDocument) bool {
	return left.Provider == right.Provider &&
		left.ProviderGroup == right.ProviderGroup &&
		left.Operation == right.Operation &&
		left.IdentifierType == right.IdentifierType &&
		left.IdentifierValue == right.IdentifierValue &&
		left.ValidFrom == right.ValidFrom
}

func companyRank(document companyMongoDocument, query string) int {
	for _, identifier := range document.Identifiers {
		if identifier.IdentifierValue == query {
			return 0
		}
	}
	if document.Name == query {
		return 1
	}
	return 2
}

func companyNaturalKey(input CompanyInput) string {
	for _, identifier := range input.Identifiers {
		if strings.TrimSpace(identifier.IdentifierType) != "" && strings.TrimSpace(identifier.IdentifierValue) != "" {
			return strings.Join([]string{strings.TrimSpace(identifier.IdentifierType), strings.TrimSpace(identifier.IdentifierValue)}, ":")
		}
	}
	return strings.Join([]string{"name", firstNonEmpty(input.Name, input.LegalName, input.EnglishName)}, ":")
}

func companyMongoID(companyID int64) string {
	return strings.Join([]string{"companies", strconvInt64(companyID)}, ":")
}

func instrumentKey(market string, securityType string, symbol string) string {
	return strings.Join([]string{market, securityType, strings.TrimSpace(symbol)}, ":")
}

func stableInt64(value string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(value))
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], hash.Sum64())
	out := int64(binary.BigEndian.Uint64(buf[:]) & 0x7fffffffffffffff)
	if out == 0 {
		return 1
	}
	return out
}

func strconvInt64(value int64) string {
	return strconv.FormatInt(value, 10)
}

func containsRegex(value string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(value), "$options": "i"}
}
