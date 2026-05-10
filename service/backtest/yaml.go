package backtest

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"

	core "github.com/ev3rlit/mwosa/packages/backtest"
	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

type Bundle struct {
	Strategy core.StrategySpec
	Run      core.BacktestRunSpec
}

func LoadFile(ctx context.Context, path string) (Bundle, error) {
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
	return Decode(ctx, bytes.NewReader(payload))
}

func Decode(ctx context.Context, reader io.Reader) (Bundle, error) {
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
		default:
			return Bundle{}, oops.In("backtest_yaml").With("kind", kind).New("unsupported YAML document kind")
		}
	}
	if bundle.Strategy.Kind == "" {
		return Bundle{}, oops.In("backtest_yaml").New("YAML stream requires Strategy document")
	}
	if bundle.Run.Kind == "" {
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
	Sizing        core.SizingSpec              `yaml:"sizing"`
	Risk          core.RiskSpec                `yaml:"risk"`
}

type indicatorDocument struct {
	ID     string             `yaml:"id"`
	Source yaml.Node          `yaml:"source"`
	Params map[string]float64 `yaml:"params"`
	Output string             `yaml:"output"`
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
		indicators[alias] = core.IndicatorSpec{
			ID:     raw.ID,
			Source: source,
			Params: raw.Params,
			Output: raw.Output,
		}
	}
	entry, err := decodeRule(&document.Entry)
	if err != nil {
		return core.StrategySpec{}, oops.In("backtest_yaml").Wrapf(err, "decode entry rule")
	}
	exit, err := decodeRule(&document.Exit)
	if err != nil {
		return core.StrategySpec{}, oops.In("backtest_yaml").Wrapf(err, "decode exit rule")
	}
	return core.StrategySpec{
		Kind:          document.Kind,
		SchemaVersion: document.SchemaVersion,
		Name:          document.Name,
		Description:   document.Description,
		Tags:          document.Tags,
		Indicators:    indicators,
		Entry:         entry,
		Exit:          exit,
		Sizing:        document.Sizing,
		Risk:          document.Risk,
	}, nil
}

func decodeRun(node *yaml.Node) (core.BacktestRunSpec, error) {
	var run core.BacktestRunSpec
	if err := node.Decode(&run); err != nil {
		return core.BacktestRunSpec{}, oops.In("backtest_yaml").Wrapf(err, "decode BacktestRun document")
	}
	return run, nil
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
		return core.ValueExpr{Kind: "indicator", Indicator: &indicator}, nil
	default:
		return core.ValueExpr{}, oops.In("backtest_yaml").With("kind", key).New("unsupported value expression")
	}
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
