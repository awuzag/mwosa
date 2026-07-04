package aggregate

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/samber/oops"
)

type TemplateContext struct {
	Params map[string]any
	Each   map[string]any
}

func ApplyParams(spec Spec, overrides []string) (map[string]any, error) {
	errb := oops.In("aggregate_params").With("name", spec.Name)
	values := make(map[string]any, len(spec.Params))
	for name, param := range spec.Params {
		if param.Default == nil {
			if param.Required {
				return nil, errb.With("param", name).New("required aggregate param has no value")
			}
			continue
		}
		value, err := parseParamValue(param.Type, param.Default)
		if err != nil {
			return nil, errb.With("param", name).Wrapf(err, "parse aggregate param default")
		}
		values[name] = value
	}
	for _, override := range overrides {
		key, raw, ok := strings.Cut(override, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, errb.With("override", override).New("aggregate param override must be key=value")
		}
		key = strings.TrimSpace(key)
		param, exists := spec.Params[key]
		if !exists {
			return nil, errb.With("param", key).Errorf("unknown aggregate param: %s", key)
		}
		value, err := parseParamValue(param.Type, raw)
		if err != nil {
			return nil, errb.With("param", key, "value", raw).Wrapf(err, "parse aggregate param")
		}
		values[key] = value
	}
	return values, nil
}

func parseParamValue(paramType ParamType, value any) (any, error) {
	switch paramType {
	case "", ParamString:
		return strings.TrimSpace(toString(value)), nil
	case ParamDate:
		text := strings.TrimSpace(toString(value))
		if _, err := time.Parse(time.DateOnly, text); err != nil {
			return nil, err
		}
		return text, nil
	case ParamInt:
		switch typed := value.(type) {
		case int:
			return int64(typed), nil
		case int64:
			return typed, nil
		case float64:
			return int64(typed), nil
		default:
			parsed, err := strconv.ParseInt(strings.TrimSpace(toString(value)), 10, 64)
			if err != nil {
				return nil, err
			}
			return parsed, nil
		}
	case ParamFloat:
		switch typed := value.(type) {
		case int:
			return float64(typed), nil
		case int64:
			return float64(typed), nil
		case float64:
			return typed, nil
		default:
			parsed, err := strconv.ParseFloat(strings.TrimSpace(toString(value)), 64)
			if err != nil {
				return nil, err
			}
			return parsed, nil
		}
	case ParamBool:
		switch typed := value.(type) {
		case bool:
			return typed, nil
		default:
			parsed, err := strconv.ParseBool(strings.TrimSpace(toString(value)))
			if err != nil {
				return nil, err
			}
			return parsed, nil
		}
	default:
		return nil, oops.In("aggregate_params").With("type", paramType).Errorf("unsupported aggregate param type: %s", paramType)
	}
}

func ResolveTemplateValue(value any, ctx TemplateContext) (any, error) {
	switch typed := value.(type) {
	case string:
		return resolveTemplateString(typed, ctx)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			resolved, err := ResolveTemplateValue(child, ctx)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(typed))
		for _, child := range typed {
			resolved, err := ResolveTemplateValue(child, ctx)
			if err != nil {
				return nil, err
			}
			out = append(out, resolved)
		}
		return out, nil
	default:
		rv := reflect.ValueOf(value)
		if rv.IsValid() && rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			out := map[string]any{}
			iter := rv.MapRange()
			for iter.Next() {
				resolved, err := ResolveTemplateValue(iter.Value().Interface(), ctx)
				if err != nil {
					return nil, err
				}
				out[iter.Key().String()] = resolved
			}
			return out, nil
		}
		return value, nil
	}
}

func resolveTemplateString(value string, ctx TemplateContext) (any, error) {
	matches := placeholderPattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return value, nil
	}
	if len(matches) == 1 && matches[0][0] == value {
		return lookupTemplateValue(matches[0][1], matches[0][2], ctx)
	}
	out := placeholderPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := placeholderPattern.FindStringSubmatch(match)
		resolved, err := lookupTemplateValue(parts[1], parts[2], ctx)
		if err != nil {
			return match
		}
		return toString(resolved)
	})
	if placeholderPattern.MatchString(out) {
		return nil, oops.In("aggregate_template").With("value", value).New("unresolved aggregate template placeholder")
	}
	return out, nil
}

func lookupTemplateValue(scope string, key string, ctx TemplateContext) (any, error) {
	var source map[string]any
	switch scope {
	case "params":
		source = ctx.Params
	case "each":
		source = ctx.Each
	default:
		return nil, oops.In("aggregate_template").With("scope", scope).New("unsupported aggregate template scope")
	}
	value, ok := source[key]
	if !ok {
		return nil, oops.In("aggregate_template").With("scope", scope, "key", key).New("unknown aggregate template value")
	}
	return value, nil
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(value)
	}
}
