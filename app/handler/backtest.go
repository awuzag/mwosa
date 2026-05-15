package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	backtestservice "github.com/ev3rlit/mwosa/service/backtest"
	"github.com/samber/oops"
)

type Backtest struct {
	service backtestservice.Service
}

func NewBacktest(service backtestservice.Service) Backtest {
	return Backtest{service: service}
}

type ValidateBacktestRequest struct {
	Path string
	View string
}

type RunBacktestRequest struct {
	Path string
	View string
}

type ValidateEvaluationRequest struct {
	Path string
}

type RunEvaluationRequest struct {
	Path        string
	Parallelism int
}

type ListBacktestStrategiesRequest struct{}

type ListBacktestRunsRequest struct{}

type ListEvaluationsRequest struct{}

type InspectBacktestStrategyRequest struct {
	Name string
}

type InspectBacktestUniverseRequest struct {
	Path string
	View string
}

type InspectEvaluationRequest struct {
	Ref  string
	View string
}

type InspectBacktestRunRequest struct {
	Ref  string
	View string
}

type CompareBacktestRunsRequest struct {
	LeftRef  string
	RightRef string
}

type CompareEvaluationRequest struct {
	Ref  string
	View string
}

type RankEvaluationRequest struct {
	Ref       string
	Objective string
}

type UpdateBacktestStrategyRequest struct {
	Name     string
	YAMLPath string
}

type DeleteBacktestStrategyRequest struct {
	Name string
}

func (h Backtest) Validate(ctx context.Context, req ValidateBacktestRequest) (BacktestValidationOutput, error) {
	view, err := normalizeBacktestView(req.View)
	if err != nil {
		return BacktestValidationOutput{}, err
	}
	result, err := h.service.Validate(ctx, req.Path)
	if err != nil {
		return BacktestValidationOutput{}, err
	}
	return BacktestValidationOutput{Result: result, View: view}, nil
}

func (h Backtest) Run(ctx context.Context, req RunBacktestRequest) (BacktestRunOutput, error) {
	view, err := normalizeBacktestRunView(req.View)
	if err != nil {
		return BacktestRunOutput{}, err
	}
	detail, err := h.service.RunAndSave(ctx, req.Path)
	if err != nil {
		return BacktestRunOutput{}, err
	}
	return BacktestRunOutput{Result: detail.Result, SavedRun: &detail.Run, View: view}, nil
}

func (h Backtest) ValidateEvaluation(ctx context.Context, req ValidateEvaluationRequest) (EvaluationValidationOutput, error) {
	result, err := h.service.ValidateEvaluation(ctx, req.Path)
	if err != nil {
		return EvaluationValidationOutput{}, err
	}
	return EvaluationValidationOutput{Result: result}, nil
}

func (h Backtest) RunEvaluation(ctx context.Context, req RunEvaluationRequest) (EvaluationRunOutput, error) {
	result, err := h.service.RunEvaluation(ctx, req.Path, backtestservice.RunEvaluationOptions{Parallelism: req.Parallelism})
	if err != nil {
		return EvaluationRunOutput{}, err
	}
	return EvaluationRunOutput{Result: result}, nil
}

func (h Backtest) ListStrategies(ctx context.Context, _ ListBacktestStrategiesRequest) (BacktestStrategyListOutput, error) {
	details, err := h.service.ListStrategies(ctx)
	if err != nil {
		return nil, err
	}
	out := make(BacktestStrategyListOutput, 0, len(details))
	for _, detail := range details {
		out = append(out, BacktestStrategyOutput{Detail: detail})
	}
	return out, nil
}

func (h Backtest) ListRuns(ctx context.Context, _ ListBacktestRunsRequest) (BacktestRunListOutput, error) {
	runs, err := h.service.ListRuns(ctx)
	if err != nil {
		return nil, err
	}
	return BacktestRunListOutput(runs), nil
}

func (h Backtest) ListEvaluations(ctx context.Context, _ ListEvaluationsRequest) (EvaluationListOutput, error) {
	result, err := h.service.ListEvaluations(ctx)
	if err != nil {
		return nil, err
	}
	return EvaluationListOutput(result), nil
}

func (h Backtest) InspectStrategy(ctx context.Context, req InspectBacktestStrategyRequest) (BacktestStrategyOutput, error) {
	detail, err := h.service.InspectStrategy(ctx, req.Name)
	if err != nil {
		return BacktestStrategyOutput{}, err
	}
	return BacktestStrategyOutput{Detail: detail, IncludeSpec: true}, nil
}

func (h Backtest) InspectUniverse(ctx context.Context, req InspectBacktestUniverseRequest) (BacktestUniverseOutput, error) {
	view, err := normalizeBacktestView(req.View)
	if err != nil {
		return BacktestUniverseOutput{}, err
	}
	explain, err := h.service.InspectUniverse(ctx, req.Path)
	if err != nil {
		return BacktestUniverseOutput{}, err
	}
	return BacktestUniverseOutput{Explain: explain, View: view}, nil
}

func (h Backtest) InspectRun(ctx context.Context, req InspectBacktestRunRequest) (BacktestRunOutput, error) {
	view, err := normalizeBacktestRunView(req.View)
	if err != nil {
		return BacktestRunOutput{}, err
	}
	detail, err := h.service.InspectRun(ctx, req.Ref)
	if err != nil {
		return BacktestRunOutput{}, err
	}
	return BacktestRunOutput{Result: detail.Result, SavedRun: &detail.Run, View: view}, nil
}

func (h Backtest) CompareRuns(ctx context.Context, req CompareBacktestRunsRequest) (BacktestRunComparisonOutput, error) {
	result, err := h.service.CompareRuns(ctx, req.LeftRef, req.RightRef)
	if err != nil {
		return BacktestRunComparisonOutput{}, err
	}
	return BacktestRunComparisonOutput{Result: result}, nil
}

func (h Backtest) InspectEvaluation(ctx context.Context, req InspectEvaluationRequest) (EvaluationDetailOutput, error) {
	view, err := normalizeEvaluationView(req.View)
	if err != nil {
		return EvaluationDetailOutput{}, err
	}
	detail, err := h.service.InspectEvaluation(ctx, req.Ref)
	if err != nil {
		return EvaluationDetailOutput{}, err
	}
	return EvaluationDetailOutput{Detail: detail, View: view}, nil
}

func (h Backtest) CompareEvaluation(ctx context.Context, req CompareEvaluationRequest) (EvaluationDetailOutput, error) {
	view, err := normalizeEvaluationView(req.View)
	if err != nil {
		return EvaluationDetailOutput{}, err
	}
	detail, err := h.service.CompareEvaluation(ctx, req.Ref)
	if err != nil {
		return EvaluationDetailOutput{}, err
	}
	return EvaluationDetailOutput{Detail: detail, View: view}, nil
}

func (h Backtest) RankEvaluation(ctx context.Context, req RankEvaluationRequest) (EvaluationRankOutput, error) {
	cases, err := h.service.RankEvaluation(ctx, req.Ref, req.Objective)
	if err != nil {
		return nil, err
	}
	return EvaluationRankOutput(cases), nil
}

func (h Backtest) UpdateStrategy(ctx context.Context, req UpdateBacktestStrategyRequest) (BacktestStrategyOutput, error) {
	detail, err := h.service.UpsertStrategy(ctx, backtestservice.SaveStrategyRequest{Name: req.Name, YAMLPath: req.YAMLPath})
	if err != nil {
		return BacktestStrategyOutput{}, err
	}
	return BacktestStrategyOutput{Detail: detail, IncludeSpec: true}, nil
}

func (h Backtest) DeleteStrategy(ctx context.Context, req DeleteBacktestStrategyRequest) (DeleteBacktestStrategyResult, error) {
	if err := h.service.DeleteStrategy(ctx, req.Name); err != nil {
		return DeleteBacktestStrategyResult{}, err
	}
	return DeleteBacktestStrategyResult{Name: req.Name, Deleted: true}, nil
}

type BacktestValidationOutput struct {
	Result backtestservice.ValidationResult
	View   string
}

type EvaluationValidationOutput struct {
	Result backtestservice.EvaluationValidationResult
}

func (o EvaluationValidationOutput) JSONValue() any {
	return o.Result
}

func (o EvaluationValidationOutput) NDJSONRows() any {
	return o.Result
}

func (o EvaluationValidationOutput) CSVRows() any {
	return []backtestservice.EvaluationValidationResult{o.Result}
}

func (o EvaluationValidationOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"valid", "evaluation", "strategy", "base_run", "cases", "walk_forward", "parallelism", "metrics"}, [][]string{{
		fmt.Sprint(result.Valid),
		result.Name,
		result.StrategyName,
		result.BaseRunName,
		fmt.Sprint(result.CaseCount),
		fmt.Sprint(result.WalkForwardSteps),
		fmt.Sprint(result.Parallelism),
		fmt.Sprint(len(result.Metrics)),
	}}
}

func (o BacktestValidationOutput) JSONValue() any {
	if o.view() == backtestViewRaw {
		return o.Result
	}
	return BacktestValidationView{
		Valid:        o.Result.Valid,
		StrategyName: o.Result.StrategyName,
		RunName:      o.Result.RunName,
		Symbols:      o.Result.Symbols,
		Instruments:  o.Result.Instruments,
		Period:       o.Result.Period,
		Market:       o.Result.Market,
		Timeframe:    o.Result.Timeframe,
		Timeframes:   o.Result.Timeframes,
		Runtime:      o.Result.Runtime,
		Currency:     o.Result.Currency,
		Execution:    o.Result.Execution,
		Metrics:      o.Result.Metrics,
		Indicators:   o.Result.Indicators,
		Universe:     universeSummary(o.Result.Universe),
	}
}

func (o BacktestValidationOutput) NDJSONRows() any {
	return o.JSONValue()
}

func (o BacktestValidationOutput) CSVRows() any {
	result := o.Result
	return []backtestValidationRow{{
		Valid:        result.Valid,
		StrategyName: result.StrategyName,
		RunName:      result.RunName,
		Symbols:      len(result.Symbols),
		From:         result.Period.From.Format("2006-01-02"),
		To:           result.Period.To.Format("2006-01-02"),
		Fill:         result.Execution.Fill,
	}}
}

func (o BacktestValidationOutput) view() string {
	view, err := normalizeBacktestView(o.View)
	if err != nil {
		return backtestViewSummary
	}
	return view
}

type BacktestValidationView struct {
	Valid        bool                             `json:"valid"`
	StrategyName string                           `json:"strategy_name"`
	RunName      string                           `json:"run_name"`
	Symbols      []string                         `json:"symbols"`
	Instruments  []core.InstrumentIdentity        `json:"instruments,omitempty"`
	Period       core.Period                      `json:"period"`
	Market       string                           `json:"market"`
	Timeframe    string                           `json:"timeframe"`
	Timeframes   core.TimeframeMetadata           `json:"timeframes"`
	Runtime      core.RuntimeMetadata             `json:"runtime"`
	Currency     string                           `json:"currency"`
	Execution    backtestservice.ExecutionSummary `json:"execution"`
	Metrics      []string                         `json:"metrics"`
	Indicators   map[string]string                `json:"indicators,omitempty"`
	Universe     BacktestUniverseSummary          `json:"universe"`
}

func (o BacktestValidationOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	return []string{"valid", "strategy", "run", "symbols", "schedule", "steps", "from", "to", "fill"}, [][]string{{
		fmt.Sprint(result.Valid),
		result.StrategyName,
		result.RunName,
		fmt.Sprint(len(result.Symbols)),
		result.Universe.Schedule,
		fmt.Sprint(len(result.Universe.Steps)),
		result.Period.From.Format("2006-01-02"),
		result.Period.To.Format("2006-01-02"),
		result.Execution.Fill,
	}}
}

type backtestValidationRow struct {
	Valid        bool   `json:"valid" csv:"valid"`
	StrategyName string `json:"strategy_name" csv:"strategy_name"`
	RunName      string `json:"run_name" csv:"run_name"`
	Symbols      int    `json:"symbols" csv:"symbols"`
	From         string `json:"from" csv:"from"`
	To           string `json:"to" csv:"to"`
	Fill         string `json:"fill" csv:"fill"`
}

type BacktestUniverseOutput struct {
	Explain core.UniverseExplain
	View    string
}

func (o BacktestUniverseOutput) JSONValue() any {
	switch o.view() {
	case backtestViewRaw:
		return o.Explain
	default:
		return universeSummary(o.Explain)
	}
}

func (o BacktestUniverseOutput) NDJSONRows() any {
	switch o.view() {
	case backtestViewRaw:
		return o.Explain.Snapshots
	default:
		return []BacktestUniverseSummary{universeSummary(o.Explain)}
	}
}

func (o BacktestUniverseOutput) CSVRows() any {
	return []backtestUniverseRow{universeRow(o.Explain)}
}

func (o BacktestUniverseOutput) view() string {
	view, err := normalizeBacktestView(o.View)
	if err != nil {
		return backtestViewSummary
	}
	return view
}

func (o BacktestUniverseOutput) TableRows() ([]string, [][]string) {
	row := universeRow(o.Explain)
	return []string{"mode", "schedule", "symbols", "snapshots", "steps", "policy"}, [][]string{{
		row.Mode,
		row.Schedule,
		fmt.Sprint(row.Symbols),
		fmt.Sprint(row.Snapshots),
		fmt.Sprint(row.Steps),
		row.PositionPolicy,
	}}
}

type backtestUniverseRow struct {
	Mode           string `json:"mode" csv:"mode"`
	Schedule       string `json:"schedule" csv:"schedule"`
	Symbols        int    `json:"symbols" csv:"symbols"`
	Snapshots      int    `json:"snapshots" csv:"snapshots"`
	Steps          int    `json:"steps" csv:"steps"`
	PositionPolicy string `json:"position_policy" csv:"position_policy"`
}

func universeRow(explain core.UniverseExplain) backtestUniverseRow {
	return backtestUniverseRow{
		Mode:           explain.Mode,
		Schedule:       explain.Schedule,
		Symbols:        len(explain.SelectedSymbols),
		Snapshots:      len(explain.Snapshots),
		Steps:          len(explain.Steps),
		PositionPolicy: explain.PositionPolicy,
	}
}

const (
	backtestViewSummary = "summary"
	backtestViewRaw     = "raw"
)

type BacktestUniverseSummary struct {
	Mode            string                            `json:"mode"`
	Schedule        string                            `json:"schedule"`
	PositionPolicy  string                            `json:"position_policy"`
	SelectedSymbols []string                          `json:"selected_symbols"`
	SymbolCount     int                               `json:"symbol_count"`
	StepCount       int                               `json:"step_count"`
	DecisionCount   int                               `json:"decision_count"`
	SnapshotCount   int                               `json:"snapshot_count"`
	Steps           []core.UniverseStepSummary        `json:"steps,omitempty"`
	Snapshots       []BacktestUniverseSnapshotSummary `json:"snapshots,omitempty"`
	RawView         string                            `json:"raw_view"`
}

type BacktestUniverseSnapshotSummary struct {
	Time           time.Time `json:"time"`
	Symbols        []string  `json:"symbols"`
	SymbolCount    int       `json:"symbol_count"`
	CandidateCount int       `json:"candidate_count"`
	DecisionCount  int       `json:"decision_count"`
	StepCount      int       `json:"step_count"`
}

func normalizeBacktestView(value string) (string, error) {
	view := strings.ToLower(strings.TrimSpace(value))
	if view == "" {
		return backtestViewSummary, nil
	}
	switch view {
	case backtestViewSummary, backtestViewRaw:
		return view, nil
	default:
		return "", oops.In("backtest_handler").With("view", value).Errorf("unsupported backtest view: %s", value)
	}
}

func universeSummary(explain core.UniverseExplain) BacktestUniverseSummary {
	snapshots := make([]BacktestUniverseSnapshotSummary, 0, len(explain.Snapshots))
	for _, snapshot := range explain.Snapshots {
		snapshots = append(snapshots, BacktestUniverseSnapshotSummary{
			Time:           snapshot.Time,
			Symbols:        append([]string(nil), snapshot.Symbols...),
			SymbolCount:    len(snapshot.Symbols),
			CandidateCount: len(snapshot.Candidates),
			DecisionCount:  len(snapshot.Decisions),
			StepCount:      len(snapshot.Steps),
		})
	}
	return BacktestUniverseSummary{
		Mode:            explain.Mode,
		Schedule:        explain.Schedule,
		PositionPolicy:  explain.PositionPolicy,
		SelectedSymbols: append([]string(nil), explain.SelectedSymbols...),
		SymbolCount:     len(explain.SelectedSymbols),
		StepCount:       len(explain.Steps),
		DecisionCount:   len(explain.Decisions),
		SnapshotCount:   len(explain.Snapshots),
		Steps:           append([]core.UniverseStepSummary(nil), explain.Steps...),
		Snapshots:       snapshots,
		RawView:         "raw",
	}
}

type BacktestRunOutput struct {
	Result   core.Result
	SavedRun *backtestservice.SavedBacktestRun
	View     string
}

func (o BacktestRunOutput) JSONValue() any {
	return o.viewValue()
}

func (o BacktestRunOutput) NDJSONRows() any {
	return o.rowsValue()
}

func (o BacktestRunOutput) CSVRows() any {
	return o.rowsValue()
}

func (o BacktestRunOutput) TableRows() ([]string, [][]string) {
	switch o.view() {
	case backtestRunViewTrades:
		rows := make([][]string, 0, len(o.Result.Trades))
		for _, trade := range o.Result.Trades {
			rows = append(rows, []string{
				trade.Time.Format(time.DateOnly),
				trade.Symbol,
				string(trade.Side),
				fmt.Sprintf("%.4f", trade.Quantity),
				fmt.Sprintf("%.4f", trade.Price),
				fmt.Sprintf("%.4f", trade.Notional),
				fmt.Sprintf("%.4f", trade.Commission),
				fmt.Sprintf("%.4f", trade.Tax),
				fmt.Sprintf("%.4f", trade.ExchangeFee),
				fmt.Sprintf("%.4f", trade.TotalCost),
				trade.Reason,
			})
		}
		return []string{"time", "symbol", "side", "quantity", "price", "notional", "commission", "tax", "exchange_fee", "total_cost", "reason"}, rows
	case backtestRunViewOrders:
		rows := make([][]string, 0, len(o.orderRows()))
		for _, order := range o.orderRows() {
			rows = append(rows, []string{order.Time, order.Symbol, order.Side, fmt.Sprintf("%.4f", order.Amount), fmt.Sprintf("%.4f", order.Quantity), order.Reason})
		}
		return []string{"time", "symbol", "side", "amount", "quantity", "reason"}, rows
	case backtestRunViewFills:
		rows := make([][]string, 0, len(o.fillRows()))
		for _, fill := range o.fillRows() {
			rows = append(rows, []string{
				fill.Time,
				fill.Type,
				fill.Symbol,
				fill.Side,
				fmt.Sprintf("%.4f", fill.Quantity),
				fmt.Sprintf("%.4f", fill.Price),
				fmt.Sprintf("%.4f", fill.Notional),
				fmt.Sprintf("%.4f", fill.Commission),
				fmt.Sprintf("%.4f", fill.Tax),
				fmt.Sprintf("%.4f", fill.ExchangeFee),
				fmt.Sprintf("%.4f", fill.TotalCost),
				fmt.Sprintf("%.4f", fill.SlippageBps),
				fill.Reason,
			})
		}
		return []string{"time", "type", "symbol", "side", "quantity", "price", "notional", "commission", "tax", "exchange_fee", "total_cost", "slippage_bps", "reason"}, rows
	case backtestRunViewPositions:
		rows := make([][]string, 0, len(o.Result.Positions))
		for _, position := range o.Result.Positions {
			entryTime := ""
			if !position.EntryTime.IsZero() {
				entryTime = position.EntryTime.Format(time.DateOnly)
			}
			rows = append(rows, []string{
				position.Symbol,
				fmt.Sprintf("%.4f", position.Quantity),
				fmt.Sprintf("%.4f", position.AvgPrice),
				fmt.Sprintf("%.4f", position.MarketPrice),
				fmt.Sprintf("%.4f", position.MarketValue),
				fmt.Sprintf("%.4f", position.WeightPct),
				entryTime,
			})
		}
		return []string{"symbol", "quantity", "avg_price", "market_price", "market_value", "weight_pct", "entry_time"}, rows
	case backtestRunViewEquity:
		rows := make([][]string, 0, len(o.Result.EquityCurve))
		for _, point := range o.Result.EquityCurve {
			rows = append(rows, []string{
				point.Time.Format(time.DateOnly),
				fmt.Sprintf("%.4f", point.Cash),
				fmt.Sprintf("%.4f", point.PositionsValue),
				fmt.Sprintf("%.4f", point.Equity),
			})
		}
		return []string{"time", "cash", "positions_value", "equity"}, rows
	case backtestRunViewEvents:
		events := o.events()
		rows := make([][]string, 0, len(events))
		for _, event := range events {
			timeText := ""
			if !event.Time.IsZero() {
				timeText = event.Time.Format(time.DateOnly)
			}
			rows = append(rows, []string{
				timeText,
				event.Layer,
				event.Type,
				event.Symbol,
				string(event.Side),
				event.Reason,
				event.Timeframe,
				event.Session,
				event.Status,
				fmt.Sprintf("%.4f", event.Quantity),
				fmt.Sprintf("%.4f", event.Price),
				fmt.Sprintf("%.4f", event.Notional),
				fmt.Sprintf("%.4f", event.Commission),
				fmt.Sprintf("%.4f", event.Tax),
				fmt.Sprintf("%.4f", event.ExchangeFee),
				fmt.Sprintf("%.4f", event.TotalCost),
				fmt.Sprintf("%.4f", event.SlippageBps),
			})
		}
		return []string{"time", "layer", "type", "symbol", "side", "reason", "timeframe", "session", "status", "quantity", "price", "notional", "commission", "tax", "exchange_fee", "total_cost", "slippage_bps"}, rows
	case backtestRunViewMetrics:
		rows := make([][]string, 0, len(o.Result.SelectedMetrics))
		for _, metric := range o.Result.SelectedMetrics {
			rows = append(rows, []string{metric, fmt.Sprintf("%.6f", o.Result.Metrics[metric])})
		}
		return []string{"metric", "value"}, rows
	case backtestRunViewUniverse:
		row := universeRow(o.Result.Universe)
		return []string{"mode", "schedule", "symbols", "snapshots", "steps", "policy"}, [][]string{{
			row.Mode,
			row.Schedule,
			fmt.Sprint(row.Symbols),
			fmt.Sprint(row.Snapshots),
			fmt.Sprint(row.Steps),
			row.PositionPolicy,
		}}
	}
	result := o.Result
	header := []string{"strategy", "run", "symbols"}
	row := []string{
		result.StrategyName,
		result.RunName,
		fmt.Sprint(len(result.Symbols)),
	}
	if o.SavedRun != nil {
		header = append(header, "id")
		row = append(row, o.SavedRun.ID)
	}
	header = append(header, "universe")
	row = append(row, result.Universe.Schedule+"/"+fmt.Sprint(len(result.Universe.Steps))+"steps")
	for _, metric := range result.SelectedMetrics {
		header = append(header, metric)
		row = append(row, fmt.Sprintf("%.6f", result.Metrics[metric]))
	}
	header = append(header, "hash")
	row = append(row, result.ResultHash)
	return header, [][]string{row}
}

const (
	backtestRunViewRaw       = "raw"
	backtestRunViewSummary   = "summary"
	backtestRunViewMetrics   = "metrics"
	backtestRunViewOrders    = "orders"
	backtestRunViewFills     = "fills"
	backtestRunViewTrades    = "trades"
	backtestRunViewPositions = "positions"
	backtestRunViewEquity    = "equity"
	backtestRunViewUniverse  = "universe"
	backtestRunViewEvents    = "events"
)

type BacktestRunSummaryView struct {
	ID              string                    `json:"id,omitempty" csv:"id"`
	StrategyName    string                    `json:"strategy_name" csv:"strategy_name"`
	RunName         string                    `json:"run_name" csv:"run_name"`
	Symbols         int                       `json:"symbols" csv:"symbols"`
	PeriodFrom      string                    `json:"period_from" csv:"period_from"`
	PeriodTo        string                    `json:"period_to" csv:"period_to"`
	Market          string                    `json:"market" csv:"market"`
	Timeframe       string                    `json:"timeframe" csv:"timeframe"`
	Timeframes      core.TimeframeMetadata    `json:"timeframes" csv:"-"`
	Runtime         core.RuntimeMetadata      `json:"runtime" csv:"-"`
	Currency        string                    `json:"currency" csv:"currency"`
	FinalEquity     float64                   `json:"final_equity" csv:"final_equity"`
	Metrics         core.Metrics              `json:"metrics,omitempty" csv:"-"`
	StrategyHash    string                    `json:"strategy_hash,omitempty" csv:"strategy_hash"`
	RunHash         string                    `json:"run_hash,omitempty" csv:"run_hash"`
	DataFingerprint string                    `json:"data_fingerprint" csv:"data_fingerprint"`
	ResultHash      string                    `json:"result_hash" csv:"result_hash"`
	Universe        backtestUniverseRow       `json:"universe" csv:"-"`
	Execution       core.ExecutionAssumption  `json:"execution" csv:"-"`
	Instruments     []core.InstrumentIdentity `json:"instruments,omitempty" csv:"-"`
}

type BacktestMetricRow struct {
	Metric string  `json:"metric" csv:"metric"`
	Value  float64 `json:"value" csv:"value"`
}

type BacktestTradeRow struct {
	Time        string  `json:"time" csv:"time"`
	Symbol      string  `json:"symbol" csv:"symbol"`
	Side        string  `json:"side" csv:"side"`
	Quantity    float64 `json:"quantity" csv:"quantity"`
	Price       float64 `json:"price" csv:"price"`
	Notional    float64 `json:"notional" csv:"notional"`
	Commission  float64 `json:"commission" csv:"commission"`
	Tax         float64 `json:"tax" csv:"tax"`
	ExchangeFee float64 `json:"exchange_fee" csv:"exchange_fee"`
	TotalCost   float64 `json:"total_cost" csv:"total_cost"`
	SlippageBps float64 `json:"slippage_bps" csv:"slippage_bps"`
	Reason      string  `json:"reason" csv:"reason"`
	RealizedPnL float64 `json:"realized_pnl" csv:"realized_pnl"`
	Return      float64 `json:"return" csv:"return"`
}

type BacktestOrderRow struct {
	Time     string  `json:"time" csv:"time"`
	Symbol   string  `json:"symbol" csv:"symbol"`
	Side     string  `json:"side" csv:"side"`
	Amount   float64 `json:"amount" csv:"amount"`
	Quantity float64 `json:"quantity" csv:"quantity"`
	Reason   string  `json:"reason" csv:"reason"`
}

type BacktestFillRow struct {
	Time        string  `json:"time" csv:"time"`
	Type        string  `json:"type" csv:"type"`
	Symbol      string  `json:"symbol" csv:"symbol"`
	Side        string  `json:"side" csv:"side"`
	Quantity    float64 `json:"quantity" csv:"quantity"`
	Price       float64 `json:"price" csv:"price"`
	Notional    float64 `json:"notional" csv:"notional"`
	Commission  float64 `json:"commission" csv:"commission"`
	Tax         float64 `json:"tax" csv:"tax"`
	ExchangeFee float64 `json:"exchange_fee" csv:"exchange_fee"`
	TotalCost   float64 `json:"total_cost" csv:"total_cost"`
	SlippageBps float64 `json:"slippage_bps" csv:"slippage_bps"`
	Reason      string  `json:"reason" csv:"reason"`
}

type BacktestPositionRow struct {
	Symbol      string  `json:"symbol" csv:"symbol"`
	Quantity    float64 `json:"quantity" csv:"quantity"`
	AvgPrice    float64 `json:"avg_price" csv:"avg_price"`
	MarketPrice float64 `json:"market_price" csv:"market_price"`
	MarketValue float64 `json:"market_value" csv:"market_value"`
	WeightPct   float64 `json:"weight_pct" csv:"weight_pct"`
	EntryTime   string  `json:"entry_time,omitempty" csv:"entry_time"`
}

type BacktestEquityRow struct {
	Time           string  `json:"time" csv:"time"`
	Cash           float64 `json:"cash" csv:"cash"`
	PositionsValue float64 `json:"positions_value" csv:"positions_value"`
	Equity         float64 `json:"equity" csv:"equity"`
}

type BacktestEventRow struct {
	Time        string  `json:"time,omitempty" csv:"time"`
	Layer       string  `json:"layer" csv:"layer"`
	Type        string  `json:"type" csv:"type"`
	Symbol      string  `json:"symbol,omitempty" csv:"symbol"`
	Side        string  `json:"side,omitempty" csv:"side"`
	Reason      string  `json:"reason" csv:"reason"`
	Timeframe   string  `json:"timeframe,omitempty" csv:"timeframe"`
	Session     string  `json:"session,omitempty" csv:"session"`
	Status      string  `json:"status,omitempty" csv:"status"`
	Quantity    float64 `json:"quantity,omitempty" csv:"quantity"`
	Price       float64 `json:"price,omitempty" csv:"price"`
	Notional    float64 `json:"notional,omitempty" csv:"notional"`
	Commission  float64 `json:"commission,omitempty" csv:"commission"`
	Tax         float64 `json:"tax,omitempty" csv:"tax"`
	ExchangeFee float64 `json:"exchange_fee,omitempty" csv:"exchange_fee"`
	TotalCost   float64 `json:"total_cost,omitempty" csv:"total_cost"`
	SlippageBps float64 `json:"slippage_bps,omitempty" csv:"slippage_bps"`
}

func normalizeBacktestRunView(value string) (string, error) {
	view := strings.ToLower(strings.TrimSpace(value))
	if view == "" {
		return backtestRunViewRaw, nil
	}
	switch view {
	case backtestRunViewRaw, backtestRunViewSummary, backtestRunViewMetrics, backtestRunViewOrders, backtestRunViewFills, backtestRunViewTrades, backtestRunViewPositions, backtestRunViewEquity, backtestRunViewUniverse, backtestRunViewEvents:
		return view, nil
	default:
		return "", oops.In("backtest_handler").With("view", value).Errorf("unsupported backtest run view: %s", value)
	}
}

func (o BacktestRunOutput) view() string {
	view, err := normalizeBacktestRunView(o.View)
	if err != nil {
		return backtestRunViewRaw
	}
	return view
}

func (o BacktestRunOutput) viewValue() any {
	switch o.view() {
	case backtestRunViewSummary:
		return o.summary()
	case backtestRunViewMetrics:
		return o.metricRows()
	case backtestRunViewOrders:
		return o.orderRows()
	case backtestRunViewFills:
		return o.fillRows()
	case backtestRunViewTrades:
		return o.tradeRows()
	case backtestRunViewPositions:
		return o.positionRows()
	case backtestRunViewEquity:
		return o.equityRows()
	case backtestRunViewUniverse:
		return universeSummary(o.Result.Universe)
	case backtestRunViewEvents:
		return o.eventRows()
	default:
		return o.Result
	}
}

func (o BacktestRunOutput) rowsValue() any {
	switch o.view() {
	case backtestRunViewSummary:
		return []BacktestRunSummaryView{o.summary()}
	case backtestRunViewMetrics:
		return o.metricRows()
	case backtestRunViewOrders:
		return o.orderRows()
	case backtestRunViewFills:
		return o.fillRows()
	case backtestRunViewTrades:
		return o.tradeRows()
	case backtestRunViewPositions:
		return o.positionRows()
	case backtestRunViewEquity:
		return o.equityRows()
	case backtestRunViewUniverse:
		return []backtestUniverseRow{universeRow(o.Result.Universe)}
	case backtestRunViewEvents:
		return o.eventRows()
	default:
		return o.tradeRows()
	}
}

func (o BacktestRunOutput) summary() BacktestRunSummaryView {
	result := o.Result
	finalEquity := result.FinalEquity
	if finalEquity == 0 {
		if value, ok := result.Metrics[core.MetricFinalEquity]; ok {
			finalEquity = value
		}
	}
	summary := BacktestRunSummaryView{
		StrategyName:    result.StrategyName,
		RunName:         result.RunName,
		Symbols:         len(result.Symbols),
		PeriodFrom:      result.Period.From.Format(time.DateOnly),
		PeriodTo:        result.Period.To.Format(time.DateOnly),
		Market:          result.Market,
		Timeframe:       result.Timeframe,
		Timeframes:      result.Timeframes,
		Runtime:         result.Runtime,
		Currency:        result.Currency,
		FinalEquity:     finalEquity,
		Metrics:         result.Metrics,
		DataFingerprint: result.DataFingerprint,
		ResultHash:      result.ResultHash,
		Universe:        universeRow(result.Universe),
		Execution:       result.Execution,
		Instruments:     result.Instruments,
	}
	if o.SavedRun != nil {
		summary.ID = o.SavedRun.ID
		summary.StrategyHash = o.SavedRun.StrategyHash
		summary.RunHash = o.SavedRun.RunHash
		if summary.DataFingerprint == "" {
			summary.DataFingerprint = o.SavedRun.DataFingerprint
		}
	}
	return summary
}

func (o BacktestRunOutput) metricRows() []BacktestMetricRow {
	rows := make([]BacktestMetricRow, 0, len(o.Result.SelectedMetrics))
	for _, metric := range o.Result.SelectedMetrics {
		rows = append(rows, BacktestMetricRow{Metric: metric, Value: o.Result.Metrics[metric]})
	}
	return rows
}

func (o BacktestRunOutput) tradeRows() []BacktestTradeRow {
	rows := make([]BacktestTradeRow, 0, len(o.Result.Trades))
	for _, trade := range o.Result.Trades {
		rows = append(rows, BacktestTradeRow{
			Time:        trade.Time.Format(time.DateOnly),
			Symbol:      trade.Symbol,
			Side:        string(trade.Side),
			Quantity:    trade.Quantity,
			Price:       trade.Price,
			Notional:    trade.Notional,
			Commission:  trade.Commission,
			Tax:         trade.Tax,
			ExchangeFee: trade.ExchangeFee,
			TotalCost:   trade.TotalCost,
			SlippageBps: trade.SlippageBps,
			Reason:      trade.Reason,
			RealizedPnL: trade.RealizedPnL,
			Return:      trade.Return,
		})
	}
	return rows
}

func (o BacktestRunOutput) orderRows() []BacktestOrderRow {
	rows := make([]BacktestOrderRow, 0)
	for _, event := range o.Result.ExecutionEvents {
		if event.Type != "order_intent" {
			continue
		}
		rows = append(rows, BacktestOrderRow{
			Time:     event.Time.Format(time.DateOnly),
			Symbol:   event.Symbol,
			Side:     string(event.Side),
			Amount:   event.Amount,
			Quantity: event.Quantity,
			Reason:   event.Reason,
		})
	}
	return rows
}

func (o BacktestRunOutput) fillRows() []BacktestFillRow {
	rows := make([]BacktestFillRow, 0)
	for _, event := range o.Result.ExecutionEvents {
		if event.Layer != "execution" || event.Type == "order_intent" {
			continue
		}
		rows = append(rows, BacktestFillRow{
			Time:        eventTimeText(event),
			Type:        event.Type,
			Symbol:      event.Symbol,
			Side:        string(event.Side),
			Quantity:    event.Quantity,
			Price:       event.Price,
			Notional:    event.Notional,
			Commission:  event.Commission,
			Tax:         event.Tax,
			ExchangeFee: event.ExchangeFee,
			TotalCost:   event.TotalCost,
			SlippageBps: event.SlippageBps,
			Reason:      event.Reason,
		})
	}
	return rows
}

func (o BacktestRunOutput) positionRows() []BacktestPositionRow {
	rows := make([]BacktestPositionRow, 0, len(o.Result.Positions))
	for _, position := range o.Result.Positions {
		entryTime := ""
		if !position.EntryTime.IsZero() {
			entryTime = position.EntryTime.Format(time.DateOnly)
		}
		rows = append(rows, BacktestPositionRow{
			Symbol:      position.Symbol,
			Quantity:    position.Quantity,
			AvgPrice:    position.AvgPrice,
			MarketPrice: position.MarketPrice,
			MarketValue: position.MarketValue,
			WeightPct:   position.WeightPct,
			EntryTime:   entryTime,
		})
	}
	return rows
}

func (o BacktestRunOutput) equityRows() []BacktestEquityRow {
	rows := make([]BacktestEquityRow, 0, len(o.Result.EquityCurve))
	for _, point := range o.Result.EquityCurve {
		rows = append(rows, BacktestEquityRow{
			Time:           point.Time.Format(time.DateOnly),
			Cash:           point.Cash,
			PositionsValue: point.PositionsValue,
			Equity:         point.Equity,
		})
	}
	return rows
}

func (o BacktestRunOutput) events() []core.Event {
	events := make([]core.Event, 0, len(o.Result.DataEvents)+len(o.Result.ExecutionEvents)+len(o.Result.RiskEvents))
	events = append(events, o.Result.DataEvents...)
	events = append(events, o.Result.RiskEvents...)
	events = append(events, o.Result.ExecutionEvents...)
	slices.SortFunc(events, func(a, b core.Event) int {
		if !a.Time.Equal(b.Time) {
			if a.Time.Before(b.Time) {
				return -1
			}
			return 1
		}
		if a.Layer != b.Layer {
			return strings.Compare(a.Layer, b.Layer)
		}
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		return strings.Compare(a.Symbol, b.Symbol)
	})
	return events
}

func (o BacktestRunOutput) eventRows() []BacktestEventRow {
	events := o.events()
	rows := make([]BacktestEventRow, 0, len(events))
	for _, event := range events {
		rows = append(rows, BacktestEventRow{
			Time:        eventTimeText(event),
			Layer:       event.Layer,
			Type:        event.Type,
			Symbol:      event.Symbol,
			Side:        string(event.Side),
			Reason:      event.Reason,
			Timeframe:   event.Timeframe,
			Session:     event.Session,
			Status:      event.Status,
			Quantity:    event.Quantity,
			Price:       event.Price,
			Notional:    event.Notional,
			Commission:  event.Commission,
			Tax:         event.Tax,
			ExchangeFee: event.ExchangeFee,
			TotalCost:   event.TotalCost,
			SlippageBps: event.SlippageBps,
		})
	}
	return rows
}

func eventTimeText(event core.Event) string {
	if event.Time.IsZero() {
		return ""
	}
	return event.Time.Format(time.DateOnly)
}

type BacktestRunListOutput []backtestservice.SavedBacktestRun

func (o BacktestRunListOutput) JSONValue() any {
	return []backtestservice.SavedBacktestRun(o)
}

func (o BacktestRunListOutput) NDJSONRows() any {
	return []backtestservice.SavedBacktestRun(o)
}

func (o BacktestRunListOutput) CSVRows() any {
	return []backtestservice.SavedBacktestRun(o)
}

func (o BacktestRunListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, run := range o {
		rows = append(rows, []string{
			run.ID,
			run.RunName,
			run.StrategyName,
			run.Market,
			run.Timeframe,
			run.PeriodFrom,
			run.PeriodTo,
			run.ResultHash,
		})
	}
	return []string{"id", "run", "strategy", "market", "timeframe", "from", "to", "result_hash"}, rows
}

type BacktestRunComparisonOutput struct {
	Result backtestservice.BacktestRunComparison
}

func (o BacktestRunComparisonOutput) JSONValue() any {
	return o.Result
}

func (o BacktestRunComparisonOutput) NDJSONRows() any {
	return o.Result.Metrics
}

func (o BacktestRunComparisonOutput) CSVRows() any {
	return o.Result.Metrics
}

func (o BacktestRunComparisonOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Metrics)+5)
	rows = append(rows,
		[]string{"strategy_hash", boolText(o.Result.SameStrategyHash), "", "", ""},
		[]string{"run_hash", boolText(o.Result.SameRunHash), "", "", ""},
		[]string{"runtime", boolText(o.Result.SameRuntime), "", "", ""},
		[]string{"data_fingerprint", boolText(o.Result.SameDataFingerprint), "", "", ""},
		[]string{"result_hash", boolText(o.Result.SameResultHash), "", "", ""},
	)
	for _, metric := range o.Result.Metrics {
		rows = append(rows, []string{
			metric.Metric,
			fmt.Sprintf("%.6f", metric.LeftValue),
			fmt.Sprintf("%.6f", metric.RightValue),
			fmt.Sprintf("%.6f", metric.Delta),
			fmt.Sprintf("%.6f", metric.DeltaPct),
		})
	}
	return []string{"metric", "left", "right", "delta", "delta_pct"}, rows
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

type EvaluationRunOutput struct {
	Result backtestservice.EvaluationRunResult
}

func (o EvaluationRunOutput) JSONValue() any {
	return o.Result
}

func (o EvaluationRunOutput) NDJSONRows() any {
	return o.Result.Cases
}

func (o EvaluationRunOutput) CSVRows() any {
	return evaluationCaseRows(o.Result.Cases)
}

func (o EvaluationRunOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Result.Ranking))
	for _, item := range o.Result.Ranking {
		rows = append(rows, evaluationCaseTableRow(item))
	}
	return evaluationCaseTableHeader(), rows
}

type EvaluationListOutput []backtestservice.SavedEvaluationSummary

func (o EvaluationListOutput) JSONValue() any {
	return []backtestservice.SavedEvaluationSummary(o)
}

func (o EvaluationListOutput) NDJSONRows() any {
	return o.JSONValue()
}

func (o EvaluationListOutput) CSVRows() any {
	return o.JSONValue()
}

func (o EvaluationListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, item := range o {
		best := ""
		if item.BestCase != nil {
			best = item.BestCase.CaseID
		}
		rows = append(rows, []string{
			item.Experiment.ID,
			item.Experiment.Name,
			item.Experiment.StrategyName,
			fmt.Sprint(item.CaseCount),
			best,
			item.Experiment.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return []string{"id", "name", "strategy", "cases", "best_case", "created_at"}, rows
}

type EvaluationDetailOutput struct {
	Detail backtestservice.SavedEvaluationDetail
	View   string
}

func (o EvaluationDetailOutput) JSONValue() any {
	switch o.view() {
	case evaluationViewSummary:
		return o.summary()
	case evaluationViewCases:
		return o.Detail.Cases
	case evaluationViewRegime:
		return o.regimeSplit()
	case evaluationViewRobustness:
		return o.robustness()
	case evaluationViewWalkForward:
		return o.Detail.WalkForward
	default:
		return o.Detail
	}
}

func (o EvaluationDetailOutput) NDJSONRows() any {
	switch o.view() {
	case evaluationViewSummary:
		return []EvaluationSummaryView{o.summary()}
	case evaluationViewRegime:
		return o.regimeSplit()
	case evaluationViewRobustness:
		return o.robustnessRows()
	case evaluationViewWalkForward:
		return o.Detail.WalkForward
	default:
		return o.Detail.Cases
	}
}

func (o EvaluationDetailOutput) CSVRows() any {
	switch o.view() {
	case evaluationViewSummary:
		return []EvaluationSummaryView{o.summary()}
	case evaluationViewRegime:
		return o.regimeSplit()
	case evaluationViewRobustness:
		return o.robustnessRows()
	case evaluationViewWalkForward:
		return o.Detail.WalkForward
	default:
		return o.Detail.Cases
	}
}

func (o EvaluationDetailOutput) TableRows() ([]string, [][]string) {
	switch o.view() {
	case evaluationViewSummary:
		summary := o.summary()
		return []string{"id", "name", "strategy", "base_run", "cases", "walk_forward", "best_case", "spec_hash", "strategy_hash", "created_at"}, [][]string{{
			summary.ID,
			summary.Name,
			summary.StrategyName,
			summary.BaseRunName,
			fmt.Sprint(summary.CaseCount),
			fmt.Sprint(summary.WalkForwardStepCount),
			summary.BestCaseID,
			summary.SpecHash,
			summary.StrategySpecHash,
			summary.CreatedAt.UTC().Format(time.RFC3339),
		}}
	case evaluationViewRegime:
		split := o.regimeSplit()
		rows := make([][]string, 0, len(split))
		for _, item := range split {
			rows = append(rows, []string{
				item.Tag,
				fmt.Sprint(item.CaseCount),
				fmt.Sprint(item.PassedCount),
				item.Objective,
				item.BestCaseID,
				fmt.Sprintf("%.6f", item.BestObjective),
				fmt.Sprintf("%.6f", item.AverageObjective),
			})
		}
		return []string{"tag", "cases", "passed", "objective", "best_case", "best_objective", "avg_objective"}, rows
	case evaluationViewRobustness:
		items := o.robustnessRows()
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				item.Parameter,
				fmt.Sprint(item.CaseCount),
				fmt.Sprint(item.ValueCount),
				item.BestValue,
				item.BestCaseID,
				fmt.Sprintf("%.6f", item.BestObjective),
				item.WorstValue,
				fmt.Sprintf("%.6f", item.ObjectiveRange),
				fmt.Sprintf("%.6f", item.TopNOverlap),
				fmt.Sprintf("%.6f", item.OutOfSampleDegradationPct),
			})
		}
		return []string{"parameter", "cases", "values", "best_value", "best_case", "best_objective", "worst_value", "objective_range", "top_n_overlap", "oos_degradation_pct"}, rows
	case evaluationViewWalkForward:
		rows := make([][]string, 0, len(o.Detail.WalkForward))
		for _, item := range o.Detail.WalkForward {
			rows = append(rows, []string{
				fmt.Sprint(item.StepIndex),
				item.TrainFrom,
				item.TrainTo,
				item.TestFrom,
				item.TestTo,
				item.TrainCaseID,
				item.TestCaseID,
				fmt.Sprintf("%.6f", item.TrainObjective),
				item.StrategyHash,
				item.RunHash,
				item.DataFingerprint,
				item.ResultHash,
			})
		}
		return []string{"step", "train_from", "train_to", "test_from", "test_to", "train_case", "test_case", "train_objective", "strategy_hash", "run_hash", "data_fingerprint", "result_hash"}, rows
	}
	rows := make([][]string, 0, len(o.Detail.Cases))
	for _, item := range o.Detail.Cases {
		rows = append(rows, []string{
			item.CaseID,
			item.PeriodFrom,
			item.PeriodTo,
			item.Status,
			fmt.Sprint(item.Rank),
			item.Objective,
			fmt.Sprintf("%.6f", item.ObjectiveValue),
			item.StrategyHash,
			item.RunHash,
			item.DataFingerprint,
			item.ResultHash,
		})
	}
	return []string{"case", "from", "to", "status", "rank", "objective", "value", "strategy_hash", "run_hash", "data_fingerprint", "result_hash"}, rows
}

func (o EvaluationDetailOutput) view() string {
	view, err := normalizeEvaluationView(o.View)
	if err != nil {
		return evaluationViewRaw
	}
	return view
}

func (o EvaluationDetailOutput) summary() EvaluationSummaryView {
	experiment := o.Detail.Experiment
	bestCaseID := ""
	for _, item := range o.Detail.Cases {
		if item.Rank == 1 {
			bestCaseID = item.CaseID
			break
		}
	}
	return EvaluationSummaryView{
		ID:                   experiment.ID,
		Name:                 experiment.Name,
		StrategyName:         experiment.StrategyName,
		BaseRunName:          experiment.BaseRunName,
		SchemaVersion:        experiment.SchemaVersion,
		SpecHash:             experiment.SpecHash,
		StrategySpecHash:     experiment.StrategySpecHash,
		DataFrom:             experiment.DataFrom,
		DataTo:               experiment.DataTo,
		CaseCount:            len(o.Detail.Cases),
		WalkForwardStepCount: len(o.Detail.WalkForward),
		BestCaseID:           bestCaseID,
		CreatedAt:            experiment.CreatedAt,
	}
}

func (o EvaluationDetailOutput) regimeSplit() []EvaluationRegimeSplitView {
	return evaluationRegimeSplitViews(core.BuildRegimeSplit(o.evaluationCaseResults()))
}

func (o EvaluationDetailOutput) evaluationCaseResults() []core.EvaluationCaseResult {
	results := make([]core.EvaluationCaseResult, 0, len(o.Detail.Cases))
	for _, item := range o.Detail.Cases {
		var tags []string
		_ = json.Unmarshal(item.RegimeTagsJSON, &tags)
		metrics := core.Metrics{}
		_ = json.Unmarshal(item.MetricsJSON, &metrics)
		parameters := map[string]any{}
		_ = json.Unmarshal(item.ParameterJSON, &parameters)
		results = append(results, core.EvaluationCaseResult{
			CaseID:            item.CaseID,
			Parameters:        parameters,
			Metrics:           metrics,
			RegimeTags:        tags,
			PassedConstraints: item.PassedConstraints,
			Rank:              item.Rank,
			Objective:         item.Objective,
			ObjectiveValue:    item.ObjectiveValue,
		})
	}
	return results
}

func evaluationRegimeSplitViews(split []core.RegimeSplitResult) []EvaluationRegimeSplitView {
	out := make([]EvaluationRegimeSplitView, 0, len(split))
	for _, item := range split {
		out = append(out, EvaluationRegimeSplitView{
			Tag:              item.Tag,
			CaseCount:        item.CaseCount,
			PassedCount:      item.PassedCount,
			Objective:        item.Objective,
			BestCaseID:       item.BestCaseID,
			BestObjective:    item.BestObjective,
			AverageObjective: item.AverageObjective,
			AverageMetrics:   item.AverageMetrics,
		})
	}
	return out
}

func (o EvaluationDetailOutput) robustness() core.RobustnessReport {
	spec := core.EvaluationSpec{}
	_ = json.Unmarshal(o.Detail.Experiment.SpecJSON, &spec)
	return core.BuildRobustnessReport(o.evaluationCaseResults(), o.walkForwardStepResults(), spec.Ranking, 3)
}

func (o EvaluationDetailOutput) walkForwardStepResults() []core.WalkForwardStepResult {
	out := make([]core.WalkForwardStepResult, 0, len(o.Detail.WalkForward))
	for _, item := range o.Detail.WalkForward {
		parameters := map[string]any{}
		_ = json.Unmarshal(item.SelectedParameterJSON, &parameters)
		metrics := core.Metrics{}
		_ = json.Unmarshal(item.TestMetricsJSON, &metrics)
		out = append(out, core.WalkForwardStepResult{
			Index:              item.StepIndex,
			SelectedParameters: parameters,
			TrainCaseID:        item.TrainCaseID,
			TestCaseID:         item.TestCaseID,
			TrainObjective:     item.TrainObjective,
			TestMetrics:        metrics,
			DataFingerprint:    item.DataFingerprint,
			ResultHash:         item.ResultHash,
		})
	}
	return out
}

func (o EvaluationDetailOutput) robustnessRows() []EvaluationRobustnessRow {
	report := o.robustness()
	rows := make([]EvaluationRobustnessRow, 0, len(report.ParameterSensitivity))
	for _, item := range report.ParameterSensitivity {
		row := EvaluationRobustnessRow{
			Parameter:                 item.Parameter,
			CaseCount:                 item.CaseCount,
			ValueCount:                item.ValueCount,
			BestValue:                 item.BestValue,
			BestCaseID:                item.BestCaseID,
			BestObjective:             item.BestObjective,
			WorstValue:                item.WorstValue,
			ObjectiveRange:            item.ObjectiveRange,
			TopNOverlap:               report.TopNStability.AverageOverlap,
			OutOfSampleDegradationPct: 0,
		}
		if report.OutOfSampleDegradation != nil {
			row.OutOfSampleDegradationPct = report.OutOfSampleDegradation.DegradationPct
		}
		rows = append(rows, row)
	}
	return rows
}

type EvaluationSummaryView struct {
	ID                   string    `json:"id" csv:"id"`
	Name                 string    `json:"name" csv:"name"`
	StrategyName         string    `json:"strategy_name" csv:"strategy_name"`
	BaseRunName          string    `json:"base_run_name" csv:"base_run_name"`
	SchemaVersion        int       `json:"schema_version" csv:"schema_version"`
	SpecHash             string    `json:"spec_hash" csv:"spec_hash"`
	StrategySpecHash     string    `json:"strategy_spec_hash" csv:"strategy_spec_hash"`
	DataFrom             string    `json:"data_from" csv:"data_from"`
	DataTo               string    `json:"data_to" csv:"data_to"`
	CaseCount            int       `json:"case_count" csv:"case_count"`
	WalkForwardStepCount int       `json:"walk_forward_step_count" csv:"walk_forward_step_count"`
	BestCaseID           string    `json:"best_case_id,omitempty" csv:"best_case_id"`
	CreatedAt            time.Time `json:"created_at" csv:"created_at"`
}

type EvaluationRegimeSplitView struct {
	Tag              string       `json:"tag" csv:"tag"`
	CaseCount        int          `json:"case_count" csv:"case_count"`
	PassedCount      int          `json:"passed_count" csv:"passed_count"`
	Objective        string       `json:"objective,omitempty" csv:"objective"`
	BestCaseID       string       `json:"best_case_id,omitempty" csv:"best_case_id"`
	BestObjective    float64      `json:"best_objective" csv:"best_objective"`
	AverageObjective float64      `json:"average_objective" csv:"average_objective"`
	AverageMetrics   core.Metrics `json:"average_metrics,omitempty" csv:"-"`
}

type EvaluationRobustnessRow struct {
	Parameter                 string  `json:"parameter" csv:"parameter"`
	CaseCount                 int     `json:"case_count" csv:"case_count"`
	ValueCount                int     `json:"value_count" csv:"value_count"`
	BestValue                 string  `json:"best_value,omitempty" csv:"best_value"`
	BestCaseID                string  `json:"best_case_id,omitempty" csv:"best_case_id"`
	BestObjective             float64 `json:"best_objective" csv:"best_objective"`
	WorstValue                string  `json:"worst_value,omitempty" csv:"worst_value"`
	ObjectiveRange            float64 `json:"objective_range" csv:"objective_range"`
	TopNOverlap               float64 `json:"top_n_overlap" csv:"top_n_overlap"`
	OutOfSampleDegradationPct float64 `json:"out_of_sample_degradation_pct" csv:"out_of_sample_degradation_pct"`
}

const (
	evaluationViewRaw         = "raw"
	evaluationViewSummary     = "summary"
	evaluationViewCases       = "cases"
	evaluationViewRegime      = "regime"
	evaluationViewRobustness  = "robustness"
	evaluationViewWalkForward = "walk_forward"
)

func normalizeEvaluationView(value string) (string, error) {
	view := strings.ToLower(strings.TrimSpace(value))
	if view == "" {
		return evaluationViewRaw, nil
	}
	switch view {
	case evaluationViewRaw, evaluationViewSummary, evaluationViewCases, evaluationViewRegime, evaluationViewRobustness, evaluationViewWalkForward:
		return view, nil
	default:
		return "", oops.In("backtest_handler").With("view", value).Errorf("unsupported evaluation view: %s", value)
	}
}

type EvaluationRankOutput []backtestservice.SavedExperimentCase

func (o EvaluationRankOutput) JSONValue() any {
	return []backtestservice.SavedExperimentCase(o)
}

func (o EvaluationRankOutput) NDJSONRows() any {
	return o.JSONValue()
}

func (o EvaluationRankOutput) CSVRows() any {
	return o.JSONValue()
}

func (o EvaluationRankOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, item := range o {
		rows = append(rows, []string{
			fmt.Sprint(item.Rank),
			item.CaseID,
			item.PeriodFrom,
			item.PeriodTo,
			item.Status,
			item.Objective,
			fmt.Sprintf("%.6f", item.ObjectiveValue),
			item.StrategyHash,
			item.RunHash,
			item.DataFingerprint,
			item.ResultHash,
		})
	}
	return []string{"rank", "case", "from", "to", "status", "objective", "value", "strategy_hash", "run_hash", "data_fingerprint", "result_hash"}, rows
}

type evaluationCaseRow struct {
	CaseID            string  `json:"case_id" csv:"case_id"`
	CaseName          string  `json:"case_name" csv:"case_name"`
	From              string  `json:"from" csv:"from"`
	To                string  `json:"to" csv:"to"`
	PassedConstraints bool    `json:"passed_constraints" csv:"passed_constraints"`
	Rank              int     `json:"rank" csv:"rank"`
	Objective         string  `json:"objective" csv:"objective"`
	ObjectiveValue    float64 `json:"objective_value" csv:"objective_value"`
	DataFingerprint   string  `json:"data_fingerprint" csv:"data_fingerprint"`
	ResultHash        string  `json:"result_hash" csv:"result_hash"`
}

func evaluationCaseRows(cases []core.EvaluationCaseResult) []evaluationCaseRow {
	rows := make([]evaluationCaseRow, 0, len(cases))
	for _, item := range cases {
		rows = append(rows, evaluationCaseRow{
			CaseID:            item.CaseID,
			CaseName:          item.CaseName,
			From:              item.Period.From.Format(time.DateOnly),
			To:                item.Period.To.Format(time.DateOnly),
			PassedConstraints: item.PassedConstraints,
			Rank:              item.Rank,
			Objective:         item.Objective,
			ObjectiveValue:    item.ObjectiveValue,
			DataFingerprint:   item.Result.DataFingerprint,
			ResultHash:        item.Result.ResultHash,
		})
	}
	return rows
}

func evaluationCaseTableHeader() []string {
	return []string{"rank", "case", "from", "to", "pass", "objective", "value", "data_fingerprint", "result_hash"}
}

func evaluationCaseTableRow(item core.EvaluationCaseResult) []string {
	return []string{
		fmt.Sprint(item.Rank),
		item.CaseID,
		item.Period.From.Format(time.DateOnly),
		item.Period.To.Format(time.DateOnly),
		fmt.Sprint(item.PassedConstraints),
		item.Objective,
		fmt.Sprintf("%.6f", item.ObjectiveValue),
		item.Result.DataFingerprint,
		item.Result.ResultHash,
	}
}

type BacktestStrategyOutput struct {
	Detail      backtestservice.SavedStrategyDetail
	IncludeSpec bool
}

type BacktestStrategyView struct {
	Name          string             `json:"name" csv:"name"`
	Version       int                `json:"version" csv:"version"`
	SchemaVersion int                `json:"schema_version" csv:"schema_version"`
	Indicators    map[string]string  `json:"indicators,omitempty" csv:"-"`
	CreatedAt     string             `json:"created_at" csv:"created_at"`
	UpdatedAt     string             `json:"updated_at" csv:"updated_at"`
	SpecHash      string             `json:"spec_hash" csv:"spec_hash"`
	Spec          *core.StrategySpec `json:"spec,omitempty" csv:"-"`
}

func (o BacktestStrategyOutput) JSONValue() any {
	return o.view()
}

func (o BacktestStrategyOutput) NDJSONRows() any {
	return o.view()
}

func (o BacktestStrategyOutput) CSVRows() any {
	return []BacktestStrategyView{o.view()}
}

func (o BacktestStrategyOutput) TableRows() ([]string, [][]string) {
	view := o.view()
	return []string{"name", "version", "schema_version", "spec_hash", "updated_at"}, [][]string{{
		view.Name,
		fmt.Sprint(view.Version),
		fmt.Sprint(view.SchemaVersion),
		view.SpecHash,
		view.UpdatedAt,
	}}
}

func (o BacktestStrategyOutput) view() BacktestStrategyView {
	detail := o.Detail
	indicators := make(map[string]string, len(detail.Spec.Indicators))
	for alias, indicator := range detail.Spec.Indicators {
		indicators[alias] = indicator.ID
	}
	view := BacktestStrategyView{
		Name:          detail.Strategy.Name,
		Version:       detail.ActiveVersion.Version,
		SchemaVersion: detail.ActiveVersion.SchemaVersion,
		Indicators:    indicators,
		CreatedAt:     detail.Strategy.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     detail.Strategy.UpdatedAt.UTC().Format(time.RFC3339),
		SpecHash:      detail.ActiveVersion.SpecHash,
	}
	if o.IncludeSpec {
		spec := detail.Spec
		view.Spec = &spec
	}
	return view
}

type BacktestStrategyListOutput []BacktestStrategyOutput

func (o BacktestStrategyListOutput) JSONValue() any {
	rows := make([]BacktestStrategyView, 0, len(o))
	for _, item := range o {
		rows = append(rows, item.view())
	}
	return rows
}

func (o BacktestStrategyListOutput) NDJSONRows() any {
	return o.JSONValue()
}

func (o BacktestStrategyListOutput) CSVRows() any {
	return o.JSONValue()
}

func (o BacktestStrategyListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o))
	for _, item := range o {
		view := item.view()
		rows = append(rows, []string{
			view.Name,
			fmt.Sprint(view.Version),
			fmt.Sprint(view.SchemaVersion),
			view.SpecHash,
			view.UpdatedAt,
		})
	}
	return []string{"name", "version", "schema_version", "spec_hash", "updated_at"}, rows
}

type DeleteBacktestStrategyResult struct {
	Name    string `json:"name" csv:"name"`
	Deleted bool   `json:"deleted" csv:"deleted"`
}

func (r DeleteBacktestStrategyResult) JSONValue() any {
	return r
}

func (r DeleteBacktestStrategyResult) NDJSONRows() any {
	return r
}

func (r DeleteBacktestStrategyResult) CSVRows() any {
	return []DeleteBacktestStrategyResult{r}
}

func (r DeleteBacktestStrategyResult) TableRows() ([]string, [][]string) {
	return []string{"name", "deleted"}, [][]string{{r.Name, fmt.Sprint(r.Deleted)}}
}
