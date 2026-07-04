package aggregate

import (
	"strings"

	"github.com/samber/oops"
)

var allowedMongoStages = map[string]struct{}{
	"$lookup":          {},
	"$group":           {},
	"$project":         {},
	"$addFields":       {},
	"$set":             {},
	"$unwind":          {},
	"$sort":            {},
	"$limit":           {},
	"$setWindowFields": {},
	"$match":           {},
}

var blockedMongoStages = map[string]struct{}{
	"$out":          {},
	"$merge":        {},
	"$function":     {},
	"$accumulator":  {},
	"$where":        {},
	"$currentOp":    {},
	"$listSessions": {},
}

func ValidateMongoPipeline(pipeline []map[string]any, aliases map[string]string) error {
	errb := oops.In("aggregate_mongodb_policy")
	for index, stage := range pipeline {
		if len(stage) != 1 {
			return errb.With("stage_index", index).New("mongodb aggregation stage must contain exactly one operator")
		}
		for operator, body := range stage {
			if _, blocked := blockedMongoStages[operator]; blocked {
				return errb.With("stage_index", index, "operator", operator).Errorf("blocked mongodb aggregation stage: %s", operator)
			}
			if _, allowed := allowedMongoStages[operator]; !allowed {
				return errb.With("stage_index", index, "operator", operator).Errorf("unsupported mongodb aggregation stage: %s", operator)
			}
			if err := validateMongoStageBody(index, operator, body, aliases); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateMongoStageBody(index int, operator string, body any, aliases map[string]string) error {
	if operator == "$lookup" {
		lookup, ok := body.(map[string]any)
		if !ok {
			return nil
		}
		from, ok := lookup["from"].(string)
		if !ok || strings.TrimSpace(from) == "" {
			return nil
		}
		if len(aliases) > 0 {
			if _, ok := aliases[from]; !ok {
				return oops.In("aggregate_mongodb_policy").With("stage_index", index, "from", from).Errorf("unknown lookup source: %s", from)
			}
		}
	}
	if containsBlockedMongoOperator(body) {
		return oops.In("aggregate_mongodb_policy").With("stage_index", index, "operator", operator).New("blocked mongodb expression operator")
	}
	return nil
}

func containsBlockedMongoOperator(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, blocked := blockedMongoStages[key]; blocked {
				return true
			}
			if containsBlockedMongoOperator(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsBlockedMongoOperator(child) {
				return true
			}
		}
	}
	return false
}
