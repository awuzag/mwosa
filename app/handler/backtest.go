package handler

import (
	"context"
	"fmt"
	"time"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	backtestservice "github.com/ev3rlit/mwosa/service/backtest"
)

type Backtest struct {
	service backtestservice.Service
}

func NewBacktest(service backtestservice.Service) Backtest {
	return Backtest{service: service}
}

type ValidateBacktestRequest struct {
	Path string
}

type RunBacktestRequest struct {
	Path string
}

type ListBacktestStrategiesRequest struct{}

type InspectBacktestStrategyRequest struct {
	Name string
}

type InspectBacktestUniverseRequest struct {
	Path string
}

type UpdateBacktestStrategyRequest struct {
	Name     string
	YAMLPath string
}

type DeleteBacktestStrategyRequest struct {
	Name string
}

func (h Backtest) Validate(ctx context.Context, req ValidateBacktestRequest) (BacktestValidationOutput, error) {
	result, err := h.service.Validate(ctx, req.Path)
	if err != nil {
		return BacktestValidationOutput{}, err
	}
	return BacktestValidationOutput{Result: result}, nil
}

func (h Backtest) Run(ctx context.Context, req RunBacktestRequest) (BacktestRunOutput, error) {
	result, err := h.service.Run(ctx, req.Path)
	if err != nil {
		return BacktestRunOutput{}, err
	}
	return BacktestRunOutput{Result: result}, nil
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

func (h Backtest) InspectStrategy(ctx context.Context, req InspectBacktestStrategyRequest) (BacktestStrategyOutput, error) {
	detail, err := h.service.InspectStrategy(ctx, req.Name)
	if err != nil {
		return BacktestStrategyOutput{}, err
	}
	return BacktestStrategyOutput{Detail: detail, IncludeSpec: true}, nil
}

func (h Backtest) InspectUniverse(ctx context.Context, req InspectBacktestUniverseRequest) (BacktestUniverseOutput, error) {
	explain, err := h.service.InspectUniverse(ctx, req.Path)
	if err != nil {
		return BacktestUniverseOutput{}, err
	}
	return BacktestUniverseOutput{Explain: explain}, nil
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
}

func (o BacktestValidationOutput) JSONValue() any {
	return o.Result
}

func (o BacktestValidationOutput) NDJSONRows() any {
	return o.Result
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
}

func (o BacktestUniverseOutput) JSONValue() any {
	return o.Explain
}

func (o BacktestUniverseOutput) NDJSONRows() any {
	return o.Explain.Snapshots
}

func (o BacktestUniverseOutput) CSVRows() any {
	return []backtestUniverseRow{universeRow(o.Explain)}
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

type BacktestRunOutput struct {
	Result core.Result
}

func (o BacktestRunOutput) JSONValue() any {
	return o.Result
}

func (o BacktestRunOutput) NDJSONRows() any {
	return o.Result.Trades
}

func (o BacktestRunOutput) CSVRows() any {
	return o.Result.Trades
}

func (o BacktestRunOutput) TableRows() ([]string, [][]string) {
	result := o.Result
	header := []string{"strategy", "run", "symbols"}
	row := []string{
		result.StrategyName,
		result.RunName,
		fmt.Sprint(len(result.Symbols)),
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
