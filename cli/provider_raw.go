package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	kisclient "github.com/ev3rlit/mwosa/clients/kis"
	provider "github.com/ev3rlit/mwosa/providers/core"
	kisprovider "github.com/ev3rlit/mwosa/providers/kis"
	"github.com/ev3rlit/mwosa/storage"
	"github.com/ev3rlit/mwosa/storage/providerauth"
	"github.com/ev3rlit/mwosa/storage/providerraw"
	"github.com/samber/oops"
	"github.com/spf13/cobra"
)

type providerRawSnapshotFlags struct {
	Group          string
	Operation      string
	From           string
	To             string
	Limit          int
	IncludePayload bool
}

type providerRawSnapshotsOutput struct {
	Snapshots []providerraw.SnapshotRecord `json:"snapshots"`
}

type providerRawInputFlags struct {
	InputJSON string
	InputFile string
	Params    []string
	Friendly  map[string]*string
	FlagNames map[string]string
}

type kisAPIListOutput struct {
	Services []kisAPIOutputRow `json:"services"`
}

type kisAPIOutputRow struct {
	Group            string `json:"provider_group" csv:"group"`
	APIID            string `json:"api_id" csv:"api_id"`
	Method           string `json:"method" csv:"method"`
	Endpoint         string `json:"endpoint" csv:"endpoint"`
	Description      string `json:"description" csv:"description"`
	CanonicalSupport string `json:"canonical_support" csv:"canonical_support"`
}

type kisAPIInspectOutput struct {
	Operation       kisprovider.RawOperation `json:"operation"`
	RequestTemplate map[string]string        `json:"request_template,omitempty"`
}

type kisRawOutput struct {
	Result kisprovider.RawResult
}

func registerProviderRawCommands(roots commandRoots, opts *Options) {
	roots.Get.AddCommand(newGetProviderRawSnapshotsCommand(opts))
	roots.Get.AddCommand(newGetProviderRawCommand(opts))
	roots.Fetch.AddCommand(newFetchProviderRawCommand(opts))
	roots.Sync.AddCommand(newSyncProviderRawCommand(opts))
	roots.Inspect.AddCommand(newInspectProviderAPICommand(opts))
}

func newGetProviderRawSnapshotsCommand(opts *Options) *cobra.Command {
	flags := providerRawSnapshotFlags{Limit: 50}
	cmd := &cobra.Command{
		Use:   "provider-raw-snapshots",
		Short: "Read stored provider-native raw snapshots",
		Long: strings.TrimSpace(`Read stored provider-native raw snapshots.

This reads provider_raw_snapshots from local SQLite. It is an escape hatch for
provider APIs that are not yet canonicalized, while keeping canonical analysis
tables separate from provider-native payloads.`),
		Args: cobra.NoArgs,
		RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
			return readProviderRawSnapshots(cmd, opts, providerraw.Query{
				Provider:       provider.ProviderID(opts.Provider),
				Group:          provider.GroupID(flags.Group),
				Operation:      provider.OperationID(flags.Operation),
				From:           flags.From,
				To:             flags.To,
				Limit:          flags.Limit,
				IncludePayload: flags.IncludePayload,
			})
		}),
	}
	addProviderRawSnapshotFlags(cmd, &flags)
	return cmd
}

func newGetProviderRawCommand(opts *Options) *cobra.Command {
	flags := providerRawSnapshotFlags{Limit: 50}
	cmd := &cobra.Command{
		Use:   "provider-raw [provider] [operation]",
		Short: "Read stored provider-native raw payload snapshots",
		Long: strings.TrimSpace(`Read stored provider-native raw payload snapshots.

This is a friendlier alias over provider_raw_snapshots for canonicalization
escape hatches. It does not call the provider live; it only reads snapshots that
previous sync commands have already stored locally.`),
		Args: cobra.MaximumNArgs(2),
		RunE: runResult(opts, func(cmd *cobra.Command, args []string) (result any, err error) {
			query, err := providerRawQueryFromArgs(opts, flags, args)
			if err != nil {
				return nil, err
			}
			return readProviderRawSnapshots(cmd, opts, query)
		}),
	}
	addProviderRawSnapshotFlags(cmd, &flags)
	return cmd
}

func newFetchProviderRawCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-raw",
		Short: "Fetch a provider-native raw API response live",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newKISProviderRawCommand(opts, false))
	return cmd
}

func newSyncProviderRawCommand(opts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider-raw",
		Short: "Fetch and store a provider-native raw API snapshot",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newKISProviderRawCommand(opts, true))
	return cmd
}

func newKISProviderRawCommand(opts *Options, syncSnapshot bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "kis",
		Short:             "Use generated KIS raw API registry",
		Args:              cobra.NoArgs,
		ValidArgsFunction: completeKISRawOperations,
	}
	for _, operation := range kisclient.RawOperations() {
		operation := operation
		flags := providerRawInputFlags{Friendly: map[string]*string{}, FlagNames: map[string]string{}}
		operationCommand := &cobra.Command{
			Use:   operation.OperationID,
			Short: firstNonEmpty(operation.Description, operation.Summary),
			Args:  cobra.NoArgs,
			RunE: runResult(opts, func(cmd *cobra.Command, _ []string) (result any, err error) {
				input, err := collectKISRawInput(cmd, operation, flags)
				if err != nil {
					return nil, err
				}
				rawResult, err := fetchKISRaw(cmd, opts, provider.OperationID(operation.OperationID), input)
				if err != nil {
					return nil, err
				}
				if !syncSnapshot {
					return kisRawOutput{Result: rawResult}, nil
				}
				database := storage.NewDatabase(opts.Database)
				defer func() {
					err = oops.Join(err, database.Close())
				}()
				repository, err := providerraw.NewRepository(database)
				if err != nil {
					return nil, err
				}
				return repository.UpsertSnapshot(cmd.Context(), providerraw.Snapshot{
					Provider:         rawResult.Provider,
					Group:            rawResult.Group,
					Operation:        rawResult.Operation,
					BaseDate:         rawResult.BaseDate,
					CanonicalSupport: rawResult.Canonical,
					Rows:             rawResult.Response,
					RowCount:         rawResult.RowCount,
				})
			}),
		}
		addKISRawInputFlags(operationCommand, operation, &flags)
		cmd.AddCommand(operationCommand)
	}
	return cmd
}

func newInspectProviderAPICommand(opts *Options) *cobra.Command {
	flags := struct {
		RequestTemplate bool
	}{}
	cmd := &cobra.Command{
		Use:               "provider-api <provider> <operation>",
		Short:             "Inspect a provider-native API schema",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeInspectProviderAPIArgs,
		RunE: runResult(opts, func(_ *cobra.Command, args []string) (any, error) {
			if provider.ProviderID(args[0]) != provider.ProviderKIS {
				return nil, oops.In("cli").With("provider", args[0]).New("provider API inspect is only available for kis")
			}
			operation, ok := kisclient.LookupRawOperation(args[1])
			if !ok {
				return nil, oops.In("cli").With("provider", args[0], "operation", args[1]).New("provider API operation not found")
			}
			out := kisAPIInspectOutput{Operation: kisRawOperationFromClient(operation)}
			if flags.RequestTemplate {
				template, err := kisclient.RawRequestTemplate(args[1])
				if err != nil {
					return nil, err
				}
				out.RequestTemplate = template
			}
			return out, nil
		}),
	}
	cmd.Flags().BoolVar(&flags.RequestTemplate, "request-template", flags.RequestTemplate, "include a KIS field request template")
	return cmd
}

func addKISRawInputFlags(cmd *cobra.Command, operation kisclient.RawOperationMetadata, flags *providerRawInputFlags) {
	cmd.Flags().StringVar(&flags.InputJSON, "input-json", flags.InputJSON, "KIS raw request JSON object keyed by original KIS field")
	cmd.Flags().StringVar(&flags.InputFile, "input-file", flags.InputFile, "KIS raw request JSON file keyed by original KIS field")
	cmd.Flags().StringArrayVar(&flags.Params, "param", flags.Params, "advanced KIS raw parameter override, KEY=VALUE")
	mustMarkFlagFilename(cmd, "input-file", "json")
	mustRegisterFlagCompletion(cmd, "param", completeKISRawParamKeys(operation))

	seenFlags := map[string]int{}
	for _, parameter := range operation.Parameters {
		if parameter.Source == "system" {
			continue
		}
		flagName := uniqueKISRawFlagName(parameter.Flag, parameter.Name, seenFlags)
		value := ""
		flags.Friendly[parameter.Name] = &value
		flags.FlagNames[parameter.Name] = flagName
		usage := parameter.Description
		if usage == "" {
			usage = parameter.Name
		}
		if parameter.Name != "" {
			usage = usage + " (KIS field: " + parameter.Name + ")"
		}
		cmd.Flags().StringVar(&value, flagName, value, usage)
		if len(parameter.Completion) > 0 {
			mustRegisterFlagCompletion(cmd, flagName, completeFixedValues(parameter.Completion))
		}
	}
}

func collectKISRawInput(cmd *cobra.Command, operation kisclient.RawOperationMetadata, flags providerRawInputFlags) (map[string]string, error) {
	errb := oops.In("cli").With("provider", provider.ProviderKIS, "operation", operation.OperationID)
	input := map[string]string{}
	seen := map[string]string{}

	if strings.TrimSpace(flags.InputJSON) != "" && strings.TrimSpace(flags.InputFile) != "" {
		return nil, errb.New("kis raw input accepts only one of --input-json or --input-file")
	}
	if strings.TrimSpace(flags.InputJSON) != "" {
		values, err := decodeKISRawJSONInput(cmd, flags.InputJSON)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		if err := mergeKISRawValues(input, seen, values, "input-json"); err != nil {
			return nil, errb.Wrap(err)
		}
	}
	if strings.TrimSpace(flags.InputFile) != "" {
		values, err := decodeKISRawJSONFile(flags.InputFile)
		if err != nil {
			return nil, errb.Wrap(err)
		}
		if err := mergeKISRawValues(input, seen, values, "input-file"); err != nil {
			return nil, errb.Wrap(err)
		}
	}
	for _, parameter := range operation.Parameters {
		value, ok := flags.Friendly[parameter.Name]
		if !ok || value == nil {
			continue
		}
		flagName := flags.FlagNames[parameter.Name]
		if !cmd.Flags().Changed(flagName) {
			continue
		}
		if err := mergeKISRawValue(input, seen, parameter.Name, *value, "flag --"+flagName); err != nil {
			return nil, errb.Wrap(err)
		}
	}
	params, err := parseKISRawParams(flags.Params)
	if err != nil {
		return nil, errb.Wrap(err)
	}
	if err := mergeKISRawValues(input, seen, params, "param"); err != nil {
		return nil, errb.Wrap(err)
	}
	return input, nil
}

func decodeKISRawJSONInput(cmd *cobra.Command, value string) (map[string]string, error) {
	if strings.TrimSpace(value) == "-" {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, oops.In("cli").Wrapf(err, "read kis raw JSON input from stdin")
		}
		return decodeKISRawJSONObject(data)
	}
	return decodeKISRawJSONObject([]byte(value))
}

func decodeKISRawJSONFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, oops.In("cli").With("path", path).Wrapf(err, "read kis raw JSON input file")
	}
	return decodeKISRawJSONObject(data)
}

func decodeKISRawJSONObject(data []byte) (map[string]string, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, oops.In("cli").Wrapf(err, "decode kis raw JSON input")
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		switch typed := value.(type) {
		case nil:
			out[key] = ""
		case string:
			out[key] = typed
		default:
			out[key] = fmt.Sprint(typed)
		}
	}
	return out, nil
}

func parseKISRawParams(params []string) (map[string]string, error) {
	out := map[string]string{}
	for _, param := range params {
		key, value, ok := strings.Cut(param, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, oops.In("cli").With("param", param).New("kis raw --param must be KEY=VALUE")
		}
		if _, exists := out[key]; exists {
			return nil, oops.In("cli").With("param", key).New("kis raw --param key is duplicated")
		}
		out[key] = value
	}
	return out, nil
}

func mergeKISRawValues(target map[string]string, seen map[string]string, values map[string]string, source string) error {
	for key, value := range values {
		if err := mergeKISRawValue(target, seen, key, value, source); err != nil {
			return err
		}
	}
	return nil
}

func mergeKISRawValue(target map[string]string, seen map[string]string, key string, value string, source string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return oops.In("cli").With("source", source).New("kis raw input key is empty")
	}
	if previous, ok := seen[key]; ok {
		return oops.In("cli").With("field", key, "previous_source", previous, "source", source).New("kis raw input field is provided more than once")
	}
	seen[key] = source
	target[key] = value
	return nil
}

func fetchKISRaw(cmd *cobra.Command, opts *Options, operationID provider.OperationID, input map[string]string) (kisprovider.RawResult, error) {
	if err := loadConfig(opts); err != nil {
		return kisprovider.RawResult{}, err
	}
	authDB := providerauth.NewDatabase(opts.ProviderAuthDatabase)
	tokenCache, err := providerauth.NewRepository(authDB)
	if err != nil {
		return kisprovider.RawResult{}, err
	}
	builder := kisprovider.NewBuilder().WithTokenCache(tokenCache)
	instance, err := builder.Build(opts.ProviderConfig)
	if err != nil {
		return kisprovider.RawResult{}, oops.Join(err, authDB.Close())
	}
	kisProvider, ok := instance.(*kisprovider.Provider)
	if !ok {
		return kisprovider.RawResult{}, oops.Join(
			oops.In("cli").With("provider", provider.ProviderKIS).New("configured KIS provider has unexpected type"),
			authDB.Close(),
		)
	}
	result, err := kisProvider.FetchRaw(cmd.Context(), kisprovider.RawRequest{
		OperationID: operationID,
		Input:       input,
	})
	return result, oops.Join(err, authDB.Close())
}

func uniqueKISRawFlagName(flag string, field string, seen map[string]int) string {
	flag = strings.TrimSpace(flag)
	if flag == "" {
		flag = strings.ToLower(strings.ReplaceAll(field, "_", "-"))
	}
	seen[flag]++
	if seen[flag] == 1 {
		return flag
	}
	return flag + "-" + strings.ToLower(strings.ReplaceAll(field, "_", "-"))
}

func completeKISRawOperations(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	operations := kisclient.RawOperations()
	values := make([]cobra.Completion, 0, len(operations))
	for _, operation := range operations {
		values = append(values, cobra.CompletionWithDesc(operation.OperationID, firstNonEmpty(operation.Description, operation.Summary)))
	}
	return values, cobra.ShellCompDirectiveNoFileComp
}

func completeInspectProviderAPIArgs(_ *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return []cobra.Completion{cobra.CompletionWithDesc(string(provider.ProviderKIS), "KIS raw API registry")}, cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 && provider.ProviderID(args[0]) == provider.ProviderKIS {
		return completeKISRawOperations(nil, nil, "")
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func completeKISRawParamKeys(operation kisclient.RawOperationMetadata) cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		values := make([]cobra.Completion, 0, len(operation.Parameters))
		for _, parameter := range operation.Parameters {
			values = append(values, cobra.CompletionWithDesc(parameter.Name+"=", parameter.Description))
		}
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeFixedValues(values []string) cobra.CompletionFunc {
	return func(_ *cobra.Command, _ []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
		completions := make([]cobra.Completion, 0, len(values))
		for _, value := range values {
			completions = append(completions, cobra.Completion(value))
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func kisRawOperationFromClient(operation kisclient.RawOperationMetadata) kisprovider.RawOperation {
	return kisprovider.RawOperation{
		OperationID:     provider.OperationID(operation.OperationID),
		Endpoint:        operation.Endpoint,
		Method:          operation.Method,
		Group:           provider.GroupID(operation.Group),
		ServiceGroup:    operation.ServiceGroup,
		RoleHint:        operation.RoleHint,
		Summary:         operation.Summary,
		Description:     operation.Description,
		RealTRID:        operation.RealTRID,
		VirtualTRID:     operation.VirtualTRID,
		SupportsVirtual: operation.SupportsVirtual,
		Parameters:      operation.Parameters,
	}
}

func kisRawCanonicalSupport(roleHint string) string {
	switch strings.TrimSpace(roleHint) {
	case "", "read_only", "market_scan":
		return "raw_only"
	default:
		return roleHint
	}
}

func addProviderRawSnapshotFlags(cmd *cobra.Command, flags *providerRawSnapshotFlags) {
	cmd.Flags().StringVar(&flags.Group, "group", flags.Group, "provider group filter")
	cmd.Flags().StringVar(&flags.Operation, "operation", flags.Operation, "provider operation/api id filter")
	cmd.Flags().StringVar(&flags.From, "from", flags.From, "base date lower bound, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().StringVar(&flags.To, "to", flags.To, "base date upper bound, YYYYMMDD or YYYY-MM-DD")
	cmd.Flags().IntVar(&flags.Limit, "limit", flags.Limit, "maximum snapshots to return")
	cmd.Flags().BoolVar(&flags.IncludePayload, "include-payload", flags.IncludePayload, "include decoded provider-native payload in JSON/NDJSON output")
}

func providerRawQueryFromArgs(opts *Options, flags providerRawSnapshotFlags, args []string) (providerraw.Query, error) {
	providerID := strings.TrimSpace(opts.Provider)
	operation := strings.TrimSpace(flags.Operation)
	switch len(args) {
	case 0:
	case 1:
		if providerID == "" {
			providerID = strings.TrimSpace(args[0])
		} else if operation == "" {
			operation = strings.TrimSpace(args[0])
		} else if operation != strings.TrimSpace(args[0]) {
			return providerraw.Query{}, oops.In("cli").With("operation", operation, "arg", args[0]).New("provider raw operation flag conflicts with positional operation")
		}
	case 2:
		if providerID != "" && providerID != strings.TrimSpace(args[0]) {
			return providerraw.Query{}, oops.In("cli").With("provider", providerID, "arg", args[0]).New("provider raw provider flag conflicts with positional provider")
		}
		providerID = strings.TrimSpace(args[0])
		if operation != "" && operation != strings.TrimSpace(args[1]) {
			return providerraw.Query{}, oops.In("cli").With("operation", operation, "arg", args[1]).New("provider raw operation flag conflicts with positional operation")
		}
		operation = strings.TrimSpace(args[1])
	}
	return providerraw.Query{
		Provider:       provider.ProviderID(providerID),
		Group:          provider.GroupID(flags.Group),
		Operation:      provider.OperationID(operation),
		From:           flags.From,
		To:             flags.To,
		Limit:          flags.Limit,
		IncludePayload: flags.IncludePayload,
	}, nil
}

func readProviderRawSnapshots(cmd *cobra.Command, opts *Options, query providerraw.Query) (result any, err error) {
	runtime, err := newAppRuntime(opts, false)
	if err != nil {
		return nil, err
	}
	defer closeAppRuntime(runtime, &err)

	repository, err := providerraw.NewRepository(runtime.Storage.Database)
	if err != nil {
		return nil, err
	}
	snapshots, err := repository.ListSnapshots(cmd.Context(), query)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, oops.In("cli").With("provider", query.Provider, "operation", query.Operation).New("stored provider raw snapshots not found")
	}
	return providerRawSnapshotsOutput{Snapshots: snapshots}, nil
}

func (o providerRawSnapshotsOutput) JSONValue() any {
	return o.Snapshots
}

func (o providerRawSnapshotsOutput) NDJSONRows() any {
	return o.Snapshots
}

func (o providerRawSnapshotsOutput) CSVRows() any {
	return o.Snapshots
}

func (o providerRawSnapshotsOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Snapshots))
	for _, snapshot := range o.Snapshots {
		rows = append(rows, []string{
			string(snapshot.Provider),
			string(snapshot.Group),
			string(snapshot.Operation),
			snapshot.BaseDate,
			snapshot.CanonicalSupport,
			strconv.Itoa(snapshot.RowCount),
		})
	}
	return []string{"provider", "group", "operation", "base_date", "canonical_support", "row_count"}, rows
}

func (o kisAPIListOutput) JSONValue() any {
	return o.Services
}

func (o kisAPIListOutput) NDJSONRows() any {
	return o.Services
}

func (o kisAPIListOutput) CSVRows() any {
	return o.Services
}

func (o kisAPIListOutput) TableRows() ([]string, [][]string) {
	rows := make([][]string, 0, len(o.Services))
	for _, service := range o.Services {
		rows = append(rows, []string{service.Group, service.APIID, service.Method, service.Endpoint, service.Description, service.CanonicalSupport})
	}
	return []string{"group", "api_id", "method", "endpoint", "description", "canonical_support"}, rows
}

func (o kisAPIInspectOutput) JSONValue() any {
	return o
}

func (o kisAPIInspectOutput) TableRows() ([]string, [][]string) {
	rows := [][]string{{
		string(o.Operation.Group),
		string(o.Operation.OperationID),
		o.Operation.Method,
		o.Operation.Endpoint,
		o.Operation.RoleHint,
	}}
	for _, parameter := range o.Operation.Parameters {
		rows = append(rows, []string{
			string(o.Operation.Group),
			"--" + parameter.Flag,
			parameter.Name,
			parameter.Source,
			strconv.FormatBool(parameter.Required),
		})
	}
	return []string{"group", "field", "kis_field", "source", "required"}, rows
}

func (o kisRawOutput) JSONValue() any {
	return o.Result
}

func (o kisRawOutput) NDJSONRows() any {
	return o.Result.Response
}

func (o kisRawOutput) CSVRows() any {
	return o.Result.Response
}

func (o kisRawOutput) TableRows() ([]string, [][]string) {
	return []string{"provider", "group", "operation", "endpoint", "rows", "canonical_support"}, [][]string{{
		string(o.Result.Provider),
		string(o.Result.Group),
		string(o.Result.Operation),
		o.Result.Endpoint,
		strconv.Itoa(o.Result.RowCount),
		o.Result.Canonical,
	}}
}
