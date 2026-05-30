package opendartcompany

import (
	"context"
	"strings"
	"time"

	opendartprovider "github.com/awuzag/mwosa/providers/opendart"
	"github.com/awuzag/mwosa/storage"
	"github.com/samber/oops"
)

type Repository struct {
	database *storage.Database
}

type UpsertResult struct {
	RowsAffected int64 `json:"rows_affected" csv:"rows_affected"`
	TotalCount   int   `json:"total_count" csv:"total_count"`
	ListedCount  int   `json:"listed_count" csv:"listed_count"`
}

func NewRepository(database *storage.Database) (Repository, error) {
	if database == nil {
		return Repository{}, oops.In("opendart_company_repository").New("opendart company repository database is nil")
	}
	return Repository{database: database}, nil
}

func (r Repository) UpsertCompanies(ctx context.Context, companies []opendartprovider.Company) (UpsertResult, error) {
	errb := oops.In("opendart_company_repository").With("rows", len(companies))
	client, err := r.database.Client(ctx)
	if err != nil {
		return UpsertResult{}, errb.Wrap(err)
	}
	nowMS := time.Now().UTC().UnixMilli()
	var totalAffected int64
	var listedCount int
	for _, company := range companies {
		row := storage.OpenDARTCompanyRow{
			CorpCode:    strings.TrimSpace(company.CorpCode),
			CorpName:    strings.TrimSpace(company.CorpName),
			CorpEngName: strings.TrimSpace(company.CorpEngName),
			StockCode:   strings.TrimSpace(company.StockCode),
			ModifyDate:  strings.TrimSpace(company.ModifyDate),
			Listed:      strings.TrimSpace(company.StockCode) != "",
			CreatedAtMS: nowMS,
			UpdatedAtMS: nowMS,
		}
		if row.CorpCode == "" {
			return UpsertResult{}, errb.New("opendart company row missing corp_code")
		}
		if row.Listed {
			listedCount++
		}
		result, err := client.NewInsert().
			Model(&row).
			On("CONFLICT (corp_code) DO UPDATE").
			Set("corp_name = EXCLUDED.corp_name").
			Set("corp_eng_name = EXCLUDED.corp_eng_name").
			Set("stock_code = EXCLUDED.stock_code").
			Set("modify_date = EXCLUDED.modify_date").
			Set("listed = EXCLUDED.listed").
			Set("updated_at_ms = EXCLUDED.updated_at_ms").
			Exec(ctx)
		if err != nil {
			return UpsertResult{}, errb.With("corp_code", row.CorpCode).Wrapf(err, "upsert opendart company row")
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return UpsertResult{}, errb.With("corp_code", row.CorpCode).Wrapf(err, "read opendart company affected rows")
		}
		totalAffected += affected
	}
	return UpsertResult{RowsAffected: totalAffected, TotalCount: len(companies), ListedCount: listedCount}, nil
}

func (r Repository) Search(ctx context.Context, query string, listedOnly bool, limit int) ([]opendartprovider.Company, error) {
	trimmed := strings.TrimSpace(query)
	errb := oops.In("opendart_company_repository").With("query", trimmed, "listed_only", listedOnly, "limit", limit)
	if trimmed == "" {
		return nil, errb.New("opendart company search requires query")
	}
	if limit <= 0 {
		limit = 20
	}
	client, err := r.database.Reader(ctx)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	var rows []storage.OpenDARTCompanyRow
	pattern := "%" + trimmed + "%"
	selectQuery := client.NewSelect().
		Model(&rows).
		Where("(corp_code = ? OR stock_code = ? OR corp_name LIKE ? OR corp_eng_name LIKE ?)", trimmed, trimmed, pattern, pattern).
		Order("listed DESC", "corp_name ASC").
		Limit(limit)
	if listedOnly {
		selectQuery = selectQuery.Where("listed = ?", true)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		return nil, errb.Wrapf(err, "search opendart companies")
	}
	companies := make([]opendartprovider.Company, 0, len(rows))
	for _, row := range rows {
		companies = append(companies, opendartprovider.Company{
			CorpCode:    row.CorpCode,
			CorpName:    row.CorpName,
			CorpEngName: row.CorpEngName,
			StockCode:   row.StockCode,
			ModifyDate:  row.ModifyDate,
		})
	}
	return companies, nil
}
