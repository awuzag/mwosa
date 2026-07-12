package aggregate

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/oops"
)

type OutputRows struct {
	Columns       []OutputColumnSpec `json:"columns"`
	Rows          []map[string]any   `json:"rows"`
	DefaultFormat string             `json:"-"`
}

func FormatOutputRows(spec OutputSpec, rows []json.RawMessage) (OutputRows, error) {
	errb := oops.In("aggregate_output").With("from", spec.From)
	decoded := make([]map[string]any, 0, len(rows))
	for index, row := range rows {
		var object map[string]any
		if err := json.Unmarshal(row, &object); err != nil {
			return OutputRows{}, errb.With("row", index).Wrapf(err, "decode aggregate output row")
		}
		object["ordinal"] = index + 1
		decoded = append(decoded, object)
	}
	columns := spec.OutputColumns()
	if len(columns) == 0 {
		columns = inferColumns(decoded)
	}
	sortOutputRows(decoded, spec.Sort)
	return OutputRows{Columns: columns, Rows: decoded, DefaultFormat: strings.TrimSpace(spec.DefaultFormat)}, nil
}

func (spec OutputSpec) OutputColumns() []OutputColumnSpec {
	return append([]OutputColumnSpec(nil), spec.Columns...)
}

func (o OutputRows) JSONValue() any {
	return o.Rows
}

func (o OutputRows) NDJSONRows() any {
	return o.Rows
}

func (o OutputRows) CSVRows() any {
	return o.Rows
}

func (o OutputRows) TableRows() ([]string, [][]string) {
	header := make([]string, 0, len(o.Columns))
	for _, column := range o.Columns {
		header = append(header, firstNonEmpty(column.Title, column.Key))
	}
	rows := make([][]string, 0, len(o.Rows))
	for _, row := range o.Rows {
		record := make([]string, 0, len(o.Columns))
		for _, column := range o.Columns {
			record = append(record, formatOutputValue(row[column.Key], column))
		}
		rows = append(rows, record)
	}
	return header, rows
}

func (o OutputRows) TableAlignments() []string {
	alignments := make([]string, 0, len(o.Columns))
	for _, column := range o.Columns {
		alignments = append(alignments, column.Align)
	}
	return alignments
}

func inferColumns(rows []map[string]any) []OutputColumnSpec {
	seen := map[string]struct{}{}
	for _, row := range rows {
		for key := range row {
			seen[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	columns := make([]OutputColumnSpec, 0, len(keys))
	for _, key := range keys {
		columns = append(columns, OutputColumnSpec{Key: key})
	}
	return columns
}

func sortOutputRows(rows []map[string]any, sorts []OutputSortSpec) {
	if len(sorts) == 0 {
		return
	}
	sort.SliceStable(rows, func(i int, j int) bool {
		left := rows[i]
		right := rows[j]
		for _, order := range sorts {
			field := strings.TrimSpace(order.Field)
			if field == "" {
				continue
			}
			cmp := compareOutputValues(left[field], right[field])
			if cmp == 0 {
				continue
			}
			if strings.EqualFold(order.Order, "desc") {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
	for index := range rows {
		rows[index]["ordinal"] = index + 1
	}
}

func compareOutputValues(left any, right any) int {
	leftNumber, leftOK := numeric(left)
	rightNumber, rightOK := numeric(right)
	if leftOK && rightOK {
		switch {
		case leftNumber < rightNumber:
			return -1
		case leftNumber > rightNumber:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(toString(left), toString(right))
}

func formatOutputValue(value any, column OutputColumnSpec) string {
	if value == nil {
		return ""
	}
	switch column.Format {
	case "integer":
		if number, ok := numeric(value); ok {
			return strconv.FormatInt(int64(math.Round(number)), 10)
		}
	case "number":
		if number, ok := numeric(value); ok {
			return strconv.FormatFloat(number, 'f', precision(column, -1), 64)
		}
	case "percent":
		if number, ok := numeric(value); ok {
			return strconv.FormatFloat(number, 'f', precision(column, 2), 64) + "%"
		}
	}
	return toString(value)
}

func precision(column OutputColumnSpec, fallback int) int {
	if column.Precision == nil {
		return fallback
	}
	return *column.Precision
}

func numeric(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(typed), ",", ""), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
