package backtest

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"

	core "github.com/awuzag/mwosa/packages/backtest"
	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

type Bundle struct {
	Strategy   core.StrategySpec
	Run        core.BacktestRunSpec
	Evaluation core.EvaluationSpec
}

func LoadFile(ctx context.Context, path string) (Bundle, error) {
	bundle, err := loadFile(ctx, path, true)
	if err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func LoadStrategyFile(ctx context.Context, path string) (core.StrategySpec, error) {
	bundle, err := loadFile(ctx, path, false)
	if err != nil {
		return core.StrategySpec{}, err
	}
	return bundle.Strategy, nil
}

func LoadEvaluationFile(ctx context.Context, path string) (Bundle, error) {
	bundle, err := loadFile(ctx, path, true)
	if err != nil {
		return Bundle{}, err
	}
	if bundle.Evaluation.Kind == "" {
		return Bundle{}, oops.In("backtest_yaml").New("YAML stream requires Evaluation document")
	}
	return bundle, nil
}

func loadFile(ctx context.Context, path string, requireRun bool) (Bundle, error) {
	if err := ctx.Err(); err != nil {
		return Bundle{}, oops.In("backtest_yaml").With("path", path).Wrap(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return Bundle{}, oops.In("backtest_yaml").With("path", path).Wrapf(err, "read backtest YAML file")
	}
	if err := ctx.Err(); err != nil {
		return Bundle{}, oops.In("backtest_yaml").With("path", path).Wrap(err)
	}
	return decode(ctx, bytes.NewReader(payload), requireRun)
}

func Decode(ctx context.Context, reader io.Reader) (Bundle, error) {
	return decode(ctx, reader, true)
}

func DecodeStrategy(ctx context.Context, reader io.Reader) (core.StrategySpec, error) {
	bundle, err := decode(ctx, reader, false)
	if err != nil {
		return core.StrategySpec{}, err
	}
	return bundle.Strategy, nil
}

func decode(ctx context.Context, reader io.Reader, requireRun bool) (Bundle, error) {
	if err := ctx.Err(); err != nil {
		return Bundle{}, oops.In("backtest_yaml").Wrap(err)
	}
	decoder := yaml.NewDecoder(reader)
	var bundle Bundle
	for {
		var node yaml.Node
		err := decoder.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Bundle{}, oops.In("backtest_yaml").Wrapf(err, "decode YAML document")
		}
		if emptyDocument(node) {
			continue
		}
		kind, err := documentKind(&node)
		if err != nil {
			return Bundle{}, err
		}
		switch kind {
		case core.KindStrategy:
			strategy, err := decodeStrategy(&node)
			if err != nil {
				return Bundle{}, err
			}
			bundle.Strategy = strategy
		case core.KindBacktestRun:
			run, err := decodeRun(&node)
			if err != nil {
				return Bundle{}, err
			}
			bundle.Run = run
		case core.KindEvaluation:
			evaluation, err := decodeEvaluation(&node)
			if err != nil {
				return Bundle{}, err
			}
			bundle.Evaluation = evaluation
		default:
			return Bundle{}, oops.In("backtest_yaml").With("kind", kind).New("unsupported YAML document kind")
		}
	}
	if bundle.Strategy.Kind == "" {
		return Bundle{}, oops.In("backtest_yaml").New("YAML stream requires Strategy document")
	}
	if requireRun && bundle.Run.Kind == "" {
		return Bundle{}, oops.In("backtest_yaml").New("YAML stream requires BacktestRun document")
	}
	return bundle, nil
}

type strategyDocument struct {
	Kind          string                       `yaml:"kind"`
	SchemaVersion int                          `yaml:"schema_version"`
	Name          string                       `yaml:"name"`
	Description   string                       `yaml:"description"`
	Tags          []string                     `yaml:"tags"`
	Indicators    map[string]indicatorDocument `yaml:"indicators"`
	Entry         yaml.Node                    `yaml:"entry"`
	Exit          yaml.Node                    `yaml:"exit"`
	Entries       yaml.Node                    `yaml:"entries"`
	Exits         yaml.Node                    `yaml:"exits"`
	Rebalance     yaml.Node                    `yaml:"rebalance"`
	Stops         yaml.Node                    `yaml:"stops"`
	Sizing        core.SizingSpec              `yaml:"sizing"`
	Risk          core.RiskSpec                `yaml:"risk"`
}

type indicatorDocument struct {
	ID      string             `yaml:"id"`
	Source  yaml.Node          `yaml:"source"`
	Compare yaml.Node          `yaml:"compare"`
	Params  map[string]float64 `yaml:"params"`
	Output  string             `yaml:"output"`
}

type timeframeValueDocument struct {
	ID    string    `yaml:"id"`
	Value yaml.Node `yaml:"value"`
}

type runDocument struct {
	Kind          string             `yaml:"kind"`
	SchemaVersion int                `yaml:"schema_version"`
	Name          string             `yaml:"name"`
	Strategy      core.StrategyRef   `yaml:"strategy"`
	Data          core.DataSpec      `yaml:"data"`
	Universe      core.UniverseSpec  `yaml:"universe"`
	Benchmark     core.BenchmarkSpec `yaml:"benchmark"`
	Portfolio     core.PortfolioSpec `yaml:"portfolio"`
	Execution     core.ExecutionSpec `yaml:"execution"`
	Report        reportDocument     `yaml:"report"`
}

type evaluationDocument struct {
	Kind          string                       `yaml:"kind"`
	SchemaVersion int                          `yaml:"schema_version"`
	Name          string                       `yaml:"name"`
	Strategy      core.StrategyRef             `yaml:"strategy"`
	BaseRun       core.EvaluationBaseRunRef    `yaml:"base_run"`
	Periods       core.EvaluationPeriodsSpec   `yaml:"periods"`
	Parameters    map[string][]any             `yaml:"parameters"`
	Search        core.EvaluationSearchSpec    `yaml:"search"`
	Metrics       yaml.Node                    `yaml:"metrics"`
	Constraints   core.EvaluationConstraintSet `yaml:"constraints"`
	Ranking       core.EvaluationRankingSpec   `yaml:"ranking"`
	Regime        core.EvaluationRegimeSpec    `yaml:"regime"`
	Execution     core.EvaluationExecutionSpec `yaml:"execution"`
	WalkForward   core.WalkForwardSpec         `yaml:"walk_forward"`
}

type reportDocument struct {
	Metrics yaml.Node `yaml:"metrics"`
}

func decodeStrategy(node *yaml.Node) (core.StrategySpec, error) {
	var document strategyDocument
	if err := node.Decode(&document); err != nil {
		return core.StrategySpec{}, oops.In("backtest_yaml").Wrapf(err, "decode Strategy document")
	}
	indicators := make(map[string]core.IndicatorSpec, len(document.Indicators))
	for alias, raw := range document.Indicators {
		source, err := decodeValue(&raw.Source)
		if err != nil {
			return core.StrategySpec{}, oops.In("backtest_yaml").With("indicator_alias", alias).Wrap(err)
		}
		var compare core.ValueExpr
		if !emptyYAMLNode(&raw.Compare) {
			compare, err = decodeValue(&raw.Compare)
			if err != nil {
				return core.StrategySpec{}, oops.In("backtest_yaml").With("indicator_alias", alias).Wrap(err)
			}
		}
		indicators[alias] = core.IndicatorSpec{
			ID:      raw.ID,
			Source:  source,
			Compare: compare,
			Params:  raw.Params,
			Output:  raw.Output,
		}
	}
	entry, err := decodeOptionalRule("entry", &document.Entry)
	if err != nil {
		return core.StrategySpec{}, oops.In("backtest_yaml").Wrapf(err, "decode entry rule")
	}
	exit, err := decodeOptionalRule("exit", &document.Exit)
	if err != nil {
		return core.StrategySpec{}, oops.In("backtest_yaml").Wrapf(err, "decode exit rule")
	}
	entries, err := decodeRuleList("entries", &document.Entries)
	if err != nil {
		return core.StrategySpec{}, err
	}
	exits, err := decodeRuleList("exits", &document.Exits)
	if err != nil {
		return core.StrategySpec{}, err
	}
	rebalance, err := decodeRuleList("rebalance", &document.Rebalance)
	if err != nil {
		return core.StrategySpec{}, err
	}
	stops, err := decodeRuleList("stops", &document.Stops)
	if err != nil {
		return core.StrategySpec{}, err
	}
	return core.NormalizeStrategySpec(core.StrategySpec{
		Kind:          document.Kind,
		SchemaVersion: document.SchemaVersion,
		Name:          document.Name,
		Description:   document.Description,
		Tags:          document.Tags,
		Indicators:    indicators,
		Entry:         entry,
		Exit:          exit,
		Entries:       entries,
		Exits:         exits,
		Rebalance:     rebalance,
		Stops:         stops,
		Sizing:        document.Sizing,
		Risk:          document.Risk,
	}), nil
}

func decodeRun(node *yaml.Node) (core.BacktestRunSpec, error) {
	var document runDocument
	if err := node.Decode(&document); err != nil {
		return core.BacktestRunSpec{}, oops.In("backtest_yaml").Wrapf(err, "decode BacktestRun document")
	}
	metrics, err := decodeMetricSelection(&document.Report.Metrics)
	if err != nil {
		return core.BacktestRunSpec{}, oops.In("backtest_yaml").Wrapf(err, "decode report metrics")
	}
	return core.BacktestRunSpec{
		Kind:          document.Kind,
		SchemaVersion: document.SchemaVersion,
		Name:          document.Name,
		Strategy:      document.Strategy,
		Data:          document.Data,
		Universe:      document.Universe,
		Benchmark:     document.Benchmark,
		Portfolio:     document.Portfolio,
		Execution:     document.Execution,
		Report:        core.ReportSpec{Metrics: metrics},
	}, nil
}

func decodeEvaluation(node *yaml.Node) (core.EvaluationSpec, error) {
	var document evaluationDocument
	if err := node.Decode(&document); err != nil {
		return core.EvaluationSpec{}, oops.In("backtest_yaml").Wrapf(err, "decode Evaluation document")
	}
	metrics, err := decodeMetricSelection(&document.Metrics)
	if err != nil {
		return core.EvaluationSpec{}, oops.In("backtest_yaml").Wrapf(err, "decode evaluation metrics")
	}
	return core.EvaluationSpec{
		Kind:          document.Kind,
		SchemaVersion: document.SchemaVersion,
		Name:          document.Name,
		Strategy:      document.Strategy,
		BaseRun:       document.BaseRun,
		Periods:       document.Periods,
		Parameters:    document.Parameters,
		Search:        document.Search,
		Metrics:       metrics,
		Constraints:   document.Constraints,
		Ranking:       document.Ranking,
		Regime:        document.Regime,
		Execution:     document.Execution,
		WalkForward:   document.WalkForward,
	}, nil
}

func decodeOptionalRule(name string, node *yaml.Node) (core.RuleExpr, error) {
	if emptyYAMLNode(node) {
		return core.RuleExpr{}, nil
	}
	rule, err := decodeRule(node)
	if err != nil {
		return core.RuleExpr{}, oops.In("backtest_yaml").With("field", name).Wrap(err)
	}
	return rule, nil
}

func decodeRuleList(name string, node *yaml.Node) ([]core.RuleExpr, error) {
	if emptyYAMLNode(node) {
		return nil, nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	if node.Kind != yaml.SequenceNode {
		return nil, oops.In("backtest_yaml").With("field", name).New("rule list requires sequence")
	}
	rules := make([]core.RuleExpr, 0, len(node.Content))
	for index, child := range node.Content {
		rule, err := decodeRule(child)
		if err != nil {
			return nil, oops.In("backtest_yaml").With("field", name, "index", index).Wrap(err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func decodeMetricSelection(node *yaml.Node) (core.MetricSelectionSpec, error) {
	if node == nil || node.Kind == 0 {
		return core.MetricSelectionSpec{}, nil
	}
	switch node.Kind {
	case yaml.SequenceNode:
		include := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			include = append(include, child.Value)
		}
		return core.MetricSelectionSpec{Include: include}, nil
	case yaml.MappingNode:
		var selection core.MetricSelectionSpec
		if err := node.Decode(&selection); err != nil {
			return core.MetricSelectionSpec{}, oops.In("backtest_yaml").Wrapf(err, "decode metric selection")
		}
		return selection, nil
	default:
		return core.MetricSelectionSpec{}, oops.In("backtest_yaml").New("report metrics must be mapping or sequence")
	}
}

func decodeRule(node *yaml.Node) (core.RuleExpr, error) {
	key, value, err := singleMapping(node)
	if err != nil {
		return core.RuleExpr{}, err
	}
	switch key {
	case "all", "any":
		if value.Kind != yaml.SequenceNode {
			return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", key).New("logical rule requires sequence")
		}
		rules := make([]core.RuleExpr, 0, len(value.Content))
		for _, child := range value.Content {
			rule, err := decodeRule(child)
			if err != nil {
				return core.RuleExpr{}, err
			}
			rules = append(rules, rule)
		}
		return core.RuleExpr{Operator: key, Rules: rules}, nil
	case "not":
		child := value
		if value.Kind == yaml.SequenceNode {
			if len(value.Content) != 1 {
				return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", key).New("not rule requires one child")
			}
			child = value.Content[0]
		}
		rule, err := decodeRule(child)
		if err != nil {
			return core.RuleExpr{}, err
		}
		return core.RuleExpr{Operator: key, Rule: &rule}, nil
	case "for_n_bars", "bars_since", "cooldown":
		return decodeTemporalRuleWithBars(key, value)
	case "weekly", "monthly", "first_trading_day", "position_exists":
		return decodeNoArgRule(key, value)
	default:
		if value.Kind != yaml.SequenceNode {
			return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", key).New("comparison rule requires sequence args")
		}
		args := make([]core.ValueExpr, 0, len(value.Content))
		for _, child := range value.Content {
			arg, err := decodeValue(child)
			if err != nil {
				return core.RuleExpr{}, err
			}
			args = append(args, arg)
		}
		return core.RuleExpr{Operator: key, Args: args}, nil
	}
}

func decodeNoArgRule(operator string, node *yaml.Node) (core.RuleExpr, error) {
	if node.Kind == yaml.MappingNode && len(node.Content) == 0 {
		return core.RuleExpr{Operator: operator}, nil
	}
	if node.Kind == yaml.SequenceNode && len(node.Content) == 0 {
		return core.RuleExpr{Operator: operator}, nil
	}
	if node.Kind == yaml.ScalarNode && (node.Tag == "!!null" || node.Value == "") {
		return core.RuleExpr{Operator: operator}, nil
	}
	return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", operator).New("no-arg rule requires empty mapping, empty sequence, or null")
}

func decodeTemporalRuleWithBars(operator string, node *yaml.Node) (core.RuleExpr, error) {
	if node.Kind != yaml.MappingNode {
		return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", operator).New("temporal rule requires mapping")
	}
	var bars *yaml.Node
	var child *yaml.Node
	for i := 0; i < len(node.Content); i += 2 {
		switch node.Content[i].Value {
		case "bars":
			bars = node.Content[i+1]
		case "rule":
			child = node.Content[i+1]
		}
	}
	if bars == nil {
		return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", operator).New("temporal rule requires bars")
	}
	if child == nil {
		return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", operator).New("temporal rule requires child rule")
	}
	barsValue, err := strconv.ParseFloat(bars.Value, 64)
	if err != nil {
		return core.RuleExpr{}, oops.In("backtest_yaml").With("operator", operator, "bars", bars.Value).Wrapf(err, "parse temporal rule count")
	}
	rule, err := decodeRule(child)
	if err != nil {
		return core.RuleExpr{}, err
	}
	return core.RuleExpr{
		Operator: operator,
		Rule:     &rule,
		Args:     []core.ValueExpr{{Kind: "value", Value: barsValue}},
	}, nil
}

func decodeValue(node *yaml.Node) (core.ValueExpr, error) {
	key, value, err := singleMapping(node)
	if err != nil {
		return core.ValueExpr{}, err
	}
	switch key {
	case "price":
		return core.ValueExpr{Kind: "price", Price: value.Value}, nil
	case "value":
		number, err := strconv.ParseFloat(value.Value, 64)
		if err != nil {
			return core.ValueExpr{}, oops.In("backtest_yaml").With("value", value.Value).Wrapf(err, "parse numeric value expression")
		}
		return core.ValueExpr{Kind: "value", Value: number}, nil
	case "ref":
		return core.ValueExpr{Kind: "ref", Ref: value.Value}, nil
	case "timeframe":
		var raw timeframeValueDocument
		if err := value.Decode(&raw); err != nil {
			return core.ValueExpr{}, oops.In("backtest_yaml").Wrapf(err, "decode timeframe value expression")
		}
		if raw.ID == "" {
			return core.ValueExpr{}, oops.In("backtest_yaml").New("timeframe value expression requires id")
		}
		if emptyYAMLNode(&raw.Value) {
			return core.ValueExpr{}, oops.In("backtest_yaml").New("timeframe value expression requires value")
		}
		child, err := decodeValue(&raw.Value)
		if err != nil {
			return core.ValueExpr{}, err
		}
		child.Timeframe = raw.ID
		return child, nil
	case "position":
		return core.ValueExpr{Kind: "position", Position: value.Value}, nil
	case "portfolio":
		return core.ValueExpr{Kind: "portfolio", Portfolio: value.Value}, nil
	case "indicator":
		var raw indicatorDocument
		if err := value.Decode(&raw); err != nil {
			return core.ValueExpr{}, oops.In("backtest_yaml").Wrapf(err, "decode inline indicator expression")
		}
		source, err := decodeValue(&raw.Source)
		if err != nil {
			return core.ValueExpr{}, err
		}
		indicator := core.IndicatorSpec{
			ID:     raw.ID,
			Source: source,
			Params: raw.Params,
			Output: raw.Output,
		}
		if !emptyYAMLNode(&raw.Compare) {
			compare, err := decodeValue(&raw.Compare)
			if err != nil {
				return core.ValueExpr{}, err
			}
			indicator.Compare = compare
		}
		return core.ValueExpr{Kind: "indicator", Indicator: &indicator}, nil
	case "add", "sub", "mul", "div", "min", "max", "abs", "rank", "percentile", "relative_strength", "spread", "ratio", "universe_rank":
		args, err := decodeValueArgs(key, value)
		if err != nil {
			return core.ValueExpr{}, err
		}
		return core.ValueExpr{Kind: key, Args: args}, nil
	default:
		return core.ValueExpr{}, oops.In("backtest_yaml").With("kind", key).New("unsupported value expression")
	}
}

func decodeValueArgs(operator string, node *yaml.Node) ([]core.ValueExpr, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, oops.In("backtest_yaml").With("operator", operator).New("value expression requires sequence args")
	}
	args := make([]core.ValueExpr, 0, len(node.Content))
	for _, child := range node.Content {
		arg, err := decodeValue(child)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
	}
	return args, nil
}

func documentKind(node *yaml.Node) (string, error) {
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return "", oops.In("backtest_yaml").New("YAML document must be mapping")
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "kind" {
			return node.Content[i+1].Value, nil
		}
	}
	return "", oops.In("backtest_yaml").New("YAML document kind is required")
}

func singleMapping(node *yaml.Node) (string, *yaml.Node, error) {
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return "", nil, oops.In("backtest_yaml").New("expected mapping node")
	}
	if len(node.Content) != 2 {
		return "", nil, oops.In("backtest_yaml").With("fields", len(node.Content)/2).New("mapping node must have one operator")
	}
	return node.Content[0].Value, node.Content[1], nil
}

func emptyDocument(node yaml.Node) bool {
	if node.Kind == 0 {
		return true
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) == 0 {
		return true
	}
	return false
}

func emptyYAMLNode(node *yaml.Node) bool {
	if node == nil || node.Kind == 0 {
		return true
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) == 0 {
		return true
	}
	return node.Kind == yaml.ScalarNode && (node.Tag == "!!null" || node.Value == "")
}
