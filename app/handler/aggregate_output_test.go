package handler

import (
	"testing"

	aggregateservice "github.com/awuzag/mwosa/service/aggregate"
	"github.com/stretchr/testify/assert"
)

func TestAggregateRunOutputKeepsEmptyResultShape(t *testing.T) {
	output := AggregateRunOutput{
		Rows: aggregateservice.OutputRows{
			Columns: []aggregateservice.OutputColumnSpec{{Key: "symbol", Title: "symbol"}},
			Rows:    []map[string]any{},
		},
	}

	assert.Equal(t, []map[string]any{}, output.JSONValue())
	header, rows := output.TableRows()
	assert.Equal(t, []string{"symbol"}, header)
	assert.Empty(t, rows)
}
