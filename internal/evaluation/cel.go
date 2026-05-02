package evaluation

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types/ref"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// CELEvaluator compiles and executes CEL expressions for custom evaluation
// metrics. It provides a sandboxed environment with access to three variables:
//
//   - output (map<string, dyn>) — the AgentRun output
//   - expected (map<string, dyn>) — the expected values from the sample
//   - sample (map<string, dyn>) — the full sample metadata
type CELEvaluator struct {
	env *cel.Env
}

// NewCELEvaluator creates a CEL evaluator with the standard variable
// declarations for evaluation expressions.
func NewCELEvaluator() (*CELEvaluator, error) {
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar("output", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("expected", decls.NewMapType(decls.String, decls.Dyn)),
			decls.NewVar("sample", decls.NewMapType(decls.String, decls.Dyn)),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create CEL environment: %w", err)
	}
	return &CELEvaluator{env: env}, nil
}

// Evaluate compiles and executes a CEL expression with the given bindings.
// Returns the result as a float64 score (0-1) and true, or (0, false) if
// the expression is invalid or returns a non-numeric result.
//
// Bool results are converted: true → 1.0, false → 0.0.
// Numeric results are returned directly.
// Any other type returns (0, false).
func (e *CELEvaluator) Evaluate(expression string, output, expected, sample map[string]interface{}) (float64, bool, error) {
	ast, issues := e.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return 0, false, fmt.Errorf("compile CEL expression: %w", issues.Err())
	}

	// Type-check the expression.
	checked, issues := e.env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return 0, false, fmt.Errorf("check CEL expression: %w", issues.Err())
	}

	// Ensure the expression returns a usable type (bool, int, uint, double).
	resultType := checked.ResultType()
	if resultType == nil {
		return 0, false, fmt.Errorf("CEL expression has no result type")
	}
	kind := resultType.GetPrimitive()
	switch kind {
	case exprpb.Type_BOOL, exprpb.Type_INT64, exprpb.Type_UINT64, exprpb.Type_DOUBLE:
		// OK
	default:
		return 0, false, fmt.Errorf("CEL expression must return bool or numeric, got %s", resultType.String())
	}

	program, err := e.env.Program(checked)
	if err != nil {
		return 0, false, fmt.Errorf("program CEL expression: %w", err)
	}

	vars := map[string]interface{}{
		"output":   output,
		"expected": expected,
		"sample":   sample,
	}

	result, _, err := program.Eval(vars)
	if err != nil {
		return 0, false, fmt.Errorf("evaluate CEL expression: %w", err)
	}

	return refValueToScore(result)
}

// refValueToScore converts a CEL ref.Value to a float64 score.
func refValueToScore(val ref.Val) (float64, bool, error) {
	switch v := val.Value().(type) {
	case bool:
		if v {
			return 1.0, true, nil
		}
		return 0.0, true, nil
	case int64:
		return float64(v), true, nil
	case uint64:
		return float64(v), true, nil
	case float64:
		return v, true, nil
	default:
		return 0, false, nil
	}
}

// CompileError returns a user-friendly error message for CEL compilation errors.
func CompileError(err error) string {
	msg := err.Error()
	// Trim common cel-go prefixes for readability.
	msg = strings.TrimPrefix(msg, "ERROR: <input>:")
	return msg
}
