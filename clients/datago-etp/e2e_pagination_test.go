package etp_test

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	etp "github.com/awuzag/mwosa/clients/datago-etp"
	"github.com/samber/oops"
)

const (
	paginationTimingEnabledEnv = "DATAGO_ETP_PAGINATION_TIMING"
	paginationRowsEnv          = "DATAGO_ETP_PAGINATION_ROWS"
	paginationRepeatsEnv       = "DATAGO_ETP_PAGINATION_REPEATS"
	paginationOpsEnv           = "DATAGO_ETP_PAGINATION_OPS"
	paginationMaxCallsEnv      = "DATAGO_ETP_PAGINATION_MAX_CALLS"
	paginationOutputEnv        = "DATAGO_ETP_PAGINATION_OUTPUT"

	defaultPaginationRows     = "10,50,100,200"
	defaultPaginationRepeats  = 1
	defaultPaginationOps      = "etf,etn,elw"
	defaultPaginationMaxCalls = 30
	defaultPaginationOutput   = "e2e-output-pagination.json"
	datagoDailyQuota          = 10000
)

type paginationTimingReport struct {
	BasDt        string                     `json:"bas_dt"`
	CallsUsed    int                        `json:"calls_used"`
	DailyQuota   int                        `json:"daily_quota"`
	GeneratedAt  time.Time                  `json:"generated_at"`
	Measurements []paginationMeasurement    `json:"measurements"`
	BestByOp     []paginationRecommendation `json:"best_by_operation"`
}

type paginationMeasurement struct {
	Operation              string  `json:"operation"`
	NumOfRows              int     `json:"num_of_rows"`
	Repeat                 int     `json:"repeat"`
	DurationMillis         float64 `json:"duration_ms"`
	ItemCount              int     `json:"item_count"`
	TotalCount             int     `json:"total_count"`
	PageNo                 int     `json:"page_no"`
	EstimatedPageCount     int     `json:"estimated_page_count"`
	EstimatedTotalMillis   float64 `json:"estimated_total_ms"`
	EstimatedMillisPerItem float64 `json:"estimated_ms_per_item"`
}

type paginationRecommendation struct {
	Operation            string  `json:"operation"`
	NumOfRows            int     `json:"num_of_rows"`
	EstimatedPageCount   int     `json:"estimated_page_count"`
	EstimatedTotalMillis float64 `json:"estimated_total_ms"`
}

func TestLivePaginationRowTiming(t *testing.T) {
	skipUnlessLiveE2EEnabled(t)
	skipUnlessPaginationTimingEnabled(t)

	// This timing test intentionally samples only page 1 for each row size. It
	// estimates full-collection cost from totalCount so it does not burn the daily
	// Datago quota by walking every page.
	config := loadLiveConfig(t)
	rows := parsePositiveIntListEnv(t, paginationRowsEnv, defaultPaginationRows)
	repeats := parsePositiveIntEnv(t, paginationRepeatsEnv, defaultPaginationRepeats)
	ops := parseOperationListEnv(t, paginationOpsEnv, defaultPaginationOps)
	plannedCalls := len(rows) * repeats * len(ops)
	maxCalls := parsePositiveIntEnv(t, paginationMaxCallsEnv, defaultPaginationMaxCalls)
	if plannedCalls > datagoDailyQuota {
		t.Fatalf("pagination timing would use %d calls, exceeding Datago daily quota %d", plannedCalls, datagoDailyQuota)
	}
	if plannedCalls > maxCalls {
		t.Fatalf("pagination timing would use %d calls, max %d. Reduce %s/%s/%s or raise %s intentionally", plannedCalls, maxCalls, paginationRowsEnv, paginationRepeatsEnv, paginationOpsEnv, paginationMaxCallsEnv)
	}

	client, err := etp.New(etp.Config{ServiceKey: config.ServiceKey})
	if err != nil {
		t.Fatalf("new live client: %v", err)
	}
	timeout := parseLiveTimeout(t, config.Timeout)

	measurements := make([]paginationMeasurement, 0, plannedCalls)
	for _, op := range ops {
		for _, numOfRows := range rows {
			for repeat := 1; repeat <= repeats; repeat++ {
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				measurement, err := measurePagination(ctx, client, op, liveFixtureBasDt, numOfRows, repeat)
				cancel()
				if err != nil {
					t.Fatalf("measure pagination operation=%s numOfRows=%d repeat=%d: %v", op, numOfRows, repeat, err)
				}
				if measurement.TotalCount <= 0 || measurement.ItemCount == 0 {
					t.Fatalf("measure pagination operation=%s numOfRows=%d returned totalCount=%d itemCount=%d; use a known trading bas_dt", op, numOfRows, measurement.TotalCount, measurement.ItemCount)
				}
				measurements = append(measurements, measurement)
				t.Logf("pagination timing operation=%s numOfRows=%d repeat=%d duration=%0.2fms items=%d total=%d estimated_pages=%d estimated_total=%0.2fms",
					measurement.Operation,
					measurement.NumOfRows,
					measurement.Repeat,
					measurement.DurationMillis,
					measurement.ItemCount,
					measurement.TotalCount,
					measurement.EstimatedPageCount,
					measurement.EstimatedTotalMillis,
				)
			}
		}
	}

	report := paginationTimingReport{
		BasDt:        liveFixtureBasDt,
		CallsUsed:    plannedCalls,
		DailyQuota:   datagoDailyQuota,
		GeneratedAt:  time.Now(),
		Measurements: measurements,
		BestByOp:     bestPaginationRows(measurements),
	}
	for _, best := range report.BestByOp {
		t.Logf("pagination best operation=%s numOfRows=%d estimated_pages=%d estimated_total=%0.2fms",
			best.Operation,
			best.NumOfRows,
			best.EstimatedPageCount,
			best.EstimatedTotalMillis,
		)
	}
	writePaginationTimingReport(t, report)
}

func skipUnlessPaginationTimingEnabled(t *testing.T) {
	t.Helper()

	if os.Getenv(paginationTimingEnabledEnv) != "1" {
		t.Skipf("set %s=1 to run live pagination timing tests", paginationTimingEnabledEnv)
	}
}

func measurePagination(ctx context.Context, client *etp.Client, operation string, basDt string, numOfRows int, repeat int) (paginationMeasurement, error) {
	query := etp.SecuritiesProductPriceQuery{
		BasDt:     basDt,
		NumOfRows: numOfRows,
		PageNo:    1,
	}

	startedAt := time.Now()
	var itemCount int
	var totalCount int
	var pageNo int
	switch operation {
	case "etf":
		result, err := client.GetETFPriceInfo(ctx, etp.ETFPriceInfoQuery{SecuritiesProductPriceQuery: query})
		if err != nil {
			return paginationMeasurement{}, err
		}
		itemCount = len(result.Items)
		totalCount = result.TotalCount
		pageNo = result.PageNo
	case "etn":
		result, err := client.GetETNPriceInfo(ctx, etp.ETNPriceInfoQuery{SecuritiesProductPriceQuery: query})
		if err != nil {
			return paginationMeasurement{}, err
		}
		itemCount = len(result.Items)
		totalCount = result.TotalCount
		pageNo = result.PageNo
	case "elw":
		result, err := client.GetELWPriceInfo(ctx, etp.ELWPriceInfoQuery{SecuritiesProductPriceQuery: query})
		if err != nil {
			return paginationMeasurement{}, err
		}
		itemCount = len(result.Items)
		totalCount = result.TotalCount
		pageNo = result.PageNo
	default:
		return paginationMeasurement{}, oops.In("datago_etp_e2e").With("operation", operation).Errorf("unsupported pagination timing operation: %s", operation)
	}

	durationMillis := float64(time.Since(startedAt).Microseconds()) / 1000
	estimatedPageCount := int(math.Ceil(float64(totalCount) / float64(numOfRows)))
	estimatedTotalMillis := durationMillis * float64(estimatedPageCount)
	estimatedMillisPerItem := 0.0
	if itemCount > 0 {
		estimatedMillisPerItem = durationMillis / float64(itemCount)
	}
	return paginationMeasurement{
		Operation:              operation,
		NumOfRows:              numOfRows,
		Repeat:                 repeat,
		DurationMillis:         durationMillis,
		ItemCount:              itemCount,
		TotalCount:             totalCount,
		PageNo:                 pageNo,
		EstimatedPageCount:     estimatedPageCount,
		EstimatedTotalMillis:   estimatedTotalMillis,
		EstimatedMillisPerItem: estimatedMillisPerItem,
	}, nil
}

func parseOperationListEnv(t *testing.T, envName string, defaultValue string) []string {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		value = defaultValue
	}
	seen := map[string]bool{}
	ops := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		op := strings.ToLower(strings.TrimSpace(part))
		if op == "" {
			continue
		}
		switch op {
		case "etf", "etn", "elw":
			if !seen[op] {
				seen[op] = true
				ops = append(ops, op)
			}
		default:
			t.Fatalf("%s contains unsupported operation %q; use etf, etn, elw", envName, op)
		}
	}
	if len(ops) == 0 {
		t.Fatalf("%s must contain at least one operation", envName)
	}
	return ops
}

func parsePositiveIntListEnv(t *testing.T, envName string, defaultValue string) []int {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		value = defaultValue
	}
	seen := map[int]bool{}
	numbers := make([]int, 0)
	for _, part := range strings.Split(value, ",") {
		raw := strings.TrimSpace(part)
		if raw == "" {
			continue
		}
		number, err := strconv.Atoi(raw)
		if err != nil {
			t.Fatalf("%s contains non-integer value %q: %v", envName, raw, err)
		}
		if number <= 0 {
			t.Fatalf("%s must contain positive values: %d", envName, number)
		}
		if !seen[number] {
			seen[number] = true
			numbers = append(numbers, number)
		}
	}
	if len(numbers) == 0 {
		t.Fatalf("%s must contain at least one row size", envName)
	}
	sort.Ints(numbers)
	return numbers
}

func parsePositiveIntEnv(t *testing.T, envName string, defaultValue int) int {
	t.Helper()

	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return defaultValue
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("%s must be an integer: %v", envName, err)
	}
	if number <= 0 {
		t.Fatalf("%s must be positive: %d", envName, number)
	}
	return number
}

func bestPaginationRows(measurements []paginationMeasurement) []paginationRecommendation {
	bestByOp := map[string]paginationRecommendation{}
	for _, measurement := range measurements {
		current, exists := bestByOp[measurement.Operation]
		if !exists || measurement.EstimatedTotalMillis < current.EstimatedTotalMillis {
			bestByOp[measurement.Operation] = paginationRecommendation{
				Operation:            measurement.Operation,
				NumOfRows:            measurement.NumOfRows,
				EstimatedPageCount:   measurement.EstimatedPageCount,
				EstimatedTotalMillis: measurement.EstimatedTotalMillis,
			}
		}
	}

	ops := make([]string, 0, len(bestByOp))
	for op := range bestByOp {
		ops = append(ops, op)
	}
	sort.Strings(ops)

	best := make([]paginationRecommendation, 0, len(ops))
	for _, op := range ops {
		best = append(best, bestByOp[op])
	}
	return best
}

func writePaginationTimingReport(t *testing.T, report paginationTimingReport) {
	t.Helper()

	path := strings.TrimSpace(os.Getenv(paginationOutputEnv))
	if path == "" {
		path = defaultPaginationOutput
	}
	content, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal pagination timing report: %v", err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o600); err != nil {
		t.Fatalf("write pagination timing report %q: %v", path, err)
	}
	t.Logf("pagination timing report written to %s", path)
}
