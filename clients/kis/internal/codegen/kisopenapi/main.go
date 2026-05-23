package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

type catalog struct {
	SchemaVersion any                 `json:"schemaVersion"`
	Source        any                 `json:"source"`
	FetchedAt     string              `json:"fetchedAt"`
	Collections   []catalogCollection `json:"collections"`
}

type catalogCollection struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	APIs []catalogAPI `json:"apis"`
}

type catalogAPI struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	APISummary       string            `json:"apiSummary"`
	Description      string            `json:"description"`
	AccessURL        string            `json:"accessUrl"`
	HTTPMethod       string            `json:"httpMethod"`
	ContentType      string            `json:"contentType"`
	RealDomain       string            `json:"realDomain"`
	VirtualDomain    string            `json:"virtualDomain"`
	RealTRID         string            `json:"realTrId"`
	VirtualTRID      string            `json:"virtualTrId"`
	OAuth            bool              `json:"oauth"`
	SourceDetailURL  string            `json:"sourceDetailUrl"`
	Properties       []catalogProperty `json:"properties"`
	LastModifiedDate string            `json:"lastModifiedDate"`
}

type catalogProperty struct {
	BodyType       string `json:"bodyType"`
	PropertyCD     string `json:"propertyCd"`
	PropertyName   string `json:"propertyNm"`
	PropertyType   string `json:"propertyType"`
	PropertyLength string `json:"propertyLength"`
	PropertyOrder  string `json:"propertyOrder"`
	RequireYN      string `json:"requireYn"`
	Description    string `json:"description"`
}

type operationConfig struct {
	AccessURL     string
	SplitPath     string
	OperationID   string
	GoName        string
	RawFunction   string
	Group         string
	ServiceGroup  string
	GroupMethod   string
	RoleCandidate string
	SecurityType  string
	DocSummary    string
}

type splitFile struct {
	Config operationConfig
	API    catalogAPI
	File   string
	Doc    map[string]any
}

type sdkNamesFile struct {
	Version    int                          `yaml:"version"`
	Operations map[string]operationOverride `yaml:"operations"`
}

type operationOverride struct {
	Enabled      *bool       `yaml:"enabled"`
	AccessURL    string      `yaml:"access_url"`
	SplitPath    string      `yaml:"split_path"`
	GoName       string      `yaml:"go_name"`
	RawFunction  string      `yaml:"raw_function"`
	GroupMethod  string      `yaml:"group_method"`
	PublicMethod string      `yaml:"public_method"`
	ServiceGroup string      `yaml:"service_group"`
	Group        string      `yaml:"group"`
	RoleHint     string      `yaml:"role_hint"`
	SecurityType string      `yaml:"security_type"`
	Doc          docOverride `yaml:"doc"`
}

type docOverride struct {
	Summary string `yaml:"summary"`
}

type groupsFile struct {
	Version int                           `yaml:"version"`
	Groups  map[string]serviceGroupConfig `yaml:"groups"`
}

type serviceGroupConfig struct {
	ServiceName    string   `yaml:"service_name"`
	SourceCategory string   `yaml:"source_category"`
	Operations     []string `yaml:"operations"`
}

type roleMappingFile struct {
	Version  int                            `yaml:"version"`
	RoleHint map[string]roleMappingOverride `yaml:"role_hints"`
}

type roleMappingOverride struct {
	Candidate       string `yaml:"candidate"`
	AdapterBoundary string `yaml:"adapter_boundary"`
}

type nestedStruct struct {
	Name  string
	Props []catalogProperty
}

func main() {
	catalogPath := flag.String("catalog", "docs/domestic-stock-openapi-catalog.json", "KIS domestic stock catalog path")
	overridesDir := flag.String("overrides", "overrides", "KIS codegen override directory")
	openAPIOut := flag.String("openapi-out", "openapi", "OpenAPI output directory")
	rawAPIOut := flag.String("rawapi-out", "internal/generated/rawapi", "generated rawapi output directory")
	publicOut := flag.String("public-out", ".", "generated public package output directory")
	includeAllCatalog := flag.Bool("include-all-catalog", false, "include every supported catalog API, not only override/group selections")
	flag.Parse()

	data, err := os.ReadFile(*catalogPath)
	if err != nil {
		fatalf("read catalog: %v", err)
	}

	var source catalog
	if err := json.Unmarshal(data, &source); err != nil {
		fatalf("decode catalog: %v", err)
	}

	configs, err := loadOperationConfigs(source, *overridesDir, *includeAllCatalog)
	if err != nil {
		fatalf("load overrides: %v", err)
	}
	files, err := buildSplitFiles(source, configs)
	if err != nil {
		fatalf("build split files: %v", err)
	}
	if err := writeOpenAPI(*openAPIOut, source, files); err != nil {
		fatalf("write openapi: %v", err)
	}
	if err := writeRawAPI(*rawAPIOut, files); err != nil {
		fatalf("write rawapi: %v", err)
	}
	if err := writePublicPackage(*publicOut, files); err != nil {
		fatalf("write public package: %v", err)
	}
}

func loadOperationConfigs(source catalog, overridesDir string, includeAllCatalog bool) ([]operationConfig, error) {
	sdkNames, err := readYAML[sdkNamesFile](filepath.Join(overridesDir, "sdk-names.yaml"))
	if err != nil {
		return nil, err
	}
	groups, err := readYAML[groupsFile](filepath.Join(overridesDir, "groups.yaml"))
	if err != nil {
		return nil, err
	}
	roles, err := readYAML[roleMappingFile](filepath.Join(overridesDir, "role-mapping.yaml"))
	if err != nil {
		return nil, err
	}

	groupByOperation := map[string]string{}
	serviceGroupByOperation := map[string]string{}
	sourceCategoryByOperation := map[string]string{}
	for groupName, group := range groups.Groups {
		for _, operationID := range group.Operations {
			groupByOperation[operationID] = groupName
			serviceGroupByOperation[operationID] = group.ServiceName
			sourceCategoryByOperation[operationID] = group.SourceCategory
		}
	}

	overrideByAccessURL := map[string]string{}
	for operationID, override := range sdkNames.Operations {
		if override.AccessURL != "" {
			overrideByAccessURL[override.AccessURL] = operationID
		}
	}

	defaultIDCounts := map[string]int{}
	for _, collection := range source.Collections {
		for _, api := range collection.APIs {
			defaultIDCounts[operationIDFromAccessURL(api.AccessURL)]++
		}
	}

	configs := []operationConfig{}
	for _, collection := range source.Collections {
		for _, api := range collection.APIs {
			operationID := operationIDFromAccessURL(api.AccessURL)
			overrideKey, hasAccessOverride := overrideByAccessURL[api.AccessURL]
			if hasAccessOverride {
				operationID = overrideKey
			}
			if includeAllCatalog && !hasAccessOverride && defaultIDCounts[operationID] > 1 {
				operationID = operationIDFromAccessURLWithNamespace(api.AccessURL)
			}
			override := sdkNames.Operations[operationID]
			if override.AccessURL != "" && override.AccessURL != api.AccessURL {
				continue
			}

			sourceCategoryMatches := sourceCategoryByOperation[operationID] == "" || sourceCategoryByOperation[operationID] == collection.Name
			selectedByGroup := groupByOperation[operationID] != "" && sourceCategoryMatches
			selectedByOverride := override.Enabled != nil && *override.Enabled
			disabledByOverride := override.Enabled != nil && !*override.Enabled
			if disabledByOverride || (!includeAllCatalog && !selectedByOverride && !selectedByGroup) {
				continue
			}
			if !includeAllCatalog && !hasAccessOverride && defaultIDCounts[operationID] > 1 {
				return nil, fmt.Errorf("operation %s matches multiple catalog APIs; set access_url in overrides/sdk-names.yaml", operationID)
			}
			if reason := exclusionReason(collection, api); reason != "" {
				if selectedByOverride || selectedByGroup {
					return nil, fmt.Errorf("operation %s is excluded from generation: %s (access_url=%s)", operationID, reason, api.AccessURL)
				}
				continue
			}

			defaultGroup := defaultServiceGroup(collection.Name)
			serviceGroup := firstNonEmpty(override.ServiceGroup, serviceGroupByOperation[operationID], goName(defaultGroup))
			group := firstNonEmpty(override.Group, groupByOperation[operationID], defaultGroup)
			roleHint := firstNonEmpty(override.RoleHint, roles.RoleHint[operationID].Candidate, defaultRoleHint(collection, api))
			config := operationConfig{
				AccessURL:     firstNonEmpty(override.AccessURL, api.AccessURL),
				SplitPath:     firstNonEmpty(override.SplitPath, defaultSplitPath(api.AccessURL)),
				OperationID:   operationID,
				GoName:        firstNonEmpty(override.GoName, goName(operationID)),
				Group:         group,
				ServiceGroup:  serviceGroup,
				RoleCandidate: roleHint,
				SecurityType:  firstNonEmpty(override.SecurityType, "oauth2-access-token"),
				DocSummary:    strings.TrimSpace(override.Doc.Summary),
			}
			config.RawFunction = firstNonEmpty(override.RawFunction, config.GoName)
			config.GroupMethod = firstNonEmpty(override.GroupMethod, override.PublicMethod, config.GoName)
			if err := validateOperationConfig(config); err != nil {
				return nil, err
			}
			configs = append(configs, config)
		}
	}
	if err := validateOperationConfigs(configs); err != nil {
		return nil, err
	}
	return configs, nil
}

func validateOperationConfig(config operationConfig) error {
	missing := []string{}
	if config.AccessURL == "" {
		missing = append(missing, "access_url")
	}
	if config.SplitPath == "" {
		missing = append(missing, "split_path")
	}
	if config.GoName == "" {
		missing = append(missing, "go_name")
	}
	if config.RawFunction == "" {
		missing = append(missing, "raw_function")
	}
	if config.Group == "" {
		missing = append(missing, "group")
	}
	if config.ServiceGroup == "" {
		missing = append(missing, "service_group")
	}
	if config.GroupMethod == "" {
		missing = append(missing, "group_method")
	}
	if config.RoleCandidate == "" {
		missing = append(missing, "role_hint")
	}
	if len(missing) > 0 {
		return fmt.Errorf("operation %s missing override fields: %s", config.OperationID, strings.Join(missing, ", "))
	}
	return nil
}

func validateOperationConfigs(configs []operationConfig) error {
	seenOperationID := map[string]operationConfig{}
	seenGoName := map[string]operationConfig{}
	seenRawFunction := map[string]operationConfig{}
	seenGroupMethod := map[string]operationConfig{}
	seenSplitPath := map[string]operationConfig{}
	seenGroupFile := map[string]operationConfig{}
	for _, config := range configs {
		if previous, ok := seenOperationID[config.OperationID]; ok {
			return duplicateConfigError("operation_id", config.OperationID, previous, config)
		}
		seenOperationID[config.OperationID] = config

		if previous, ok := seenGoName[config.GoName]; ok {
			return duplicateConfigError("go_name", config.GoName, previous, config)
		}
		seenGoName[config.GoName] = config

		if previous, ok := seenRawFunction[config.RawFunction]; ok {
			return duplicateConfigError("raw_function", config.RawFunction, previous, config)
		}
		seenRawFunction[config.RawFunction] = config

		groupMethodKey := config.ServiceGroup + "." + config.GroupMethod
		if previous, ok := seenGroupMethod[groupMethodKey]; ok {
			return duplicateConfigError("group_method", groupMethodKey, previous, config)
		}
		seenGroupMethod[groupMethodKey] = config

		if previous, ok := seenSplitPath[config.SplitPath]; ok {
			return duplicateConfigError("split_path", config.SplitPath, previous, config)
		}
		seenSplitPath[config.SplitPath] = config

		groupFileKey := snakeName(config.Group)
		if previous, ok := seenGroupFile[groupFileKey]; ok && previous.Group != config.Group {
			return duplicateConfigError("generated_group_file", groupFileKey, previous, config)
		}
		seenGroupFile[groupFileKey] = config
	}
	return nil
}

func validateGeneratedSurface(files []splitFile) error {
	seenTypeName := map[string]splitFile{}
	seenAliasName := map[string]splitFile{}
	seenConstName := map[string]splitFile{}
	for _, file := range files {
		for _, typeName := range operationTypeNames(file) {
			if previous, ok := seenTypeName[typeName]; ok {
				return duplicateFileError("generated type", typeName, previous, file)
			}
			seenTypeName[typeName] = file
			if previous, ok := seenAliasName[typeName]; ok {
				return duplicateFileError("alias", typeName, previous, file)
			}
			seenAliasName[typeName] = file
		}
		for _, constName := range []string{
			"Operation" + file.Config.RawFunction,
			"Endpoint" + file.Config.RawFunction,
			"Group" + file.Config.RawFunction,
			"ServiceGroup" + file.Config.RawFunction,
			"RealTRID" + file.Config.RawFunction,
			"VirtualTRID" + file.Config.RawFunction,
			"SupportsVirtual" + file.Config.RawFunction,
		} {
			if previous, ok := seenConstName[constName]; ok {
				return duplicateFileError("constant", constName, previous, file)
			}
			seenConstName[constName] = file
		}
		if err := validateOperationFields(file); err != nil {
			return err
		}
	}
	return nil
}

func validateOperationFields(file splitFile) error {
	if err := validateFieldNames(file, file.Config.GoName+"Request", filterProperties(file.API.Properties, "req_b", "")); err != nil {
		return err
	}
	if err := validateFieldNames(file, file.Config.GoName+"Response", filterProperties(file.API.Properties, "res_b", "")); err != nil {
		return err
	}
	for _, root := range filterProperties(file.API.Properties, "res_b", "") {
		children := childProperties(file.API.Properties, root)
		switch root.PropertyType {
		case "A0003":
			if err := validateFieldNames(file, file.Config.GoName+goName(root.PropertyCD), children); err != nil {
				return err
			}
		case "A0005":
			if err := validateFieldNames(file, file.Config.GoName+goName(root.PropertyCD)+"Item", children); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateFieldNames(file splitFile, structName string, props []catalogProperty) error {
	seen := map[string]catalogProperty{}
	for _, prop := range props {
		fieldName := goName(prop.PropertyCD)
		if previous, ok := seen[fieldName]; ok {
			return fmt.Errorf(
				"field name collision: operation=%s struct=%s field=%s property=%s previous_property=%s access_url=%s",
				file.Config.OperationID,
				structName,
				fieldName,
				prop.PropertyCD,
				previous.PropertyCD,
				file.API.AccessURL,
			)
		}
		seen[fieldName] = prop
	}
	return nil
}

func duplicateConfigError(kind string, value string, previous operationConfig, current operationConfig) error {
	return fmt.Errorf(
		"%s collision: %s used by operations %s (%s) and %s (%s)",
		kind,
		value,
		previous.OperationID,
		previous.AccessURL,
		current.OperationID,
		current.AccessURL,
	)
}

func duplicateFileError(kind string, value string, previous splitFile, current splitFile) error {
	return fmt.Errorf(
		"%s collision: %s used by operations %s (%s) and %s (%s)",
		kind,
		value,
		previous.Config.OperationID,
		previous.API.AccessURL,
		current.Config.OperationID,
		current.API.AccessURL,
	)
}

func readYAML[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}

func buildSplitFiles(source catalog, configs []operationConfig) ([]splitFile, error) {
	apis := map[string]struct {
		collection catalogCollection
		api        catalogAPI
	}{}
	for _, collection := range source.Collections {
		for _, api := range collection.APIs {
			apis[api.AccessURL] = struct {
				collection catalogCollection
				api        catalogAPI
			}{collection: collection, api: api}
		}
	}

	files := make([]splitFile, 0, len(configs))
	for _, config := range configs {
		item, ok := apis[config.AccessURL]
		if !ok {
			return nil, fmt.Errorf("catalog API not found: %s", config.AccessURL)
		}
		files = append(files, splitFile{
			Config: config,
			API:    item.api,
			File:   config.SplitPath,
			Doc:    openAPIDoc(source, item.collection, item.api, config),
		})
	}
	if err := validateGeneratedSurface(files); err != nil {
		return nil, err
	}
	return files, nil
}

func openAPIDoc(source catalog, collection catalogCollection, api catalogAPI, config operationConfig) map[string]any {
	operation := map[string]any{
		"operationId": config.OperationID,
		"summary":     cleanText(api.Name),
		"description": cleanText(firstNonEmpty(config.DocSummary, api.APISummary, api.Description)),
		"tags":        []string{config.ServiceGroup},
		"parameters":  queryParameters(api),
		"responses": map[string]any{
			"200": map[string]any{
				"description": "KIS business response envelope",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{
							"$ref": "#/components/schemas/" + config.GoName + "Response",
						},
					},
				},
			},
		},
		"x-kis": xKIS(source, collection, api, config),
	}
	if api.OAuth {
		operation["security"] = []map[string][]string{{"KISOAuth": {}}}
	}

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "KIS Domestic Stock OpenAPI - " + config.OperationID,
			"version": "0.1.0",
		},
		"paths": map[string]any{
			api.AccessURL: map[string]any{
				strings.ToLower(api.HTTPMethod): operation,
			},
		},
		"components": map[string]any{
			"securitySchemes": securitySchemes(),
			"schemas": map[string]any{
				config.GoName + "Response": responseSchema(api),
			},
		},
	}
}

func writeOpenAPI(out string, source catalog, files []splitFile) error {
	for _, file := range files {
		if err := writeJSON(filepath.Join(out, file.File), file.Doc); err != nil {
			return err
		}
	}
	if err := writeJSON(filepath.Join(out, "kis.openapi.json"), rootIndex(source, files)); err != nil {
		return err
	}
	return writeJSON(filepath.Join(out, "kis.openapi.bundle.json"), bundle(source, files))
}

func rootIndex(source catalog, files []splitFile) map[string]any {
	paths := map[string]any{}
	splitFiles := make([]map[string]any, 0, len(files))
	for _, file := range files {
		paths[file.API.AccessURL] = map[string]any{
			"$ref": "./" + file.File + "#/paths/" + escapeJSONPointer(file.API.AccessURL),
		}
		splitFiles = append(splitFiles, map[string]any{
			"file":         file.File,
			"operationId":  file.Config.OperationID,
			"group":        file.Config.Group,
			"serviceGroup": file.Config.ServiceGroup,
			"goName":       file.Config.GoName,
			"rawFunction":  file.Config.RawFunction,
			"groupMethod":  file.Config.GroupMethod,
		})
	}
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "KIS Domestic Stock OpenAPI split index",
			"version": "0.1.0",
		},
		"paths":             paths,
		"x-kis-source":      source.Source,
		"x-kis-fetched-at":  source.FetchedAt,
		"x-kis-split-files": splitFiles,
	}
}

func bundle(source catalog, files []splitFile) map[string]any {
	paths := map[string]any{}
	schemas := map[string]any{}
	for _, file := range files {
		docPaths := file.Doc["paths"].(map[string]any)
		for path, item := range docPaths {
			paths[path] = item
		}
		docComponents := file.Doc["components"].(map[string]any)
		docSchemas := docComponents["schemas"].(map[string]any)
		for name, schema := range docSchemas {
			schemas[name] = schema
		}
	}
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "KIS Domestic Stock OpenAPI bundle",
			"version": "0.1.0",
		},
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": securitySchemes(),
			"schemas":         schemas,
		},
		"x-kis-source":     source.Source,
		"x-kis-fetched-at": source.FetchedAt,
	}
}

func queryParameters(api catalogAPI) []map[string]any {
	props := filterProperties(api.Properties, "req_b", "")
	params := make([]map[string]any, 0, len(props))
	for _, prop := range props {
		params = append(params, map[string]any{
			"name":        prop.PropertyCD,
			"in":          "query",
			"required":    prop.RequireYN == "Y",
			"description": cleanText(prop.Description),
			"schema":      propertySchema(prop),
			"x-kis":       propertyExtension(prop),
		})
	}
	return params
}

func responseSchema(api catalogAPI) map[string]any {
	roots := filterProperties(api.Properties, "res_b", "")
	properties := map[string]any{}
	required := []string{}
	for _, root := range roots {
		rootName := responseRootName(root.PropertyCD)
		children := childProperties(api.Properties, root)
		schema := propertySchema(root)
		switch root.PropertyType {
		case "A0003":
			schema = objectSchema(children)
		case "A0005":
			schema = map[string]any{
				"type":  "array",
				"items": objectSchema(children),
			}
		}
		properties[rootName] = schema
		if root.RequireYN == "Y" {
			required = append(required, rootName)
		}
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}

func objectSchema(props []catalogProperty) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, prop := range props {
		properties[prop.PropertyCD] = propertySchema(prop)
		if prop.RequireYN == "Y" {
			required = append(required, prop.PropertyCD)
		}
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
		"required":             required,
	}
}

func propertySchema(prop catalogProperty) map[string]any {
	return map[string]any{
		"type":        "string",
		"description": cleanText(firstNonEmpty(prop.PropertyName, prop.Description)),
		"x-kis":       propertyExtension(prop),
	}
}

func propertyExtension(prop catalogProperty) map[string]any {
	return map[string]any{
		"bodyType":       prop.BodyType,
		"propertyType":   prop.PropertyType,
		"propertyLength": strings.TrimSpace(prop.PropertyLength),
		"propertyOrder":  prop.PropertyOrder,
		"required":       prop.RequireYN == "Y",
		"name":           cleanText(prop.PropertyName),
		"description":    cleanText(prop.Description),
	}
}

func xKIS(source catalog, collection catalogCollection, api catalogAPI, config operationConfig) map[string]any {
	return map[string]any{
		"catalogSchemaVersion": source.SchemaVersion,
		"source":               source.Source,
		"sourceDetailUrl":      api.SourceDetailURL,
		"collectionId":         collection.ID,
		"collectionName":       collection.Name,
		"apiId":                api.ID,
		"apiName":              api.Name,
		"apiSummary":           firstNonEmpty(config.DocSummary, api.APISummary),
		"accessUrl":            api.AccessURL,
		"realDomain":           api.RealDomain,
		"virtualDomain":        api.VirtualDomain,
		"realTrId":             api.RealTRID,
		"virtualTrId":          api.VirtualTRID,
		"supportsReal":         api.RealTRID != "",
		"supportsVirtual":      supportsVirtual(api.VirtualTRID, api.VirtualDomain),
		"group":                config.Group,
		"serviceGroup":         config.ServiceGroup,
		"roleCandidate":        config.RoleCandidate,
		"securityType":         config.SecurityType,
		"oauth":                api.OAuth,
		"lastModifiedDate":     api.LastModifiedDate,
	}
}

func securitySchemes() map[string]any {
	return map[string]any{
		"KISOAuth": map[string]any{
			"type":         "http",
			"scheme":       "bearer",
			"bearerFormat": "KIS OAuth access token",
			"description":  "Issued by KIS /oauth2/tokenP and injected by the handwritten runtime.",
		},
	}
}

func writeRawAPI(out string, files []splitFile) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	if err := removeGeneratedFiles(out, func(name string) bool {
		return strings.HasSuffix(name, "_gen.go")
	}); err != nil {
		return err
	}
	if err := writeGo(filepath.Join(out, "runtime_gen.go"), generatedRuntime()); err != nil {
		return err
	}
	if err := writeGo(filepath.Join(out, "constants_gen.go"), generatedConstants(files)); err != nil {
		return err
	}
	for _, group := range groupedFiles(files) {
		name := snakeName(group.Group)
		if err := writeGo(filepath.Join(out, name+"_types_gen.go"), generatedGroupTypes(group.Files)); err != nil {
			return err
		}
		if err := writeGo(filepath.Join(out, name+"_api_gen.go"), generatedGroupAPI(group.Files)); err != nil {
			return err
		}
	}
	return nil
}

func generatedRuntime() string {
	return `// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.

package rawapi

import (
	"context"
	"errors"
)

var ErrRuntimeRequired = errors.New("kis rawapi runtime is required")

type Executor interface {
	ExecuteKIS(context.Context, Request, any) error
}

type Request struct {
	Group        string
	Operation    string
	Method       string
	Path         string
	RealTRID     string
	VirtualTRID  string
	Query        map[string]string
}

type Status struct {
	RTCD  string
	MsgCD string
	Msg1  string
}

type StatusProvider interface {
	KISStatus() Status
}
`
}

func generatedConstants(files []splitFile) string {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.\n\n")
	buf.WriteString("package rawapi\n\n")
	buf.WriteString("type OperationMetadata struct {\n")
	buf.WriteString("\tOperationID string\n")
	buf.WriteString("\tEndpoint string\n")
	buf.WriteString("\tGroup string\n")
	buf.WriteString("\tServiceGroup string\n")
	buf.WriteString("\tRealTRID string\n")
	buf.WriteString("\tVirtualTRID string\n")
	buf.WriteString("\tSupportsVirtual bool\n")
	buf.WriteString("}\n\n")
	buf.WriteString("const (\n")
	for _, file := range files {
		suffix := file.Config.RawFunction
		fmt.Fprintf(&buf, "\tOperation%s = %q\n", suffix, file.Config.OperationID)
		fmt.Fprintf(&buf, "\tEndpoint%s = %q\n", suffix, file.API.AccessURL)
		fmt.Fprintf(&buf, "\tGroup%s = %q\n", suffix, file.Config.Group)
		fmt.Fprintf(&buf, "\tServiceGroup%s = %q\n", suffix, file.Config.ServiceGroup)
		fmt.Fprintf(&buf, "\tRealTRID%s = %q\n", suffix, file.API.RealTRID)
		fmt.Fprintf(&buf, "\tVirtualTRID%s = %q\n", suffix, file.API.VirtualTRID)
		fmt.Fprintf(&buf, "\tSupportsVirtual%s = %t\n", suffix, supportsVirtual(file.API.VirtualTRID, file.API.VirtualDomain))
	}
	buf.WriteString(")\n\n")
	buf.WriteString("var operationMetadata = map[string]OperationMetadata{\n")
	for _, file := range files {
		suffix := file.Config.RawFunction
		fmt.Fprintf(&buf, "\tOperation%s: {\n", suffix)
		fmt.Fprintf(&buf, "\t\tOperationID: Operation%s,\n", suffix)
		fmt.Fprintf(&buf, "\t\tEndpoint: Endpoint%s,\n", suffix)
		fmt.Fprintf(&buf, "\t\tGroup: Group%s,\n", suffix)
		fmt.Fprintf(&buf, "\t\tServiceGroup: ServiceGroup%s,\n", suffix)
		fmt.Fprintf(&buf, "\t\tRealTRID: RealTRID%s,\n", suffix)
		fmt.Fprintf(&buf, "\t\tVirtualTRID: VirtualTRID%s,\n", suffix)
		fmt.Fprintf(&buf, "\t\tSupportsVirtual: SupportsVirtual%s,\n", suffix)
		buf.WriteString("\t},\n")
	}
	buf.WriteString("}\n\n")
	buf.WriteString("func LookupOperationMetadata(operationID string) (OperationMetadata, bool) {\n")
	buf.WriteString("\tmetadata, ok := operationMetadata[operationID]\n")
	buf.WriteString("\treturn metadata, ok\n")
	buf.WriteString("}\n")
	return buf.String()
}

func generatedGroupTypes(files []splitFile) string {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.\n\n")
	buf.WriteString("package rawapi\n\n")
	for _, file := range files {
		writeOperationTypes(&buf, file)
	}
	return buf.String()
}

func generatedGroupAPI(files []splitFile) string {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.\n\n")
	buf.WriteString("package rawapi\n\n")
	buf.WriteString("import \"context\"\n\n")
	for _, file := range files {
		writeOperationFunction(&buf, file)
	}
	return buf.String()
}

func writeOperationTypes(buf *bytes.Buffer, file splitFile) {
	config := file.Config
	api := file.API
	queryProps := filterProperties(api.Properties, "req_b", "")
	responseRoots := filterProperties(api.Properties, "res_b", "")

	writeOperationDoc(buf, config.GoName+"Request", "is the request for the KIS API.", file)
	fmt.Fprintf(buf, "type %sRequest struct {\n", config.GoName)
	used := map[string]int{}
	for _, prop := range queryProps {
		fieldName := uniqueName(goName(prop.PropertyCD), used)
		writeFieldDoc(buf, fieldName, prop)
		fmt.Fprintf(buf, "\t%s string `json:\"%s\"`\n", fieldName, prop.PropertyCD)
	}
	buf.WriteString("}\n\n")

	fmt.Fprintf(buf, "func (r %sRequest) query() map[string]string {\n", config.GoName)
	buf.WriteString("\treturn map[string]string{\n")
	used = map[string]int{}
	for _, prop := range queryProps {
		fmt.Fprintf(buf, "\t\t%q: r.%s,\n", prop.PropertyCD, uniqueName(goName(prop.PropertyCD), used))
	}
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	writeOperationDoc(buf, config.GoName+"Response", "is the response for the KIS API.", file)
	fmt.Fprintf(buf, "type %sResponse struct {\n", config.GoName)
	used = map[string]int{}
	nested := []nestedStruct{}
	for _, root := range responseRoots {
		fieldName := uniqueName(goName(root.PropertyCD), used)
		typeName := "string"
		children := childProperties(api.Properties, root)
		switch root.PropertyType {
		case "A0003":
			typeName = config.GoName + goName(root.PropertyCD)
			nested = append(nested, nestedStruct{Name: typeName, Props: children})
		case "A0005":
			typeName = "[]" + config.GoName + goName(root.PropertyCD) + "Item"
			nested = append(nested, nestedStruct{Name: config.GoName + goName(root.PropertyCD) + "Item", Props: children})
		}
		writeFieldDoc(buf, fieldName, root)
		fmt.Fprintf(buf, "\t%s %s `json:\"%s\"`\n", fieldName, typeName, responseRootName(root.PropertyCD))
	}
	buf.WriteString("}\n\n")
	for _, item := range nested {
		writeStruct(buf, item.Name, item.Props)
	}

	fmt.Fprintf(buf, "func (r *%sResponse) KISStatus() Status {\n", config.GoName)
	buf.WriteString("\tif r == nil {\n\t\treturn Status{}\n\t}\n")
	buf.WriteString("\treturn Status{RTCD: r.RtCd, MsgCD: r.MsgCd, Msg1: r.Msg1}\n")
	buf.WriteString("}\n\n")
}

func writeOperationFunction(buf *bytes.Buffer, file splitFile) {
	config := file.Config
	writeOperationDoc(buf, config.RawFunction, "calls the KIS raw API.", file)
	fmt.Fprintf(buf, "func %s(ctx context.Context, executor Executor, input %sRequest) (%sResponse, error) {\n", config.RawFunction, config.GoName, config.GoName)
	buf.WriteString("\tif executor == nil {\n")
	fmt.Fprintf(buf, "\t\treturn %sResponse{}, ErrRuntimeRequired\n", config.GoName)
	buf.WriteString("\t}\n")
	fmt.Fprintf(buf, "\tvar result %sResponse\n", config.GoName)
	buf.WriteString("\terr := executor.ExecuteKIS(ctx, Request{\n")
	fmt.Fprintf(buf, "\t\tGroup: Group%s,\n", config.RawFunction)
	fmt.Fprintf(buf, "\t\tOperation: Operation%s,\n", config.RawFunction)
	fmt.Fprintf(buf, "\t\tMethod: %q,\n", file.API.HTTPMethod)
	fmt.Fprintf(buf, "\t\tPath: Endpoint%s,\n", config.RawFunction)
	fmt.Fprintf(buf, "\t\tRealTRID: RealTRID%s,\n", config.RawFunction)
	fmt.Fprintf(buf, "\t\tVirtualTRID: VirtualTRID%s,\n", config.RawFunction)
	buf.WriteString("\t\tQuery: input.query(),\n")
	buf.WriteString("\t}, &result)\n")
	buf.WriteString("\treturn result, err\n")
	buf.WriteString("}\n\n")
}

func writeStruct(buf *bytes.Buffer, name string, props []catalogProperty) {
	writeDoc(buf, name, "is a KIS response object.", nil)
	fmt.Fprintf(buf, "type %s struct {\n", name)
	used := map[string]int{}
	for _, prop := range props {
		fieldName := uniqueName(goName(prop.PropertyCD), used)
		writeFieldDoc(buf, fieldName, prop)
		fmt.Fprintf(buf, "\t%s string `json:\"%s\"`\n", fieldName, prop.PropertyCD)
	}
	buf.WriteString("}\n\n")
}

func writePublicPackage(out string, files []splitFile) error {
	if err := removeGeneratedFiles(out, func(name string) bool {
		return (strings.HasPrefix(name, "aliases_") && strings.HasSuffix(name, "_gen.go")) ||
			(strings.HasPrefix(name, "service_") && strings.HasSuffix(name, ".go"))
	}); err != nil {
		return err
	}
	for _, group := range groupedFiles(files) {
		name := snakeName(group.Group)
		if err := writeGo(filepath.Join(out, "aliases_"+name+"_gen.go"), generatedAliases(group.Files)); err != nil {
			return err
		}
		if err := writeGo(filepath.Join(out, "service_"+name+".go"), generatedService(group)); err != nil {
			return err
		}
	}
	return nil
}

func removeGeneratedFiles(dir string, match func(string) bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !match(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(data, []byte("// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.")) {
			continue
		}
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

func generatedAliases(files []splitFile) string {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.\n\n")
	buf.WriteString("package kis\n\n")
	buf.WriteString("import \"github.com/ev3rlit/mwosa/clients/kis/internal/generated/rawapi\"\n\n")
	for _, file := range files {
		for _, typeName := range operationTypeNames(file) {
			fmt.Fprintf(&buf, "// %s aliases rawapi.%s.\n", typeName, typeName)
			fmt.Fprintf(&buf, "type %s = rawapi.%s\n\n", typeName, typeName)
		}
	}
	return buf.String()
}

func generatedService(group fileGroup) string {
	var buf bytes.Buffer
	buf.WriteString("// Code generated by clients/kis/internal/codegen/kisopenapi; DO NOT EDIT.\n\n")
	buf.WriteString("package kis\n\n")
	buf.WriteString("import (\n")
	buf.WriteString("\t\"context\"\n\n")
	buf.WriteString("\t\"github.com/ev3rlit/mwosa/clients/kis/internal/generated/rawapi\"\n")
	buf.WriteString(")\n\n")
	serviceName := group.ServiceGroup + "Service"
	writeDoc(&buf, serviceName, "calls KIS APIs in the "+group.Group+" group.", nil)
	fmt.Fprintf(&buf, "type %s struct {\n", serviceName)
	buf.WriteString("\texecutor rawapi.Executor\n")
	buf.WriteString("}\n\n")
	writeDoc(&buf, group.ServiceGroup, "returns the "+group.ServiceGroup+" service.", nil)
	fmt.Fprintf(&buf, "func (c *Client) %s() %s {\n", group.ServiceGroup, serviceName)
	fmt.Fprintf(&buf, "\treturn %s{executor: c.rawAPIExecutor()}\n", serviceName)
	buf.WriteString("}\n\n")
	for _, file := range group.Files {
		config := file.Config
		writeServiceMethodDoc(&buf, file)
		fmt.Fprintf(&buf, "func (s %s) %s(ctx context.Context, input %sRequest) (%sResponse, error) {\n", serviceName, config.GroupMethod, config.GoName, config.GoName)
		fmt.Fprintf(&buf, "\treturn rawapi.%s(ctx, s.executor, input)\n", config.RawFunction)
		buf.WriteString("}\n\n")
	}
	return buf.String()
}

type fileGroup struct {
	Group        string
	ServiceGroup string
	Files        []splitFile
}

func groupedFiles(files []splitFile) []fileGroup {
	order := []string{}
	byGroup := map[string]*fileGroup{}
	for _, file := range files {
		group, ok := byGroup[file.Config.Group]
		if !ok {
			order = append(order, file.Config.Group)
			group = &fileGroup{Group: file.Config.Group, ServiceGroup: file.Config.ServiceGroup}
			byGroup[file.Config.Group] = group
		}
		group.Files = append(group.Files, file)
	}
	out := make([]fileGroup, 0, len(order))
	for _, name := range order {
		out = append(out, *byGroup[name])
	}
	return out
}

func operationTypeNames(file splitFile) []string {
	names := []string{file.Config.GoName + "Request", file.Config.GoName + "Response"}
	for _, root := range filterProperties(file.API.Properties, "res_b", "") {
		switch root.PropertyType {
		case "A0003":
			names = append(names, file.Config.GoName+goName(root.PropertyCD))
		case "A0005":
			names = append(names, file.Config.GoName+goName(root.PropertyCD)+"Item")
		}
	}
	return names
}

func writeOperationDoc(buf *bytes.Buffer, name string, firstLine string, file splitFile) {
	config := file.Config
	api := file.API
	lines := []string{
		fmt.Sprintf("KIS API: %s", cleanText(api.Name)),
		fmt.Sprintf("Summary: %s", cleanText(firstNonEmpty(config.DocSummary, api.APISummary, api.Description))),
		fmt.Sprintf("Operation ID: %s", config.OperationID),
		fmt.Sprintf("Endpoint: %s %s", api.HTTPMethod, api.AccessURL),
		fmt.Sprintf("TR ID: %s", api.RealTRID),
		fmt.Sprintf("Virtual TR ID: %s", api.VirtualTRID),
		fmt.Sprintf("Group: %s", config.Group),
		fmt.Sprintf("Service group: %s", config.ServiceGroup),
		fmt.Sprintf("Role hint: %s", config.RoleCandidate),
	}
	writeDoc(buf, name, firstLine, lines)
}

func writeServiceMethodDoc(buf *bytes.Buffer, file splitFile) {
	config := file.Config
	api := file.API
	lines := []string{
		fmt.Sprintf("KIS API: %s", cleanText(api.Name)),
		fmt.Sprintf("Summary: %s", cleanText(firstNonEmpty(config.DocSummary, api.APISummary, api.Description))),
		fmt.Sprintf("Operation ID: %s", config.OperationID),
		fmt.Sprintf("Endpoint: %s %s", api.HTTPMethod, api.AccessURL),
		fmt.Sprintf("TR ID: %s", api.RealTRID),
		fmt.Sprintf("Virtual TR ID: %s", api.VirtualTRID),
		fmt.Sprintf("Role hint: %s", config.RoleCandidate),
	}
	writeDoc(buf, config.GroupMethod, "calls the KIS "+cleanText(api.Name)+" API.", lines)
}

func writeFieldDoc(buf *bytes.Buffer, fieldName string, prop catalogProperty) {
	lines := []string{
		fmt.Sprintf("KIS field: %s", cleanText(firstNonEmpty(prop.PropertyName, prop.Description))),
		fmt.Sprintf("Property code: %s", prop.PropertyCD),
		fmt.Sprintf("Required: %t", prop.RequireYN == "Y"),
		fmt.Sprintf("Length: %s", strings.TrimSpace(prop.PropertyLength)),
		fmt.Sprintf("Order: %s", prop.PropertyOrder),
	}
	if description := cleanText(prop.Description); description != "" {
		lines = append(lines, "Description: "+description)
	}
	writeDoc(buf, fieldName, "maps "+prop.PropertyCD+".", lines)
}

func writeDoc(buf *bytes.Buffer, name string, firstLine string, details []string) {
	fmt.Fprintf(buf, "// %s %s\n", name, firstLine)
	if len(details) > 0 {
		buf.WriteString("//\n")
		for _, detail := range details {
			detail = cleanComment(detail)
			if detail == "" {
				continue
			}
			fmt.Fprintf(buf, "// %s\n", detail)
		}
	}
}

func cleanComment(s string) string {
	s = cleanText(s)
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func writeGo(path string, source string) error {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return fmt.Errorf("format %s: %w\n%s", path, err, source)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, formatted, 0o644)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func filterProperties(props []catalogProperty, bodyType string, orderPrefix string) []catalogProperty {
	out := []catalogProperty{}
	for _, prop := range props {
		if prop.BodyType != bodyType {
			continue
		}
		if orderPrefix == "" {
			if strings.Contains(prop.PropertyOrder, ".") {
				continue
			}
		} else if !strings.HasPrefix(prop.PropertyOrder, orderPrefix+".") {
			continue
		}
		out = append(out, prop)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return naturalOrder(out[i].PropertyOrder) < naturalOrder(out[j].PropertyOrder)
	})
	return out
}

func childProperties(props []catalogProperty, root catalogProperty) []catalogProperty {
	return filterProperties(props, root.BodyType, root.PropertyOrder)
}

func naturalOrder(order string) string {
	parts := strings.Split(order, ".")
	for i, part := range parts {
		parts[i] = fmt.Sprintf("%06s", part)
	}
	return strings.Join(parts, ".")
}

func operationIDFromAccessURL(accessURL string) string {
	parts := strings.Split(strings.Trim(accessURL, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func operationIDFromAccessURLWithNamespace(accessURL string) string {
	parts := strings.Split(strings.Trim(accessURL, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	namespace := parts[0]
	if namespace == "uapi" && len(parts) > 1 {
		namespace = parts[1]
	}
	parent := ""
	if len(parts) > 1 {
		parent = parts[len(parts)-2]
	}
	return strings.Join(nonEmptyStrings(namespace, parent, parts[len(parts)-1]), "-")
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func defaultSplitPath(accessURL string) string {
	path := strings.TrimPrefix(strings.Trim(accessURL, "/"), "uapi/")
	return "apis/" + path + ".json"
}

func exclusionReason(collection catalogCollection, api catalogAPI) string {
	collectionName := cleanText(collection.Name)
	accessURL := strings.ToLower(api.AccessURL)
	if strings.Contains(collectionName, "실시간시세") || strings.Contains(accessURL, "websocket") || strings.Contains(accessURL, "websocket") {
		return "WebSocket/real-time API is out of scope"
	}
	if strings.Contains(collectionName, "주문/계좌") || strings.Contains(accessURL, "/trading/") {
		return "order/account API is out of scope"
	}
	if strings.EqualFold(api.HTTPMethod, "POST") {
		return "non-read API method is out of scope"
	}
	return ""
}

func defaultServiceGroup(collectionName string) string {
	switch collectionName {
	case "[국내주식] 기본시세":
		return "quote"
	case "[국내주식] 순위분석":
		return "ranking"
	case "[국내주식] 종목정보":
		return "instrument"
	case "[국내주식] 시세분석":
		return "analysis"
	case "[국내주식] 업종/기타":
		return "market"
	case "[국내주식] ELW 시세":
		return "elw-quote"
	default:
		return ""
	}
}

func defaultRoleHint(collection catalogCollection, api catalogAPI) string {
	switch collection.Name {
	case "[국내주식] 순위분석", "[국내주식] 시세분석":
		return "market_scan"
	case "[국내주식] 종목정보":
		return "instrument"
	}
	operationID := operationIDFromAccessURL(api.AccessURL)
	switch {
	case strings.Contains(operationID, "chartprice"), strings.Contains(operationID, "daily-price"):
		return "daily_bar"
	case strings.Contains(operationID, "time"):
		return "intraday_bar"
	case strings.Contains(operationID, "asking-price"):
		return "orderbook"
	case strings.Contains(operationID, "ccnl"), strings.Contains(operationID, "conclusion"):
		return "trades"
	default:
		return "read_only"
	}
}

func goName(s string) string {
	parts := regexp.MustCompile(`[^A-Za-z0-9]+`).Split(strings.TrimSpace(s), -1)
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)
		switch lower {
		case "etf":
			b.WriteString("ETF")
		case "etn":
			b.WriteString("ETN")
		case "etfetn":
			b.WriteString("ETFETN")
		case "elw":
			b.WriteString("ELW")
		case "id":
			b.WriteString("ID")
		case "iscd":
			b.WriteString("ISCD")
		case "tr":
			b.WriteString("TR")
		case "rt":
			b.WriteString("Rt")
		case "cd":
			b.WriteString("Cd")
		default:
			runes := []rune(lower)
			if len(runes) == 0 {
				continue
			}
			b.WriteRune(unicode.ToUpper(runes[0]))
			b.WriteString(string(runes[1:]))
		}
	}
	if b.Len() == 0 {
		return "Value"
	}
	name := b.String()
	if unicode.IsDigit([]rune(name)[0]) {
		return "Value" + name
	}
	return name
}

func uniqueName(name string, used map[string]int) string {
	used[name]++
	if used[name] == 1 {
		return name
	}
	return fmt.Sprintf("%s%d", name, used[name])
}

func supportsVirtual(trID string, domain string) bool {
	unsupported := "모의투자 미지원"
	return strings.TrimSpace(trID) != "" && !strings.Contains(trID, unsupported) && !strings.Contains(domain, unsupported)
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.TrimSpace(s)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func escapeJSONPointer(path string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, "~", "~0"), "/", "~1")
}

func responseRootName(name string) string {
	if strings.EqualFold(name, "output") {
		return "output"
	}
	return name
}

func snakeName(s string) string {
	var b strings.Builder
	for i, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
				b.WriteByte('_')
			}
			continue
		}
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
