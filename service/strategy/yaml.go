package strategy

import (
	"context"
	"os"
	"strings"

	universecore "github.com/ev3rlit/mwosa/packages/universe"
	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

const (
	KindScreenStrategy = "ScreenStrategy"
	KindScreenRun      = "ScreenRun"
)

type screenStrategyFile struct {
	Kind          string                  `yaml:"kind"`
	SchemaVersion int                     `yaml:"schema_version"`
	Name          string                  `yaml:"name"`
	Engine        Engine                  `yaml:"engine"`
	Data          ScreenPipelineDataSpec  `yaml:"data"`
	Pipeline      []universecore.StepSpec `yaml:"pipeline"`
	JQ            *JQStrategySpec         `yaml:"jq"`
}

func LoadScreenStrategyFile(ctx context.Context, path string) (ScreenStrategySpec, error) {
	if err := ctx.Err(); err != nil {
		return ScreenStrategySpec{}, oops.In("screen_strategy_yaml").With("path", path).Wrap(err)
	}
	file, err := os.Open(path)
	if err != nil {
		return ScreenStrategySpec{}, oops.In("screen_strategy_yaml").With("path", path).Wrapf(err, "read screen strategy YAML file")
	}
	defer file.Close()

	var raw screenStrategyFile
	if err := yaml.NewDecoder(file).Decode(&raw); err != nil {
		return ScreenStrategySpec{}, oops.In("screen_strategy_yaml").With("path", path).Wrapf(err, "decode screen strategy YAML")
	}
	if raw.SchemaVersion == 0 {
		raw.SchemaVersion = defaultInputSchemaVersion
	}
	switch raw.Kind {
	case KindScreenRun:
		return ScreenStrategySpec{
			Kind:          KindScreenStrategy,
			SchemaVersion: raw.SchemaVersion,
			Name:          raw.Name,
			Engine:        EngineYAMLPipeline,
			Pipeline: &ScreenPipelineStrategySpec{
				Data:     raw.Data,
				Pipeline: raw.Pipeline,
			},
		}, nil
	case KindScreenStrategy:
		engine := raw.Engine
		if engine == "" {
			if raw.JQ != nil {
				engine = EngineJQ
			} else {
				engine = EngineYAMLPipeline
			}
		}
		spec := ScreenStrategySpec{
			Kind:          KindScreenStrategy,
			SchemaVersion: raw.SchemaVersion,
			Name:          raw.Name,
			Engine:        engine,
			JQ:            raw.JQ,
		}
		if engine == EngineYAMLPipeline {
			spec.Pipeline = &ScreenPipelineStrategySpec{
				Data:     raw.Data,
				Pipeline: raw.Pipeline,
			}
		}
		if engine == EngineJQ && spec.JQ != nil {
			spec.JQ.InputDataset = strings.TrimSpace(spec.JQ.InputDataset)
		}
		return spec, nil
	default:
		return ScreenStrategySpec{}, oops.In("screen_strategy_yaml").With("path", path, "kind", raw.Kind).New("screen strategy YAML kind must be ScreenStrategy or ScreenRun")
	}
}
