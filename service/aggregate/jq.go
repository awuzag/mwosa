package aggregate

import (
	"context"
	"encoding/json"

	"github.com/itchyny/gojq"
	"github.com/samber/oops"
)

func ExecuteJQRows(ctx context.Context, rows []json.RawMessage, queryText string) ([]json.RawMessage, error) {
	errb := oops.In("aggregate_jq")
	query, err := gojq.Parse(queryText)
	if err != nil {
		return nil, errb.Wrapf(err, "execute aggregate jq")
	}
	input := make([]any, 0, len(rows))
	for index, row := range rows {
		var value any
		if err := json.Unmarshal(row, &value); err != nil {
			return nil, errb.With("row", index).Wrapf(err, "decode aggregate jq input row")
		}
		input = append(input, value)
	}
	iter := query.RunWithContext(ctx, input)
	out := make([]json.RawMessage, 0)
	for {
		value, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := value.(error); ok {
			return nil, errb.Wrapf(err, "execute aggregate jq")
		}
		switch typed := value.(type) {
		case []any:
			for index, item := range typed {
				raw, err := json.Marshal(item)
				if err != nil {
					return nil, errb.With("row", index).Wrapf(err, "encode aggregate jq output row")
				}
				out = append(out, raw)
			}
		default:
			raw, err := json.Marshal(typed)
			if err != nil {
				return nil, errb.Wrapf(err, "encode aggregate jq output row")
			}
			out = append(out, raw)
		}
	}
	return out, nil
}
