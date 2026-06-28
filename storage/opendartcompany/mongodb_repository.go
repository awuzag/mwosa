package opendartcompany

import (
	"context"
	"regexp"
	"sort"
	"strings"

	provider "github.com/awuzag/mwosa/providers/core"
	opendartprovider "github.com/awuzag/mwosa/providers/opendart"
	"github.com/awuzag/mwosa/storage/companyidentity"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type MongoRepository struct {
	companies *mongo.Collection
	identity  companyidentity.MongoRepository
}

type opendartCompanyDocument struct {
	Name        string                       `bson:"name"`
	LegalName   string                       `bson:"legal_name,omitempty"`
	EnglishName string                       `bson:"english_name,omitempty"`
	Identifiers []opendartIdentifierDocument `bson:"identifiers"`
}

type opendartIdentifierDocument struct {
	Provider        string `bson:"provider"`
	ProviderGroup   string `bson:"provider_group"`
	Operation       string `bson:"operation"`
	IdentifierType  string `bson:"identifier_type"`
	IdentifierValue string `bson:"identifier_value"`
	Primary         bool   `bson:"primary"`
	SourceUpdatedAt string `bson:"source_updated_at,omitempty"`
}

func NewMongoRepository(database *mongo.Database) (MongoRepository, error) {
	if database == nil {
		return MongoRepository{}, oops.In("opendart_company_repository").New("mongodb database is nil")
	}
	identity, err := companyidentity.NewMongoRepository(database)
	if err != nil {
		return MongoRepository{}, oops.In("opendart_company_repository").Wrap(err)
	}
	return MongoRepository{
		companies: database.Collection("companies"),
		identity:  identity,
	}, nil
}

func (r MongoRepository) UpsertCompanies(ctx context.Context, companies []opendartprovider.Company) (UpsertResult, error) {
	errb := oops.In("opendart_company_repository").With("backend", "mongodb", "rows", len(companies))
	inputs := make([]companyidentity.CompanyInput, 0, len(companies))
	listedCount := 0
	for _, company := range companies {
		input, listed, err := openDARTCompanyToIdentityInput(company)
		if err != nil {
			return UpsertResult{}, errb.Wrap(err)
		}
		if listed {
			listedCount++
		}
		inputs = append(inputs, input)
	}
	if _, err := r.identity.UpsertCompanies(ctx, inputs); err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	return UpsertResult{RowsAffected: int64(len(inputs)), TotalCount: len(inputs), ListedCount: listedCount}, nil
}

func (r MongoRepository) Search(ctx context.Context, query string, listedOnly bool, limit int) ([]opendartprovider.Company, error) {
	trimmed := strings.TrimSpace(query)
	errb := oops.In("opendart_company_repository").With("backend", "mongodb", "query", trimmed, "listed_only", listedOnly, "limit", limit)
	if trimmed == "" {
		return nil, errb.New("opendart company search requires query")
	}
	if limit <= 0 {
		limit = 20
	}
	filter := opendartCompanySearchFilter(trimmed, listedOnly)
	cursor, err := r.companies.Find(ctx, filter)
	if err != nil {
		return nil, errb.Wrapf(err, "search opendart companies")
	}
	defer cursor.Close(ctx)

	var documents []opendartCompanyDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode opendart company search")
	}
	companies := make([]opendartprovider.Company, 0, len(documents))
	for _, document := range documents {
		company := opendartCompanyFromDocument(document)
		if listedOnly && strings.TrimSpace(company.StockCode) == "" {
			continue
		}
		companies = append(companies, company)
	}
	sort.SliceStable(companies, func(i, j int) bool {
		leftListed := strings.TrimSpace(companies[i].StockCode) != ""
		rightListed := strings.TrimSpace(companies[j].StockCode) != ""
		if leftListed != rightListed {
			return leftListed
		}
		return companies[i].CorpName < companies[j].CorpName
	})
	if len(companies) > limit {
		companies = companies[:limit]
	}
	return companies, nil
}

func openDARTCompanyToIdentityInput(company opendartprovider.Company) (companyidentity.CompanyInput, bool, error) {
	corpCode := strings.TrimSpace(company.CorpCode)
	if corpCode == "" {
		return companyidentity.CompanyInput{}, false, oops.In("opendart_company_repository").New("opendart company row missing corp_code")
	}
	corpName := strings.TrimSpace(company.CorpName)
	englishName := strings.TrimSpace(company.CorpEngName)
	modifyDate := strings.TrimSpace(company.ModifyDate)
	stockCode := strings.TrimSpace(company.StockCode)

	input := companyidentity.CompanyInput{
		Name:        corpName,
		LegalName:   corpName,
		EnglishName: englishName,
		CountryCode: "KR",
		Identifiers: []companyidentity.IdentifierInput{
			{
				Provider:        provider.ProviderOpenDART,
				Group:           provider.GroupOpenDARTDisclosure,
				Operation:       provider.OperationOpenDARTCorpCode,
				IdentifierType:  companyidentity.IdentifierTypeDARTCorpCode,
				IdentifierValue: corpCode,
				Primary:         true,
				Confidence:      1,
				SourceUpdatedAt: modifyDate,
			},
		},
	}
	if stockCode == "" {
		return input, false, nil
	}
	input.Identifiers = append(input.Identifiers, companyidentity.IdentifierInput{
		Provider:        provider.ProviderOpenDART,
		Group:           provider.GroupOpenDARTDisclosure,
		Operation:       provider.OperationOpenDARTCorpCode,
		IdentifierType:  companyidentity.IdentifierTypeKRXStockCode,
		IdentifierValue: stockCode,
		Confidence:      1,
		SourceUpdatedAt: modifyDate,
	})
	input.InstrumentRef = companyidentity.InstrumentRef{
		Market:       provider.MarketKRX,
		SecurityType: provider.SecurityTypeStock,
		Symbol:       stockCode,
		Name:         corpName,
		RelationType: companyidentity.RelationTypeIssuer,
	}
	return input, true, nil
}

func opendartCompanySearchFilter(query string, listedOnly bool) bson.D {
	matches := bson.A{
		bson.D{{Key: "identifiers", Value: bson.D{{Key: "$elemMatch", Value: opendartIdentifierValueFilter(query)}}}},
		bson.D{{Key: "name", Value: containsRegex(query)}},
		bson.D{{Key: "legal_name", Value: containsRegex(query)}},
		bson.D{{Key: "english_name", Value: containsRegex(query)}},
	}
	filter := bson.D{{Key: "$or", Value: matches}}
	if !listedOnly {
		return filter
	}
	return bson.D{{Key: "$and", Value: bson.A{
		filter,
		bson.D{{Key: "identifiers", Value: bson.D{{Key: "$elemMatch", Value: opendartStockIdentifierFilter()}}}},
	}}}
}

func opendartIdentifierValueFilter(value string) bson.D {
	filter := opendartIdentifierFilter()
	return append(filter, bson.E{Key: "identifier_value", Value: value})
}

func opendartStockIdentifierFilter() bson.D {
	filter := opendartIdentifierFilter()
	return append(filter,
		bson.E{Key: "identifier_type", Value: companyidentity.IdentifierTypeKRXStockCode},
		bson.E{Key: "identifier_value", Value: bson.D{{Key: "$ne", Value: ""}}},
	)
}

func opendartIdentifierFilter() bson.D {
	return bson.D{
		{Key: "provider", Value: string(provider.ProviderOpenDART)},
		{Key: "provider_group", Value: string(provider.GroupOpenDARTDisclosure)},
		{Key: "operation", Value: string(provider.OperationOpenDARTCorpCode)},
	}
}

func opendartCompanyFromDocument(document opendartCompanyDocument) opendartprovider.Company {
	out := opendartprovider.Company{
		CorpName:    firstNonEmpty(document.LegalName, document.Name),
		CorpEngName: strings.TrimSpace(document.EnglishName),
	}
	for _, identifier := range document.Identifiers {
		if !isOpenDARTCorpCodeIdentifier(identifier) && !isOpenDARTStockCodeIdentifier(identifier) {
			continue
		}
		switch identifier.IdentifierType {
		case companyidentity.IdentifierTypeDARTCorpCode:
			out.CorpCode = strings.TrimSpace(identifier.IdentifierValue)
			out.ModifyDate = strings.TrimSpace(identifier.SourceUpdatedAt)
		case companyidentity.IdentifierTypeKRXStockCode:
			out.StockCode = strings.TrimSpace(identifier.IdentifierValue)
			if out.ModifyDate == "" {
				out.ModifyDate = strings.TrimSpace(identifier.SourceUpdatedAt)
			}
		}
	}
	return out
}

func isOpenDARTCorpCodeIdentifier(identifier opendartIdentifierDocument) bool {
	return isOpenDARTIdentifier(identifier) && identifier.IdentifierType == companyidentity.IdentifierTypeDARTCorpCode
}

func isOpenDARTStockCodeIdentifier(identifier opendartIdentifierDocument) bool {
	return isOpenDARTIdentifier(identifier) && identifier.IdentifierType == companyidentity.IdentifierTypeKRXStockCode
}

func isOpenDARTIdentifier(identifier opendartIdentifierDocument) bool {
	return identifier.Provider == string(provider.ProviderOpenDART) &&
		identifier.ProviderGroup == string(provider.GroupOpenDARTDisclosure) &&
		identifier.Operation == string(provider.OperationOpenDARTCorpCode)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func containsRegex(value string) bson.M {
	return bson.M{"$regex": regexp.QuoteMeta(value), "$options": "i"}
}
