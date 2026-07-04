package aggregate

import (
	"context"
	"os"

	"github.com/samber/oops"
	"gopkg.in/yaml.v3"
)

func LoadSpecFile(ctx context.Context, path string) (Spec, error) {
	if err := ctx.Err(); err != nil {
		return Spec{}, oops.In("aggregate_yaml").With("path", path).Wrap(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, oops.In("aggregate_yaml").With("path", path).Wrapf(err, "read aggregate YAML file")
	}
	return LoadSpecBytes(ctx, data)
}

func LoadSpecBytes(ctx context.Context, data []byte) (Spec, error) {
	if err := ctx.Err(); err != nil {
		return Spec{}, oops.In("aggregate_yaml").Wrap(err)
	}
	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return Spec{}, oops.In("aggregate_yaml").Wrapf(err, "decode aggregate YAML")
	}
	if spec.SchemaVersion == 0 {
		spec.SchemaVersion = 1
	}
	return spec, nil
}
