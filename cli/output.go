package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/jszwec/csvutil"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type OutputMode string

const (
	DefaultOutputMode OutputMode = OutputModeTable

	OutputModeTable  OutputMode = "table"
	OutputModeJSON   OutputMode = "json"
	OutputModeNDJSON OutputMode = "ndjson"
	OutputModeCSV    OutputMode = "csv"
)

var supportedOutputModes = []OutputMode{
	OutputModeTable,
	OutputModeJSON,
	OutputModeNDJSON,
	OutputModeCSV,
}

func SupportedOutputModeStrings() []string {
	values := make([]string, 0, len(supportedOutputModes))
	for _, mode := range supportedOutputModes {
		values = append(values, string(mode))
	}
	return values
}

func OutputModeHelp() string {
	return "output format: " + strings.Join(SupportedOutputModeStrings(), ", ")
}

func ParseOutputMode(value string) (OutputMode, error) {
	if value == "" {
		return DefaultOutputMode, nil
	}
	for _, mode := range supportedOutputModes {
		if value == string(mode) {
			return mode, nil
		}
	}
	return "", oops.In("cli_output").With("format", value).Errorf("unsupported output format: %s", value)
}

func (m OutputMode) String() string {
	if m == "" {
		return string(DefaultOutputMode)
	}
	return string(m)
}

func (m *OutputMode) Set(value string) error {
	mode, err := ParseOutputMode(value)
	if err != nil {
		return err
	}
	*m = mode
	return nil
}

func (m OutputMode) Type() string {
	return "output"
}

type resultHandler func(cmd *cobra.Command, args []string) (any, error)

type JSONOutput interface {
	JSONValue() any
}

type NDJSONOutput interface {
	NDJSONRows() any
}

type CSVOutput interface {
	CSVRows() any
}

type TableOutput interface {
	TableRows() (header []string, rows [][]string)
}

type TableAlignmentOutput interface {
	TableAlignments() []string
}

type DefaultOutputModeOutput interface {
	DefaultOutputMode() string
}

type TableBlock struct {
	Title  string
	Header []string
	Rows   [][]string
}

type MultiTableOutput interface {
	TableBlocks() []TableBlock
}

func runResult(opts *Options, handler resultHandler) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		result, err := handler(cmd, args)
		if err != nil {
			return err
		}
		output := opts.Output
		if !cmd.Flags().Changed("output") {
			if preferred, ok := result.(DefaultOutputModeOutput); ok {
				resolved, parseErr := ParseOutputMode(preferred.DefaultOutputMode())
				if parseErr != nil {
					return parseErr
				}
				output = resolved
			}
		}
		return Render(cmd.OutOrStdout(), output, result)
	}
}

func runJSONResult(handler resultHandler) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		result, err := handler(cmd, args)
		if err != nil {
			return err
		}
		return writeIndentedJSON(cmd.OutOrStdout(), result)
	}
}

func Render(w io.Writer, output OutputMode, result any) error {
	errb := oops.In("cli_output").With("format", output)
	switch output {
	case "", OutputModeTable:
		if value, ok := result.(MultiTableOutput); ok {
			return writeTableBlocks(w, value.TableBlocks())
		}
		if value, ok := result.(TableOutput); ok {
			header, rows := value.TableRows()
			alignments := []string(nil)
			if aligned, ok := result.(TableAlignmentOutput); ok {
				alignments = aligned.TableAlignments()
			}
			return writeAlignedTable(w, header, rows, alignments)
		}
		return writeTableValue(w, result)
	case OutputModeJSON:
		if value, ok := result.(JSONOutput); ok {
			result = value.JSONValue()
		}
		return writeIndentedJSON(w, result)
	case OutputModeNDJSON:
		if value, ok := result.(NDJSONOutput); ok {
			result = value.NDJSONRows()
		}
		return writeNDJSONValue(w, result)
	case OutputModeCSV:
		if value, ok := result.(CSVOutput); ok {
			result = value.CSVRows()
		}
		return writeCSV(w, result)
	default:
		return errb.Errorf("unsupported output format: %s", output)
	}
}

func writeTableBlocks(w io.Writer, blocks []TableBlock) error {
	for index, block := range blocks {
		if len(block.Header) == 0 {
			continue
		}
		if index > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return oops.In("cli_output").With("block", index).Wrapf(err, "write table separator")
			}
		}
		if strings.TrimSpace(block.Title) != "" {
			if _, err := fmt.Fprintln(w, block.Title); err != nil {
				return oops.In("cli_output").With("block", index).Wrapf(err, "write table title")
			}
		}
		if err := writeTable(w, block.Header, block.Rows); err != nil {
			return oops.In("cli_output").With("block", index).Wrap(err)
		}
	}
	return nil
}

func writeIndentedJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return oops.In("cli_output").Wrap(encoder.Encode(value))
}

func writeJSONLine(w io.Writer, value any) error {
	return oops.In("cli_output").Wrap(json.NewEncoder(w).Encode(value))
}

func writeNDJSONValue(w io.Writer, value any) error {
	if value == nil {
		return writeJSONLine(w, nil)
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return writeJSONLine(w, nil)
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		encoder := json.NewEncoder(w)
		for i := 0; i < rv.Len(); i++ {
			if err := encoder.Encode(rv.Index(i).Interface()); err != nil {
				return oops.In("cli_output").With("row", i).Wrapf(err, "write ndjson row")
			}
		}
		return nil
	default:
		return writeJSONLine(w, value)
	}
}

func writeTable(w io.Writer, header []string, rows [][]string) error {
	return writeAlignedTable(w, header, rows, nil)
}

func writeAlignedTable(w io.Writer, header []string, rows [][]string, alignments []string) error {
	errb := oops.In("cli_output").With("columns", len(header), "rows", len(rows))
	table := newAlignedOutputTable(w, alignments)
	table.Header(header)
	if err := table.Bulk(rows); err != nil {
		return errb.Wrapf(err, "write table rows")
	}
	return errb.Wrap(table.Render())
}

func writeTableValue(w io.Writer, value any) error {
	errb := oops.In("cli_output")
	table := newOutputTable(w)
	if value == nil {
		table.Header([]string{"value"})
		if err := table.Append(""); err != nil {
			return errb.Wrapf(err, "write table row")
		}
		return errb.Wrap(table.Render())
	}
	rv := reflect.ValueOf(value)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			table.Header([]string{"value"})
			if err := table.Append(""); err != nil {
				return errb.Wrapf(err, "write table row")
			}
			return errb.Wrap(table.Render())
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Len() == 0 {
			return nil
		}
		if err := table.Bulk(value); err != nil {
			return errb.Wrapf(err, "write table rows")
		}
	default:
		if err := table.Append(value); err != nil {
			return errb.Wrapf(err, "write table row")
		}
	}
	return errb.Wrap(table.Render())
}

func newOutputTable(w io.Writer) *tablewriter.Table {
	return newAlignedOutputTable(w, nil)
}

func newAlignedOutputTable(w io.Writer, alignments []string) *tablewriter.Table {
	tableOptions := []tablewriter.Option{
		tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Lines:      tw.LinesNone,
				Separators: tw.SeparatorsNone,
			},
			Symbols: tw.NewSymbols(tw.StyleNone),
		})),
		tablewriter.WithHeaderAlignment(tw.AlignLeft),
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRowAutoFormat(tw.Off),
		tablewriter.WithPadding(tw.Padding{Right: "  ", Overwrite: true}),
	}
	if parsed := parseTableAlignments(alignments); len(parsed) > 0 {
		tableOptions = append(tableOptions, tablewriter.WithAlignment(parsed))
	}
	return tablewriter.NewTable(w,
		tableOptions...,
	)
}

func parseTableAlignments(alignments []string) tw.Alignment {
	out := make(tw.Alignment, 0, len(alignments))
	for _, align := range alignments {
		switch strings.ToLower(strings.TrimSpace(align)) {
		case "right":
			out = append(out, tw.AlignRight)
		case "center":
			out = append(out, tw.AlignCenter)
		case "left", "":
			out = append(out, tw.AlignLeft)
		default:
			out = append(out, tw.AlignLeft)
		}
	}
	return out
}

func writeCSV(w io.Writer, rows any) error {
	if mapRows, ok := rows.([]map[string]any); ok {
		return writeCSVMapRows(w, mapRows)
	}
	writer := csv.NewWriter(w)
	encoder := csvutil.NewEncoder(writer)
	if err := encoder.Encode(rows); err != nil {
		return oops.In("cli_output").Wrapf(err, "write csv")
	}
	writer.Flush()
	return oops.In("cli_output").Wrap(writer.Error())
}

func writeCSVMapRows(w io.Writer, rows []map[string]any) error {
	writer := csv.NewWriter(w)
	headers := csvMapHeaders(rows)
	if len(headers) == 0 {
		return nil
	}
	if err := writer.Write(headers); err != nil {
		return oops.In("cli_output").Wrapf(err, "write csv header")
	}
	for index, row := range rows {
		record := make([]string, 0, len(headers))
		for _, header := range headers {
			record = append(record, csvMapValue(row[header]))
		}
		if err := writer.Write(record); err != nil {
			return oops.In("cli_output").With("row", index).Wrapf(err, "write csv row")
		}
	}
	writer.Flush()
	return oops.In("cli_output").Wrap(writer.Error())
}

func csvMapHeaders(rows []map[string]any) []string {
	seen := map[string]struct{}{}
	for _, row := range rows {
		for key := range row {
			seen[key] = struct{}{}
		}
	}
	preferred := []string{
		"ordinal",
		"symbol",
		"name",
		"first_date",
		"first_open",
		"first_close",
		"latest_date",
		"latest_close",
		"return_from_first_open_pct",
		"return_from_first_close_pct",
		"latest_traded_amount",
		"avg_traded_amount_5d",
		"avg_traded_amount_20d",
	}
	headers := make([]string, 0, len(seen))
	for _, key := range preferred {
		if _, ok := seen[key]; ok {
			headers = append(headers, key)
			delete(seen, key)
		}
	}
	rest := make([]string, 0, len(seen))
	for key := range seen {
		rest = append(rest, key)
	}
	sort.Strings(rest)
	return append(headers, rest...)
}

func csvMapValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	case []string:
		return strings.Join(typed, ",")
	default:
		rv := reflect.ValueOf(value)
		if rv.IsValid() {
			switch rv.Kind() {
			case reflect.Map, reflect.Slice, reflect.Array:
				data, err := json.Marshal(value)
				if err == nil {
					return string(data)
				}
			}
		}
		return fmt.Sprint(value)
	}
}
