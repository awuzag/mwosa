package backtest

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	core "github.com/awuzag/mwosa/packages/backtest"
	backtestservice "github.com/awuzag/mwosa/service/backtest"
	"github.com/awuzag/mwosa/storage"
	storagemongodb "github.com/awuzag/mwosa/storage/mongodb"
	"github.com/samber/oops"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type mongoRepository struct {
	strategies  *mongo.Collection
	runs        *mongo.Collection
	experiments *mongo.Collection
	cases       *mongo.Collection
}

type strategyMongoDocument struct {
	ID              string                         `bson:"_id"`
	SchemaVersion   string                         `bson:"schema_version"`
	Revision        int64                          `bson:"revision"`
	CreatedAt       time.Time                      `bson:"created_at"`
	UpdatedAt       time.Time                      `bson:"updated_at"`
	DeletedAt       *time.Time                     `bson:"deleted_at,omitempty"`
	StrategyID      string                         `bson:"strategy_id"`
	Name            string                         `bson:"name"`
	ActiveVersionID string                         `bson:"active_version_id"`
	Versions        []strategyVersionMongoDocument `bson:"versions"`
}

type strategyVersionMongoDocument struct {
	ID            string    `bson:"id"`
	StrategyID    string    `bson:"strategy_id"`
	Version       int       `bson:"version"`
	SchemaVersion int       `bson:"strategy_schema_version"`
	Spec          any       `bson:"spec"`
	SpecHash      string    `bson:"spec_hash"`
	CreatedAt     time.Time `bson:"created_at"`
}

type backtestRunMongoDocument struct {
	ID                       string    `bson:"_id"`
	SchemaVersion            string    `bson:"schema_version"`
	Revision                 int64     `bson:"revision"`
	CreatedAt                time.Time `bson:"created_at"`
	UpdatedAt                time.Time `bson:"updated_at"`
	RunID                    string    `bson:"run_id"`
	RunName                  string    `bson:"run_name"`
	StrategyName             string    `bson:"strategy_name"`
	Market                   string    `bson:"market"`
	Timeframe                string    `bson:"timeframe"`
	PeriodFrom               string    `bson:"period_from"`
	PeriodTo                 string    `bson:"period_to"`
	StrategyHash             string    `bson:"strategy_hash"`
	RunHash                  string    `bson:"run_hash"`
	EngineVersion            string    `bson:"engine_version"`
	IndicatorRegistryVersion string    `bson:"indicator_registry_version"`
	MetricRegistryVersion    string    `bson:"metric_registry_version"`
	DataFingerprint          string    `bson:"data_fingerprint"`
	ResultHash               string    `bson:"result_hash"`
	Result                   any       `bson:"result"`
	Metrics                  any       `bson:"metrics"`
}

type experimentMongoDocument struct {
	ID                      string                         `bson:"_id"`
	SchemaVersion           string                         `bson:"schema_version"`
	Revision                int64                          `bson:"revision"`
	CreatedAt               time.Time                      `bson:"created_at"`
	UpdatedAt               time.Time                      `bson:"updated_at"`
	ExperimentID            string                         `bson:"experiment_id"`
	Name                    string                         `bson:"name"`
	StrategyName            string                         `bson:"strategy_name"`
	BaseRunName             string                         `bson:"base_run_name"`
	BaseRunKey              string                         `bson:"base_run_key"`
	EvaluationSchemaVersion int                            `bson:"evaluation_schema_version"`
	Spec                    any                            `bson:"spec"`
	SpecHash                string                         `bson:"spec_hash"`
	StrategySpecHash        string                         `bson:"strategy_spec_hash"`
	DataFrom                string                         `bson:"data_from"`
	DataTo                  string                         `bson:"data_to"`
	WalkForwardSteps        []walkForwardStepMongoDocument `bson:"walk_forward_steps"`
}

type experimentCaseMongoDocument struct {
	ID                       string    `bson:"_id"`
	SchemaVersion            string    `bson:"schema_version"`
	Revision                 int64     `bson:"revision"`
	CreatedAt                time.Time `bson:"created_at"`
	UpdatedAt                time.Time `bson:"updated_at"`
	CaseRowID                string    `bson:"case_row_id"`
	ExperimentID             string    `bson:"experiment_id"`
	CaseID                   string    `bson:"case_id"`
	CaseName                 string    `bson:"case_name"`
	RunName                  string    `bson:"run_name"`
	PeriodFrom               string    `bson:"period_from"`
	PeriodTo                 string    `bson:"period_to"`
	Parameters               any       `bson:"parameters"`
	RegimeTags               any       `bson:"regime_tags"`
	Status                   string    `bson:"status"`
	PassedConstraints        bool      `bson:"passed_constraints"`
	Rank                     int       `bson:"rank"`
	Objective                string    `bson:"objective,omitempty"`
	ObjectiveValue           float64   `bson:"objective_value"`
	StrategyHash             string    `bson:"strategy_hash"`
	RunHash                  string    `bson:"run_hash"`
	EngineVersion            string    `bson:"engine_version"`
	IndicatorRegistryVersion string    `bson:"indicator_registry_version"`
	MetricRegistryVersion    string    `bson:"metric_registry_version"`
	DataFingerprint          string    `bson:"data_fingerprint"`
	ResultHash               string    `bson:"result_hash"`
	Result                   any       `bson:"result"`
	Metrics                  any       `bson:"metrics"`
}

type walkForwardStepMongoDocument struct {
	StepRowID                string    `bson:"step_row_id"`
	ExperimentID             string    `bson:"experiment_id"`
	StepIndex                int       `bson:"step_index"`
	TrainFrom                string    `bson:"train_from"`
	TrainTo                  string    `bson:"train_to"`
	TestFrom                 string    `bson:"test_from"`
	TestTo                   string    `bson:"test_to"`
	SelectedParameter        any       `bson:"selected_parameter"`
	TrainCaseID              string    `bson:"train_case_id"`
	TestCaseID               string    `bson:"test_case_id"`
	TrainObjective           float64   `bson:"train_objective"`
	TestMetrics              any       `bson:"test_metrics"`
	StrategyHash             string    `bson:"strategy_hash"`
	RunHash                  string    `bson:"run_hash"`
	EngineVersion            string    `bson:"engine_version"`
	IndicatorRegistryVersion string    `bson:"indicator_registry_version"`
	MetricRegistryVersion    string    `bson:"metric_registry_version"`
	DataFingerprint          string    `bson:"data_fingerprint"`
	ResultHash               string    `bson:"result_hash"`
	CreatedAt                time.Time `bson:"created_at"`
}

var _ backtestservice.StrategyRepository = (*mongoRepository)(nil)
var _ backtestservice.BacktestRunRepository = (*mongoRepository)(nil)
var _ backtestservice.EvaluationRepository = (*mongoRepository)(nil)

func NewMongoRepository(database *mongo.Database) (backtestservice.StrategyRepository, error) {
	if database == nil {
		return nil, oops.In("backtest_repository").New("mongodb database is nil")
	}
	return &mongoRepository{
		strategies:  database.Collection("backtest_strategies"),
		runs:        database.Collection("backtest_runs"),
		experiments: database.Collection("backtest_experiments"),
		cases:       database.Collection("backtest_experiment_cases"),
	}, nil
}

func (r *mongoRepository) CreateStrategyWithVersion(ctx context.Context, in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("backend", "mongodb", "name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	document, err := strategyToMongoDocument(in, version)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	if _, err := r.strategies.InsertOne(ctx, document); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create backtest strategy mongodb document")
	}
	return strategyDetailFromMongoDocument(document)
}

func (r *mongoRepository) ListStrategies(ctx context.Context) ([]backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("backend", "mongodb")
	cursor, err := r.strategies.Find(ctx, bson.D{{Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}}}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, errb.Wrapf(err, "list backtest strategy mongodb documents")
	}
	defer cursor.Close(ctx)

	var documents []strategyMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode backtest strategy mongodb documents")
	}
	details := make([]backtestservice.SavedStrategyDetail, 0, len(documents))
	for _, document := range documents {
		detail, err := strategyDetailFromMongoDocument(document)
		if err != nil {
			return nil, errb.With("strategy_id", document.StrategyID).Wrap(err)
		}
		details = append(details, detail)
	}
	return details, nil
}

func (r *mongoRepository) GetStrategy(ctx context.Context, name string) (backtestservice.SavedStrategyDetail, error) {
	document, err := r.getStrategyDocument(ctx, name)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, err
	}
	return strategyDetailFromMongoDocument(document)
}

func (r *mongoRepository) AddStrategyVersion(ctx context.Context, name string, version backtestservice.SavedStrategyVersion, now time.Time) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("backend", "mongodb", "name", name, "version_id", version.ID)
	current, err := r.getStrategyDocument(ctx, name)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, err
	}
	activeVersion, err := activeMongoVersion(current)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	version.StrategyID = current.StrategyID
	if version.Version == 0 {
		version.Version = activeVersion.Version + 1
	}
	nextVersion, err := strategyVersionToMongoDocument(version)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	current.ActiveVersionID = version.ID
	current.UpdatedAt = now.UTC()
	current.Revision++
	current.Versions = append(current.Versions, nextVersion)
	if err := r.replaceStrategyWithRevision(ctx, current, current.Revision-1); err != nil {
		return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
	}
	return strategyDetailFromMongoDocument(current)
}

func (r *mongoRepository) UpsertStrategyWithVersion(ctx context.Context, in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion, now time.Time) (backtestservice.SavedStrategyDetail, error) {
	errb := oops.In("backtest_strategy_repository").With("backend", "mongodb", "name", in.Name, "strategy_id", in.ID, "version_id", version.ID)
	current, err := r.getStrategyDocument(ctx, in.Name)
	switch {
	case err == nil:
		activeVersion, err := activeMongoVersion(current)
		if err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
		}
		version.StrategyID = current.StrategyID
		version.Version = activeVersion.Version + 1
		nextVersion, err := strategyVersionToMongoDocument(version)
		if err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
		}
		current.ActiveVersionID = version.ID
		current.UpdatedAt = now.UTC()
		current.Revision++
		current.Versions = append(current.Versions, nextVersion)
		if err := r.replaceStrategyWithRevision(ctx, current, current.Revision-1); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
		}
		return strategyDetailFromMongoDocument(current)
	case errors.Is(err, mongo.ErrNoDocuments):
		version.Version = 1
		version.StrategyID = in.ID
		in.ActiveVersionID = version.ID
		document, err := strategyToMongoDocument(in, version)
		if err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrap(err)
		}
		if _, err := r.strategies.InsertOne(ctx, document); err != nil {
			return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "create upserted backtest strategy mongodb document")
		}
		return strategyDetailFromMongoDocument(document)
	default:
		return backtestservice.SavedStrategyDetail{}, errb.Wrapf(err, "load backtest strategy for upsert")
	}
}

func (r *mongoRepository) DeleteStrategy(ctx context.Context, name string, deletedAt time.Time) error {
	errb := oops.In("backtest_strategy_repository").With("backend", "mongodb", "name", name)
	result, err := r.strategies.UpdateOne(ctx,
		bson.D{{Key: "name", Value: name}, {Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "deleted_at", Value: deletedAt.UTC()}, {Key: "updated_at", Value: deletedAt.UTC()}}}, {Key: "$inc", Value: bson.D{{Key: "revision", Value: int64(1)}}}},
	)
	if err != nil {
		return errb.Wrapf(err, "delete backtest strategy mongodb document")
	}
	if result.MatchedCount == 0 {
		return errb.Errorf("backtest strategy not found: %s", name)
	}
	return nil
}

func (r *mongoRepository) SaveRun(ctx context.Context, run backtestservice.SavedBacktestRun, now time.Time) (backtestservice.SavedBacktestRunDetail, error) {
	errb := oops.In("backtest_run_repository").With("backend", "mongodb", "run_id", run.ID, "run", run.RunName)
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	document, err := backtestRunToMongoDocument(run)
	if err != nil {
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrap(err)
	}
	if _, err := r.runs.InsertOne(ctx, document); err != nil {
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrapf(err, "create backtest run mongodb document")
	}
	return backtestRunDetailFromMongoDocument(document)
}

func (r *mongoRepository) ListRuns(ctx context.Context) ([]backtestservice.SavedBacktestRun, error) {
	errb := oops.In("backtest_run_repository").With("backend", "mongodb")
	cursor, err := r.runs.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, errb.Wrapf(err, "list backtest run mongodb documents")
	}
	defer cursor.Close(ctx)

	var documents []backtestRunMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode backtest run mongodb documents")
	}
	out := make([]backtestservice.SavedBacktestRun, 0, len(documents))
	for _, document := range documents {
		run, err := mongoDocumentToBacktestRun(document)
		if err != nil {
			return nil, errb.With("run_id", document.RunID).Wrap(err)
		}
		run.ResultJSON = nil
		run.MetricsJSON = nil
		out = append(out, run)
	}
	return out, nil
}

func (r *mongoRepository) GetRun(ctx context.Context, ref string) (backtestservice.SavedBacktestRunDetail, error) {
	errb := oops.In("backtest_run_repository").With("backend", "mongodb", "ref", ref)
	var document backtestRunMongoDocument
	filter := bson.D{}
	switch {
	case looksLikeID(ref):
		filter = bson.D{{Key: "run_id", Value: ref}}
	case strings.HasPrefix(ref, "sha256:"):
		filter = bson.D{{Key: "result_hash", Value: ref}}
	default:
		filter = bson.D{{Key: "run_name", Value: ref}}
	}
	err := r.runs.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return backtestservice.SavedBacktestRunDetail{}, errb.Errorf("backtest run not found: %s", ref)
	}
	if err != nil {
		return backtestservice.SavedBacktestRunDetail{}, errb.Wrapf(err, "get backtest run mongodb document")
	}
	return backtestRunDetailFromMongoDocument(document)
}

func (r *mongoRepository) SaveEvaluation(ctx context.Context, experiment backtestservice.SavedExperiment, cases []backtestservice.SavedExperimentCase, steps []backtestservice.SavedWalkForwardStep, now time.Time) (backtestservice.SavedEvaluationDetail, error) {
	errb := oops.In("backtest_evaluation_repository").With("backend", "mongodb", "name", experiment.Name, "experiment_id", experiment.ID)
	if experiment.CreatedAt.IsZero() {
		experiment.CreatedAt = now
	}
	experimentDocument, normalizedSteps, err := experimentToMongoDocument(experiment, steps, now)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrap(err)
	}
	if _, err := r.experiments.InsertOne(ctx, experimentDocument); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "create backtest experiment mongodb document")
	}

	normalizedCases := make([]backtestservice.SavedExperimentCase, 0, len(cases))
	caseDocuments := make([]any, 0, len(cases))
	for _, item := range cases {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		document, err := experimentCaseToMongoDocument(item)
		if err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", item.CaseID).Wrap(err)
		}
		normalizedCases = append(normalizedCases, item)
		caseDocuments = append(caseDocuments, document)
	}
	if len(caseDocuments) > 0 {
		if _, err := r.cases.InsertMany(ctx, caseDocuments); err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "create backtest experiment case mongodb documents")
		}
	}
	return backtestservice.SavedEvaluationDetail{
		Experiment:  experiment,
		Cases:       normalizedCases,
		WalkForward: normalizedSteps,
	}, nil
}

func (r *mongoRepository) ListEvaluations(ctx context.Context) ([]backtestservice.SavedEvaluationSummary, error) {
	errb := oops.In("backtest_evaluation_repository").With("backend", "mongodb")
	cursor, err := r.experiments.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, errb.Wrapf(err, "list backtest experiment mongodb documents")
	}
	defer cursor.Close(ctx)

	var documents []experimentMongoDocument
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, errb.Wrapf(err, "decode backtest experiment mongodb documents")
	}
	out := make([]backtestservice.SavedEvaluationSummary, 0, len(documents))
	for _, document := range documents {
		detail, err := r.evaluationDetailByID(ctx, document.ExperimentID)
		if err != nil {
			return nil, errb.With("experiment_id", document.ExperimentID).Wrap(err)
		}
		var best *backtestservice.SavedExperimentCase
		for i := range detail.Cases {
			if detail.Cases[i].Rank == 1 {
				item := detail.Cases[i]
				best = &item
				break
			}
		}
		out = append(out, backtestservice.SavedEvaluationSummary{
			Experiment: detail.Experiment,
			CaseCount:  len(detail.Cases),
			BestCase:   best,
		})
	}
	return out, nil
}

func (r *mongoRepository) GetEvaluation(ctx context.Context, ref string) (backtestservice.SavedEvaluationDetail, error) {
	errb := oops.In("backtest_evaluation_repository").With("backend", "mongodb", "ref", ref)
	var document experimentMongoDocument
	filter := bson.D{}
	if looksLikeID(ref) {
		filter = bson.D{{Key: "experiment_id", Value: ref}}
	} else {
		filter = bson.D{{Key: "name", Value: ref}}
	}
	err := r.experiments.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return backtestservice.SavedEvaluationDetail{}, errb.Errorf("backtest evaluation not found: %s", ref)
	}
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "get backtest experiment mongodb document")
	}
	return r.evaluationDetailByID(ctx, document.ExperimentID)
}

func (r *mongoRepository) evaluationDetailByID(ctx context.Context, id string) (backtestservice.SavedEvaluationDetail, error) {
	errb := oops.In("backtest_evaluation_repository").With("backend", "mongodb", "experiment_id", id)
	var experimentDocument experimentMongoDocument
	if err := r.experiments.FindOne(ctx, bson.D{{Key: "experiment_id", Value: id}}).Decode(&experimentDocument); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "get backtest experiment mongodb document")
	}

	cursor, err := r.cases.Find(ctx, bson.D{{Key: "experiment_id", Value: id}}, options.Find().SetSort(bson.D{{Key: "rank", Value: 1}, {Key: "case_id", Value: 1}}))
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "list backtest experiment case mongodb documents")
	}
	defer cursor.Close(ctx)
	var caseDocuments []experimentCaseMongoDocument
	if err := cursor.All(ctx, &caseDocuments); err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrapf(err, "decode backtest experiment case mongodb documents")
	}

	experiment, err := mongoDocumentToExperiment(experimentDocument)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrap(err)
	}
	cases := make([]backtestservice.SavedExperimentCase, 0, len(caseDocuments))
	for _, document := range caseDocuments {
		item, err := mongoDocumentToExperimentCase(document)
		if err != nil {
			return backtestservice.SavedEvaluationDetail{}, errb.With("case_id", document.CaseID).Wrap(err)
		}
		cases = append(cases, item)
	}
	steps, err := mongoDocumentsToWalkForwardSteps(experimentDocument.WalkForwardSteps)
	if err != nil {
		return backtestservice.SavedEvaluationDetail{}, errb.Wrap(err)
	}
	return backtestservice.SavedEvaluationDetail{
		Experiment:  experiment,
		Cases:       cases,
		WalkForward: steps,
	}, nil
}

func (r *mongoRepository) getStrategyDocument(ctx context.Context, name string) (strategyMongoDocument, error) {
	var document strategyMongoDocument
	err := r.strategies.FindOne(ctx, bson.D{{Key: "name", Value: name}, {Key: "deleted_at", Value: bson.D{{Key: "$exists", Value: false}}}}).Decode(&document)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return strategyMongoDocument{}, mongo.ErrNoDocuments
	}
	if err != nil {
		return strategyMongoDocument{}, oops.In("backtest_strategy_repository").With("backend", "mongodb", "name", name).Wrapf(err, "get backtest strategy mongodb document")
	}
	return document, nil
}

func (r *mongoRepository) replaceStrategyWithRevision(ctx context.Context, document strategyMongoDocument, revision int64) error {
	result, err := r.strategies.ReplaceOne(ctx, bson.D{{Key: "_id", Value: document.ID}, {Key: "revision", Value: revision}}, document)
	if err != nil {
		return oops.In("backtest_strategy_repository").With("backend", "mongodb", "strategy_id", document.StrategyID, "revision", revision).Wrapf(err, "replace backtest strategy mongodb document")
	}
	if result.MatchedCount == 0 {
		return storagemongodb.NewRevisionConflictError("backtest_strategies", document.ID, revision)
	}
	return nil
}

func strategyToMongoDocument(in backtestservice.SavedStrategy, version backtestservice.SavedStrategyVersion) (strategyMongoDocument, error) {
	versionDocument, err := strategyVersionToMongoDocument(version)
	if err != nil {
		return strategyMongoDocument{}, err
	}
	return strategyMongoDocument{
		ID:              backtestStrategyMongoID(in.ID),
		SchemaVersion:   storagemongodb.SchemaVersion1,
		Revision:        1,
		CreatedAt:       in.CreatedAt.UTC(),
		UpdatedAt:       in.UpdatedAt.UTC(),
		DeletedAt:       in.DeletedAt,
		StrategyID:      in.ID,
		Name:            in.Name,
		ActiveVersionID: in.ActiveVersionID,
		Versions:        []strategyVersionMongoDocument{versionDocument},
	}, nil
}

func strategyVersionToMongoDocument(version backtestservice.SavedStrategyVersion) (strategyVersionMongoDocument, error) {
	spec, err := rawJSONToBSONValue(version.SpecJSON)
	if err != nil {
		return strategyVersionMongoDocument{}, err
	}
	return strategyVersionMongoDocument{
		ID:            version.ID,
		StrategyID:    version.StrategyID,
		Version:       version.Version,
		SchemaVersion: version.SchemaVersion,
		Spec:          spec,
		SpecHash:      version.SpecHash,
		CreatedAt:     version.CreatedAt.UTC(),
	}, nil
}

func strategyDetailFromMongoDocument(document strategyMongoDocument) (backtestservice.SavedStrategyDetail, error) {
	version, err := activeMongoVersion(document)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, err
	}
	strategyRow := storage.BacktestStrategyRow{
		ID:              document.StrategyID,
		Name:            document.Name,
		ActiveVersionID: document.ActiveVersionID,
		CreatedAt:       document.CreatedAt,
		UpdatedAt:       document.UpdatedAt,
		DeletedAt:       document.DeletedAt,
	}
	versionRow, err := mongoVersionToRow(version)
	if err != nil {
		return backtestservice.SavedStrategyDetail{}, err
	}
	return detailFromRows(&strategyRow, &versionRow)
}

func activeMongoVersion(document strategyMongoDocument) (strategyVersionMongoDocument, error) {
	for _, version := range document.Versions {
		if version.ID == document.ActiveVersionID {
			return version, nil
		}
	}
	return strategyVersionMongoDocument{}, oops.In("backtest_strategy_repository").With("backend", "mongodb", "strategy_id", document.StrategyID, "version_id", document.ActiveVersionID).New("backtest strategy version not found")
}

func mongoVersionToRow(version strategyVersionMongoDocument) (storage.BacktestStrategyVersionRow, error) {
	specJSON, err := bsonValueToRawJSON(version.Spec)
	if err != nil {
		return storage.BacktestStrategyVersionRow{}, err
	}
	return storage.BacktestStrategyVersionRow{
		ID:            version.ID,
		StrategyID:    version.StrategyID,
		Version:       version.Version,
		SchemaVersion: version.SchemaVersion,
		SpecJSON:      string(specJSON),
		SpecHash:      version.SpecHash,
		CreatedAt:     version.CreatedAt,
	}, nil
}

func backtestRunToMongoDocument(run backtestservice.SavedBacktestRun) (backtestRunMongoDocument, error) {
	result, err := rawJSONToBSONValue(run.ResultJSON)
	if err != nil {
		return backtestRunMongoDocument{}, err
	}
	metrics, err := rawJSONToBSONValue(run.MetricsJSON)
	if err != nil {
		return backtestRunMongoDocument{}, err
	}
	createdAt := run.CreatedAt.UTC()
	return backtestRunMongoDocument{
		ID:                       backtestRunMongoID(run.ID),
		SchemaVersion:            storagemongodb.SchemaVersion1,
		Revision:                 1,
		CreatedAt:                createdAt,
		UpdatedAt:                createdAt,
		RunID:                    run.ID,
		RunName:                  run.RunName,
		StrategyName:             run.StrategyName,
		Market:                   run.Market,
		Timeframe:                run.Timeframe,
		PeriodFrom:               run.PeriodFrom,
		PeriodTo:                 run.PeriodTo,
		StrategyHash:             run.StrategyHash,
		RunHash:                  run.RunHash,
		EngineVersion:            run.EngineVersion,
		IndicatorRegistryVersion: run.IndicatorRegistry,
		MetricRegistryVersion:    run.MetricRegistry,
		DataFingerprint:          run.DataFingerprint,
		ResultHash:               run.ResultHash,
		Result:                   result,
		Metrics:                  metrics,
	}, nil
}

func backtestRunDetailFromMongoDocument(document backtestRunMongoDocument) (backtestservice.SavedBacktestRunDetail, error) {
	run, err := mongoDocumentToBacktestRun(document)
	if err != nil {
		return backtestservice.SavedBacktestRunDetail{}, err
	}
	var result core.Result
	if err := json.Unmarshal(run.ResultJSON, &result); err != nil {
		return backtestservice.SavedBacktestRunDetail{}, oops.In("backtest_run_repository").With("backend", "mongodb", "run_id", document.RunID).Wrapf(err, "decode backtest result json")
	}
	return backtestservice.SavedBacktestRunDetail{Run: run, Result: result}, nil
}

func mongoDocumentToBacktestRun(document backtestRunMongoDocument) (backtestservice.SavedBacktestRun, error) {
	resultJSON, err := bsonValueToRawJSON(document.Result)
	if err != nil {
		return backtestservice.SavedBacktestRun{}, err
	}
	metricsJSON, err := bsonValueToRawJSON(document.Metrics)
	if err != nil {
		return backtestservice.SavedBacktestRun{}, err
	}
	return backtestservice.SavedBacktestRun{
		ID:                document.RunID,
		RunName:           document.RunName,
		StrategyName:      document.StrategyName,
		Market:            document.Market,
		Timeframe:         document.Timeframe,
		PeriodFrom:        document.PeriodFrom,
		PeriodTo:          document.PeriodTo,
		StrategyHash:      document.StrategyHash,
		RunHash:           document.RunHash,
		EngineVersion:     document.EngineVersion,
		IndicatorRegistry: document.IndicatorRegistryVersion,
		MetricRegistry:    document.MetricRegistryVersion,
		DataFingerprint:   document.DataFingerprint,
		ResultHash:        document.ResultHash,
		ResultJSON:        resultJSON,
		MetricsJSON:       metricsJSON,
		CreatedAt:         document.CreatedAt,
	}, nil
}

func experimentToMongoDocument(experiment backtestservice.SavedExperiment, steps []backtestservice.SavedWalkForwardStep, now time.Time) (experimentMongoDocument, []backtestservice.SavedWalkForwardStep, error) {
	spec, err := rawJSONToBSONValue(experiment.SpecJSON)
	if err != nil {
		return experimentMongoDocument{}, nil, err
	}
	normalizedSteps := make([]backtestservice.SavedWalkForwardStep, 0, len(steps))
	stepDocuments := make([]walkForwardStepMongoDocument, 0, len(steps))
	for _, item := range steps {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = now
		}
		document, err := walkForwardStepToMongoDocument(item)
		if err != nil {
			return experimentMongoDocument{}, nil, err
		}
		normalizedSteps = append(normalizedSteps, item)
		stepDocuments = append(stepDocuments, document)
	}
	sort.SliceStable(stepDocuments, func(i, j int) bool {
		return stepDocuments[i].StepIndex < stepDocuments[j].StepIndex
	})
	createdAt := experiment.CreatedAt.UTC()
	return experimentMongoDocument{
		ID:                      backtestExperimentMongoID(experiment.ID),
		SchemaVersion:           storagemongodb.SchemaVersion1,
		Revision:                1,
		CreatedAt:               createdAt,
		UpdatedAt:               createdAt,
		ExperimentID:            experiment.ID,
		Name:                    experiment.Name,
		StrategyName:            experiment.StrategyName,
		BaseRunName:             experiment.BaseRunName,
		BaseRunKey:              experiment.BaseRunName,
		EvaluationSchemaVersion: experiment.SchemaVersion,
		Spec:                    spec,
		SpecHash:                experiment.SpecHash,
		StrategySpecHash:        experiment.StrategySpecHash,
		DataFrom:                experiment.DataFrom,
		DataTo:                  experiment.DataTo,
		WalkForwardSteps:        stepDocuments,
	}, normalizedSteps, nil
}

func experimentCaseToMongoDocument(item backtestservice.SavedExperimentCase) (experimentCaseMongoDocument, error) {
	parameters, err := rawJSONToBSONValue(item.ParameterJSON)
	if err != nil {
		return experimentCaseMongoDocument{}, err
	}
	regimeTags, err := rawJSONToBSONValue(item.RegimeTagsJSON)
	if err != nil {
		return experimentCaseMongoDocument{}, err
	}
	result, err := rawJSONToBSONValue(item.ResultJSON)
	if err != nil {
		return experimentCaseMongoDocument{}, err
	}
	metrics, err := rawJSONToBSONValue(item.MetricsJSON)
	if err != nil {
		return experimentCaseMongoDocument{}, err
	}
	createdAt := item.CreatedAt.UTC()
	return experimentCaseMongoDocument{
		ID:                       backtestExperimentCaseMongoID(item.ExperimentID, item.CaseID),
		SchemaVersion:            storagemongodb.SchemaVersion1,
		Revision:                 1,
		CreatedAt:                createdAt,
		UpdatedAt:                createdAt,
		CaseRowID:                item.ID,
		ExperimentID:             item.ExperimentID,
		CaseID:                   item.CaseID,
		CaseName:                 item.CaseName,
		RunName:                  item.RunName,
		PeriodFrom:               item.PeriodFrom,
		PeriodTo:                 item.PeriodTo,
		Parameters:               parameters,
		RegimeTags:               regimeTags,
		Status:                   item.Status,
		PassedConstraints:        item.PassedConstraints,
		Rank:                     item.Rank,
		Objective:                item.Objective,
		ObjectiveValue:           item.ObjectiveValue,
		StrategyHash:             item.StrategyHash,
		RunHash:                  item.RunHash,
		EngineVersion:            item.EngineVersion,
		IndicatorRegistryVersion: item.IndicatorRegistry,
		MetricRegistryVersion:    item.MetricRegistry,
		DataFingerprint:          item.DataFingerprint,
		ResultHash:               item.ResultHash,
		Result:                   result,
		Metrics:                  metrics,
	}, nil
}

func walkForwardStepToMongoDocument(item backtestservice.SavedWalkForwardStep) (walkForwardStepMongoDocument, error) {
	selectedParameter, err := rawJSONToBSONValue(item.SelectedParameterJSON)
	if err != nil {
		return walkForwardStepMongoDocument{}, err
	}
	testMetrics, err := rawJSONToBSONValue(item.TestMetricsJSON)
	if err != nil {
		return walkForwardStepMongoDocument{}, err
	}
	return walkForwardStepMongoDocument{
		StepRowID:                item.ID,
		ExperimentID:             item.ExperimentID,
		StepIndex:                item.StepIndex,
		TrainFrom:                item.TrainFrom,
		TrainTo:                  item.TrainTo,
		TestFrom:                 item.TestFrom,
		TestTo:                   item.TestTo,
		SelectedParameter:        selectedParameter,
		TrainCaseID:              item.TrainCaseID,
		TestCaseID:               item.TestCaseID,
		TrainObjective:           item.TrainObjective,
		TestMetrics:              testMetrics,
		StrategyHash:             item.StrategyHash,
		RunHash:                  item.RunHash,
		EngineVersion:            item.EngineVersion,
		IndicatorRegistryVersion: item.IndicatorRegistry,
		MetricRegistryVersion:    item.MetricRegistry,
		DataFingerprint:          item.DataFingerprint,
		ResultHash:               item.ResultHash,
		CreatedAt:                item.CreatedAt.UTC(),
	}, nil
}

func mongoDocumentToExperiment(document experimentMongoDocument) (backtestservice.SavedExperiment, error) {
	specJSON, err := bsonValueToRawJSON(document.Spec)
	if err != nil {
		return backtestservice.SavedExperiment{}, err
	}
	return backtestservice.SavedExperiment{
		ID:               document.ExperimentID,
		Name:             document.Name,
		StrategyName:     document.StrategyName,
		BaseRunName:      document.BaseRunName,
		SchemaVersion:    document.EvaluationSchemaVersion,
		SpecJSON:         specJSON,
		SpecHash:         document.SpecHash,
		StrategySpecHash: document.StrategySpecHash,
		DataFrom:         document.DataFrom,
		DataTo:           document.DataTo,
		CreatedAt:        document.CreatedAt,
	}, nil
}

func mongoDocumentToExperimentCase(document experimentCaseMongoDocument) (backtestservice.SavedExperimentCase, error) {
	parameterJSON, err := bsonValueToRawJSON(document.Parameters)
	if err != nil {
		return backtestservice.SavedExperimentCase{}, err
	}
	regimeTagsJSON, err := bsonValueToRawJSON(document.RegimeTags)
	if err != nil {
		return backtestservice.SavedExperimentCase{}, err
	}
	resultJSON, err := bsonValueToRawJSON(document.Result)
	if err != nil {
		return backtestservice.SavedExperimentCase{}, err
	}
	metricsJSON, err := bsonValueToRawJSON(document.Metrics)
	if err != nil {
		return backtestservice.SavedExperimentCase{}, err
	}
	return backtestservice.SavedExperimentCase{
		ID:                document.CaseRowID,
		ExperimentID:      document.ExperimentID,
		CaseID:            document.CaseID,
		CaseName:          document.CaseName,
		RunName:           document.RunName,
		PeriodFrom:        document.PeriodFrom,
		PeriodTo:          document.PeriodTo,
		ParameterJSON:     parameterJSON,
		RegimeTagsJSON:    regimeTagsJSON,
		Status:            document.Status,
		PassedConstraints: document.PassedConstraints,
		Rank:              document.Rank,
		Objective:         document.Objective,
		ObjectiveValue:    document.ObjectiveValue,
		ResultJSON:        resultJSON,
		MetricsJSON:       metricsJSON,
		StrategyHash:      document.StrategyHash,
		RunHash:           document.RunHash,
		EngineVersion:     document.EngineVersion,
		IndicatorRegistry: document.IndicatorRegistryVersion,
		MetricRegistry:    document.MetricRegistryVersion,
		DataFingerprint:   document.DataFingerprint,
		ResultHash:        document.ResultHash,
		CreatedAt:         document.CreatedAt,
	}, nil
}

func mongoDocumentsToWalkForwardSteps(documents []walkForwardStepMongoDocument) ([]backtestservice.SavedWalkForwardStep, error) {
	sort.SliceStable(documents, func(i, j int) bool {
		return documents[i].StepIndex < documents[j].StepIndex
	})
	steps := make([]backtestservice.SavedWalkForwardStep, 0, len(documents))
	for _, document := range documents {
		selectedParameterJSON, err := bsonValueToRawJSON(document.SelectedParameter)
		if err != nil {
			return nil, err
		}
		testMetricsJSON, err := bsonValueToRawJSON(document.TestMetrics)
		if err != nil {
			return nil, err
		}
		steps = append(steps, backtestservice.SavedWalkForwardStep{
			ID:                    document.StepRowID,
			ExperimentID:          document.ExperimentID,
			StepIndex:             document.StepIndex,
			TrainFrom:             document.TrainFrom,
			TrainTo:               document.TrainTo,
			TestFrom:              document.TestFrom,
			TestTo:                document.TestTo,
			SelectedParameterJSON: selectedParameterJSON,
			TrainCaseID:           document.TrainCaseID,
			TestCaseID:            document.TestCaseID,
			TrainObjective:        document.TrainObjective,
			TestMetricsJSON:       testMetricsJSON,
			StrategyHash:          document.StrategyHash,
			RunHash:               document.RunHash,
			EngineVersion:         document.EngineVersion,
			IndicatorRegistry:     document.IndicatorRegistryVersion,
			MetricRegistry:        document.MetricRegistryVersion,
			DataFingerprint:       document.DataFingerprint,
			ResultHash:            document.ResultHash,
			CreatedAt:             document.CreatedAt,
		})
	}
	return steps, nil
}

func rawJSONToBSONValue(raw json.RawMessage) (any, error) {
	normalized := normalizeRawMessage(raw)
	var value any
	if err := json.Unmarshal(normalized, &value); err != nil {
		return nil, oops.In("backtest_repository").Wrapf(err, "decode json payload")
	}
	return value, nil
}

func bsonValueToRawJSON(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage(`{}`), nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, oops.In("backtest_repository").Wrapf(err, "encode bson payload")
	}
	return json.RawMessage(bytes), nil
}

func normalizeRawMessage(raw json.RawMessage) json.RawMessage {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(trimmed)
}

func backtestStrategyMongoID(strategyID string) string {
	return strings.Join([]string{"backtest_strategies", strategyID}, ":")
}

func backtestRunMongoID(runID string) string {
	return strings.Join([]string{"backtest_runs", runID}, ":")
}

func backtestExperimentMongoID(experimentID string) string {
	return strings.Join([]string{"backtest_experiments", experimentID}, ":")
}

func backtestExperimentCaseMongoID(experimentID string, caseID string) string {
	return strings.Join([]string{"backtest_experiment_cases", experimentID, caseID}, ":")
}
