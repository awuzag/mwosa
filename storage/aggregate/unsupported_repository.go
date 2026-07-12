package aggregate

import (
	"context"
	"time"

	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	"github.com/samber/oops"
)

type unsupportedRepository struct {
	backend string
}

func (r unsupportedRepository) CreateAggregateWithVersion(context.Context, aggregateservice.Aggregate, aggregateservice.Version) (aggregateservice.Detail, error) {
	return aggregateservice.Detail{}, r.err()
}

func (r unsupportedRepository) ListAggregates(context.Context) ([]aggregateservice.Detail, error) {
	return nil, r.err()
}

func (r unsupportedRepository) GetAggregate(context.Context, string) (aggregateservice.Detail, error) {
	return aggregateservice.Detail{}, r.err()
}

func (r unsupportedRepository) GetAggregateVersion(context.Context, string, aggregateservice.VersionRef) (aggregateservice.Detail, error) {
	return aggregateservice.Detail{}, r.err()
}

func (r unsupportedRepository) ListAggregateVersions(context.Context, string) ([]aggregateservice.Version, error) {
	return nil, r.err()
}

func (r unsupportedRepository) AddAggregateVersion(context.Context, string, aggregateservice.Version, time.Time) (aggregateservice.Detail, error) {
	return aggregateservice.Detail{}, r.err()
}

func (r unsupportedRepository) ArchiveAggregate(context.Context, string, time.Time) error {
	return r.err()
}

func (r unsupportedRepository) HasRunAlias(context.Context, string) (bool, error) {
	return false, r.err()
}

func (r unsupportedRepository) CreateRun(context.Context, aggregateservice.Run, []aggregateservice.RunItem) (aggregateservice.RunDetail, error) {
	return aggregateservice.RunDetail{}, r.err()
}

func (r unsupportedRepository) ListRuns(context.Context, aggregateservice.RunHistoryFilter) ([]aggregateservice.Run, error) {
	return nil, r.err()
}

func (r unsupportedRepository) GetRun(context.Context, string, int) (aggregateservice.RunDetail, error) {
	return aggregateservice.RunDetail{}, r.err()
}

func (r unsupportedRepository) err() error {
	return oops.In("aggregate_repository").With("backend", r.backend).New("aggregate repository requires mongodb backend")
}
