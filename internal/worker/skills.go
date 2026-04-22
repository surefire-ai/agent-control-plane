package worker

import (
	"fmt"
	"strings"
)

type BuiltinSkillFunction func(state map[string]interface{}) map[string]interface{}

type BuiltinSkill struct {
	Name      string
	Functions map[string]BuiltinSkillFunction
}

var builtinSkills = map[string]BuiltinSkill{
	"ehs": {
		Name: "ehs",
		Functions: map[string]BuiltinSkillFunction{
			"score_risk_by_matrix": scoreRiskByMatrix,
		},
	},
}

func resolveBuiltinSkillFunction(implementation string) (string, string, BuiltinSkillFunction, error) {
	const prefix = "app.skills."
	if !strings.HasPrefix(implementation, prefix) {
		return "", "", nil, FailureReasonError{
			Reason:  "UnsupportedGraphNode",
			Message: fmt.Sprintf("function implementation %q is not supported yet", implementation),
		}
	}

	trimmed := strings.TrimPrefix(implementation, prefix)
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", nil, FailureReasonError{
			Reason:  "UnsupportedGraphNode",
			Message: fmt.Sprintf("function implementation %q is not a valid skill reference", implementation),
		}
	}

	skillName := parts[0]
	functionName := parts[1]
	skill, ok := builtinSkills[skillName]
	if !ok {
		return "", "", nil, FailureReasonError{
			Reason:  "UnsupportedGraphNode",
			Message: fmt.Sprintf("skill %q is not supported yet", skillName),
		}
	}
	fn, ok := skill.Functions[functionName]
	if !ok {
		return "", "", nil, FailureReasonError{
			Reason:  "UnsupportedGraphNode",
			Message: fmt.Sprintf("skill function %q on %q is not supported yet", functionName, skillName),
		}
	}
	return skill.Name, functionName, fn, nil
}
