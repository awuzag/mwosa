package universe

import (
	"context"
	"os"
	"strings"
	"time"

	core "github.com/awuzag/mwosa/packages/universe"
	provider "github.com/awuzag/mwosa/providers/core"
	"github.com/awuzag/mwosa/service/daily"
	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

const KindStrategySet = "StrategySet"

type StrategySetSpec struct {
	Kind          string              `json:"kind" yaml:"kind"`
	SchemaVersion int                 `json:"schema_version" yaml:"schema_version"`
	Name          string              `json:"name" yaml:"name"`
	Spec          StrategySetBodySpec `json:"spec" yaml:"spec"`
}

type StrategySetBodySpec struct {
	Regime     string                       `json:"regime" yaml:"regime"`
	RegimeFile string                       `json:"regime_file,omitempty" yaml:"regime_file,omitempty"`
	Routes     map[string]StrategyRouteSpec `json:"routes" yaml:"routes"`
}

type StrategyRouteSpec struct {
	Strategy      string   `json:"strategy" yaml:"strategy"`
	Version       string   `json:"version,omitempty" yaml:"version,omitempty"`
	SpecHash      string   `json:"spec_hash,omitempty" yaml:"spec_hash,omitempty"`
	MinConfidence *float64 `json:"min_confidence,omitempty" yaml:"min_confidence,omitempty"`
}

type StrategySetSelectionResult struct {
	Kind          string                  `json:"kind"`
	Name          string                  `json:"name"`
	AsOf          string                  `json:"as_of"`
	Regime        core.MarketRegimeResult `json:"regime"`
	SelectedRoute StrategyRouteSpec       `json:"selected_route"`
	Hints         []string                `json:"hints,omitempty"`
}

func LoadMarketRegimeFile(ctx context.Context, path string) (core.MarketRegimeSpec, error) {
	if err := ctx.Err(); err != nil {
		return core.MarketRegimeSpec{}, oops.In("market_regime_yaml").With("path", path).Wrap(err)
	}
	file, err := os.Open(path)
	if err != nil {
		return core.MarketRegimeSpec{}, oops.In("market_regime_yaml").With("path", path).Wrapf(err, "read market regime YAML file")
	}
	defer file.Close()
	var spec core.MarketRegimeSpec
	if err := yaml.NewDecoder(file).Decode(&spec); err != nil {
		return core.MarketRegimeSpec{}, oops.In("market_regime_yaml").With("path", path).Wrapf(err, "decode market regime YAML")
	}
	if spec.SchemaVersion == 0 {
		spec.SchemaVersion = 1
	}
	return spec, nil
}

func LoadStrategySetFile(ctx context.Context, path string) (StrategySetSpec, error) {
	if err := ctx.Err(); err != nil {
		return StrategySetSpec{}, oops.In("strategy_set_yaml").With("path", path).Wrap(err)
	}
	file, err := os.Open(path)
	if err != nil {
		return StrategySetSpec{}, oops.In("strategy_set_yaml").With("path", path).Wrapf(err, "read strategy set YAML file")
	}
	defer file.Close()
	var spec StrategySetSpec
	if err := yaml.NewDecoder(file).Decode(&spec); err != nil {
		return StrategySetSpec{}, oops.In("strategy_set_yaml").With("path", path).Wrapf(err, "decode strategy set YAML")
	}
	if spec.SchemaVersion == 0 {
		spec.SchemaVersion = 1
	}
	return spec, nil
}

func (r Runner) InspectMarketRegime(ctx context.Context, path string, asOfOverride string) (core.MarketRegimeResult, error) {
	spec, err := LoadMarketRegimeFile(ctx, path)
	if err != nil {
		return core.MarketRegimeResult{}, err
	}
	asOf, err := marketRegimeAsOf(spec, asOfOverride)
	if err != nil {
		return core.MarketRegimeResult{}, err
	}
	bars, err := r.loadMarketRegimeBars(ctx, spec, asOf)
	if err != nil {
		return core.MarketRegimeResult{}, err
	}
	return core.EvaluateMarketRegime(ctx, spec, bars, asOf)
}

func (r Runner) InspectStrategySet(ctx context.Context, path string, asOfOverride string) (StrategySetSelectionResult, error) {
	spec, err := LoadStrategySetFile(ctx, path)
	if err != nil {
		return StrategySetSelectionResult{}, err
	}
	if err := validateStrategySetSpec(spec); err != nil {
		return StrategySetSelectionResult{}, err
	}
	regimePath := resolveFilePath(path, spec.Spec.RegimeFile)
	regime, err := r.InspectMarketRegime(ctx, regimePath, asOfOverride)
	if err != nil {
		return StrategySetSelectionResult{}, err
	}
	if spec.Spec.Regime != regime.Name {
		return StrategySetSelectionResult{}, oops.In("strategy_set").With("name", spec.Name, "regime", spec.Spec.Regime, "loaded_regime", regime.Name).New("strategy set regime does not match loaded market regime")
	}
	route, ok := spec.Spec.Routes[regime.Regime]
	if !ok {
		return StrategySetSelectionResult{}, oops.In("strategy_set").With("name", spec.Name, "regime", regime.Regime).New("strategy set route not found for regime")
	}
	if route.MinConfidence != nil && regime.Confidence < *route.MinConfidence {
		return StrategySetSelectionResult{}, oops.In("strategy_set").
			With("name", spec.Name, "regime", regime.Regime, "confidence", regime.Confidence, "min_confidence", *route.MinConfidence).
			New("strategy set route confidence is below minimum")
	}
	hints := []string(nil)
	if strings.TrimSpace(route.Version) == "" || route.Version == "latest" {
		hints = append(hints, "latest version is convenient for screening, but pin version or spec_hash for reproducible backtests")
	}
	return StrategySetSelectionResult{
		Kind:          KindStrategySet,
		Name:          spec.Name,
		AsOf:          regime.AsOf,
		Regime:        regime,
		SelectedRoute: route,
		Hints:         hints,
	}, nil
}

func marketRegimeAsOf(spec core.MarketRegimeSpec, override string) (time.Time, error) {
	text := strings.TrimSpace(override)
	if text == "" {
		return time.Time{}, oops.In("market_regime").With("name", spec.Name).New("market regime as_of is required")
	}
	parsed, err := time.Parse(time.DateOnly, text)
	if err != nil {
		return time.Time{}, oops.In("market_regime").With("as_of", text).Wrapf(err, "parse market regime as_of")
	}
	return parsed, nil
}

func (r Runner) loadMarketRegimeBars(ctx context.Context, spec core.MarketRegimeSpec, asOf time.Time) ([]core.Bar, error) {
	benchmark := spec.Spec.Benchmark
	market := provider.Market(benchmark.Market)
	if market == "" {
		market = provider.MarketKRX
	}
	evaluation, err := core.NormalizeMarketRegimeEvaluationSpec(spec.Spec.Evaluation)
	if err != nil {
		return nil, err
	}
	requiredBars := core.MarketRegimeRequiredBarCount(evaluation)
	calendarLookbackDays := requiredBars*2 + 30
	rows, err := r.reader.QueryDailyBars(ctx, daily.Query{
		Market:       market,
		SecurityType: provider.SecurityType(benchmark.SecurityType),
		Symbol:       benchmark.Symbol,
		From:         asOf.AddDate(0, 0, -calendarLookbackDays).Format(time.DateOnly),
		To:           asOf.Format(time.DateOnly),
	})
	if err != nil {
		return nil, oops.In("market_regime").With("symbol", benchmark.Symbol, "as_of", asOf.Format(time.DateOnly)).Wrapf(err, "query market regime benchmark bars")
	}
	bars := make([]core.Bar, 0, len(rows))
	for _, row := range rows {
		bar, err := canonicalDailyBarToUniverseBar(row)
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	return bars, nil
}

func validateStrategySetSpec(spec StrategySetSpec) error {
	errb := oops.In("strategy_set").With("name", spec.Name)
	if spec.Kind != KindStrategySet {
		return errb.With("kind", spec.Kind).New("strategy set kind must be StrategySet")
	}
	if spec.SchemaVersion != 1 {
		return errb.With("schema_version", spec.SchemaVersion).New("unsupported strategy set schema version")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return errb.New("strategy set name is required")
	}
	if strings.TrimSpace(spec.Spec.Regime) == "" {
		return errb.New("strategy set regime is required")
	}
	if strings.TrimSpace(spec.Spec.RegimeFile) == "" {
		return errb.New("strategy set regime_file is required for file-based inspection")
	}
	if len(spec.Spec.Routes) == 0 {
		return errb.New("strategy set requires routes")
	}
	for regime, route := range spec.Spec.Routes {
		if strings.TrimSpace(route.Strategy) == "" {
			return errb.With("regime", regime).New("strategy set route strategy is required")
		}
		if strings.TrimSpace(route.Version) != "" && strings.TrimSpace(route.SpecHash) != "" {
			return errb.With("regime", regime).New("strategy set route requires either version or spec_hash, not both")
		}
		if route.MinConfidence != nil && (*route.MinConfidence < 0 || *route.MinConfidence > 1) {
			return errb.With("regime", regime, "min_confidence", *route.MinConfidence).New("strategy set route min_confidence must be between 0 and 1")
		}
	}
	return nil
}
