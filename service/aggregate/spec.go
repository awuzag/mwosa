package aggregate

import (
	"encoding/json"
	"regexp"
	"slices"
	"strings"

	"github.com/samber/oops"
)

const (
	KindAggregate = "Aggregate"
)

type ParamType string

const (
	ParamString ParamType = "string"
	ParamDate   ParamType = "date"
	ParamInt    ParamType = "int"
	ParamFloat  ParamType = "float"
	ParamBool   ParamType = "bool"
)

type StageType string

const (
	StageProvider        StageType = "provider"
	StageProviderRaw     StageType = "provider_raw"
	StageLocalCollection StageType = "local_collection"
	StageLocalDataset    StageType = "local_dataset"
	StageSnapshot        StageType = "snapshot"
	StageAggregateRun    StageType = "aggregate_run"
	StageAggregate       StageType = "aggregate"
	StageJQ              StageType = "jq"
)

type Spec struct {
	Kind          string               `json:"kind" yaml:"kind"`
	SchemaVersion int                  `json:"schema_version" yaml:"schema_version"`
	Name          string               `json:"name" yaml:"name"`
	Description   string               `json:"description,omitempty" yaml:"description,omitempty"`
	Params        map[string]ParamSpec `json:"params,omitempty" yaml:"params,omitempty"`
	Workspace     WorkspaceSpec        `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	Pipeline      []StageSpec          `json:"pipeline" yaml:"pipeline"`
	Output        OutputSpec           `json:"output" yaml:"output"`
}

type ParamSpec struct {
	Type        ParamType `json:"type" yaml:"type"`
	Default     any       `json:"default,omitempty" yaml:"default,omitempty"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool      `json:"required,omitempty" yaml:"required,omitempty"`
}

type WorkspaceSpec struct {
	TTL string `json:"ttl,omitempty" yaml:"ttl,omitempty"`
}

type ForeachSpec struct {
	Stage string `json:"stage,omitempty" yaml:"stage,omitempty"`
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	As    string `json:"as,omitempty" yaml:"as,omitempty"`
}

type StageSpec struct {
	Name       string           `json:"name" yaml:"name"`
	Type       StageType        `json:"type" yaml:"type"`
	From       string           `json:"from,omitempty" yaml:"from,omitempty"`
	Provider   string           `json:"provider,omitempty" yaml:"provider,omitempty"`
	Role       string           `json:"role,omitempty" yaml:"role,omitempty"`
	Operation  string           `json:"operation,omitempty" yaml:"operation,omitempty"`
	Collection string           `json:"collection,omitempty" yaml:"collection,omitempty"`
	Dataset    string           `json:"dataset,omitempty" yaml:"dataset,omitempty"`
	Run        string           `json:"run,omitempty" yaml:"run,omitempty"`
	Aggregate  string           `json:"aggregate,omitempty" yaml:"aggregate,omitempty"`
	Filter     map[string]any   `json:"filter,omitempty" yaml:"filter,omitempty"`
	Request    map[string]any   `json:"request,omitempty" yaml:"request,omitempty"`
	Params     map[string]any   `json:"params,omitempty" yaml:"params,omitempty"`
	Pipeline   []map[string]any `json:"pipeline,omitempty" yaml:"pipeline,omitempty"`
	Query      string           `json:"query,omitempty" yaml:"query,omitempty"`
	Foreach    *ForeachSpec     `json:"foreach,omitempty" yaml:"foreach,omitempty"`
	Raw        json.RawMessage  `json:"raw,omitempty" yaml:"-"`
}

type OutputSpec struct {
	From          string             `json:"from" yaml:"from"`
	DefaultFormat string             `json:"default_format,omitempty" yaml:"default_format,omitempty"`
	Sort          []OutputSortSpec   `json:"sort,omitempty" yaml:"sort,omitempty"`
	Columns       []OutputColumnSpec `json:"columns,omitempty" yaml:"columns,omitempty"`
}

type OutputSortSpec struct {
	Field string `json:"field" yaml:"field"`
	Order string `json:"order,omitempty" yaml:"order,omitempty"`
}

type OutputColumnSpec struct {
	Key       string `json:"key" yaml:"key"`
	Title     string `json:"title,omitempty" yaml:"title,omitempty"`
	Format    string `json:"format,omitempty" yaml:"format,omitempty"`
	Precision *int   `json:"precision,omitempty" yaml:"precision,omitempty"`
	Align     string `json:"align,omitempty" yaml:"align,omitempty"`
}

var allowedStageTypes = []StageType{
	StageProvider,
	StageProviderRaw,
	StageLocalCollection,
	StageLocalDataset,
	StageSnapshot,
	StageAggregateRun,
	StageAggregate,
	StageJQ,
}

func ValidateSpec(spec Spec) error {
	errb := oops.In("aggregate_spec").With("name", spec.Name)
	if strings.TrimSpace(spec.Kind) != KindAggregate {
		return errb.With("kind", spec.Kind).New("aggregate spec kind must be Aggregate")
	}
	if spec.SchemaVersion != 1 {
		return errb.With("schema_version", spec.SchemaVersion).New("unsupported aggregate schema version")
	}
	if strings.TrimSpace(spec.Name) == "" {
		return errb.New("aggregate name is required")
	}
	if len(spec.Pipeline) == 0 {
		return errb.New("aggregate pipeline requires at least one stage")
	}
	seen := map[string]struct{}{}
	for i, stage := range spec.Pipeline {
		stageErr := oops.In("aggregate_spec").With("name", spec.Name, "stage_index", i, "stage", stage.Name, "type", stage.Type)
		if strings.TrimSpace(stage.Name) == "" {
			return stageErr.New("aggregate stage name is required")
		}
		if _, exists := seen[stage.Name]; exists {
			return stageErr.New("duplicate aggregate stage name")
		}
		if !slices.Contains(allowedStageTypes, stage.Type) {
			return stageErr.Errorf("unsupported aggregate stage type: %s", stage.Type)
		}
		if stage.Foreach != nil {
			if _, ok := seen[stage.Foreach.Stage]; !ok {
				return stageErr.With("foreach_stage", stage.Foreach.Stage).New("unknown aggregate foreach stage")
			}
		}
		if requiresInput(stage.Type) {
			if strings.TrimSpace(stage.From) == "" {
				return stageErr.New("aggregate stage input is required")
			}
			if _, ok := seen[stage.From]; !ok {
				return stageErr.With("from", stage.From).New("unknown aggregate stage input")
			}
		}
		if err := validateStageShape(stage); err != nil {
			return stageErr.Wrap(err)
		}
		if stage.Type == StageAggregate {
			if err := ValidateMongoPipeline(stage.Pipeline, seenAliases(seen)); err != nil {
				return stageErr.Wrap(err)
			}
		}
		seen[stage.Name] = struct{}{}
	}
	if strings.TrimSpace(spec.Output.From) == "" {
		return errb.New("aggregate output.from is required")
	}
	if _, ok := seen[spec.Output.From]; !ok {
		return errb.With("output_from", spec.Output.From).New("unknown aggregate output stage")
	}
	for i, column := range spec.Output.Columns {
		if strings.TrimSpace(column.Key) == "" {
			return errb.With("column_index", i).New("aggregate output column key is required")
		}
	}
	return nil
}

func requiresInput(stageType StageType) bool {
	return stageType == StageAggregate || stageType == StageJQ
}

func validateStageShape(stage StageSpec) error {
	errb := oops.In("aggregate_stage").With("stage", stage.Name, "type", stage.Type)
	switch stage.Type {
	case StageProvider:
		if strings.TrimSpace(stage.Provider) == "" || strings.TrimSpace(stage.Role) == "" {
			return errb.New("provider stage requires provider and role")
		}
	case StageProviderRaw:
		if strings.TrimSpace(stage.Provider) == "" || strings.TrimSpace(stage.Operation) == "" {
			return errb.New("provider_raw stage requires provider and operation")
		}
	case StageLocalCollection:
		if strings.TrimSpace(stage.Collection) == "" {
			return errb.New("local_collection stage requires collection")
		}
	case StageLocalDataset:
		if strings.TrimSpace(stage.Dataset) == "" {
			return errb.New("local_dataset stage requires dataset")
		}
	case StageAggregateRun:
		if strings.TrimSpace(stage.Run) == "" && strings.TrimSpace(stage.Aggregate) == "" {
			return errb.New("aggregate_run stage requires run or aggregate")
		}
	case StageAggregate:
		if len(stage.Pipeline) == 0 {
			return errb.New("aggregate stage requires mongodb pipeline")
		}
	case StageJQ:
		if strings.TrimSpace(stage.Query) == "" {
			return errb.New("jq stage requires query")
		}
	}
	return nil
}

func seenAliases(seen map[string]struct{}) map[string]string {
	out := make(map[string]string, len(seen))
	for alias := range seen {
		out[alias] = alias
	}
	return out
}

var placeholderPattern = regexp.MustCompile(`\$\{(params|each)\.([A-Za-z0-9_.-]+)\}`)
