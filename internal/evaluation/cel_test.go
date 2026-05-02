package evaluation

import (
	"testing"
)

func TestNewCELEvaluator(t *testing.T) {
	eval, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("NewCELEvaluator() error = %v", err)
	}
	if eval == nil {
		t.Fatal("NewCELEvaluator() returned nil")
	}
}

func TestCELEvaluateBool(t *testing.T) {
	eval, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("NewCELEvaluator() error = %v", err)
	}

	tests := []struct {
		name       string
		expression string
		output     map[string]interface{}
		expected   map[string]interface{}
		wantScore  float64
		wantOK     bool
	}{
		{
			name:       "true returns 1.0",
			expression: "true",
			output:     map[string]interface{}{},
			expected:   map[string]interface{}{},
			wantScore:  1.0,
			wantOK:     true,
		},
		{
			name:       "false returns 0.0",
			expression: "false",
			output:     map[string]interface{}{},
			expected:   map[string]interface{}{},
			wantScore:  0.0,
			wantOK:     true,
		},
		{
			name:       "field comparison true",
			expression: "output.riskLevel == 'high'",
			output:     map[string]interface{}{"riskLevel": "high"},
			expected:   map[string]interface{}{},
			wantScore:  1.0,
			wantOK:     true,
		},
		{
			name:       "field comparison false",
			expression: "output.riskLevel == 'low'",
			output:     map[string]interface{}{"riskLevel": "high"},
			expected:   map[string]interface{}{},
			wantScore:  0.0,
			wantOK:     true,
		},
		{
			name:       "size check",
			expression: "size(output.hazards) > 2",
			output:     map[string]interface{}{"hazards": []interface{}{"a", "b", "c"}},
			expected:   map[string]interface{}{},
			wantScore:  1.0,
			wantOK:     true,
		},
		{
			name:       "complex boolean",
			expression: "size(output.hazards) > 0 && output.riskLevel != 'low'",
			output:     map[string]interface{}{"hazards": []interface{}{"a"}, "riskLevel": "high"},
			expected:   map[string]interface{}{},
			wantScore:  1.0,
			wantOK:     true,
		},
		{
			name:       "expected variable access",
			expression: "output.level == expected.level",
			output:     map[string]interface{}{"level": "high"},
			expected:   map[string]interface{}{"level": "high"},
			wantScore:  1.0,
			wantOK:     true,
		},
		{
			name:       "has field check",
			expression: "has(output.confidence)",
			output:     map[string]interface{}{"confidence": 0.9},
			expected:   map[string]interface{}{},
			wantScore:  1.0,
			wantOK:     true,
		},
		{
			name:       "missing field via has",
			expression: "has(output.missing)",
			output:     map[string]interface{}{},
			expected:   map[string]interface{}{},
			wantScore:  0.0,
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, ok, err := eval.Evaluate(tt.expression, tt.output, tt.expected, nil)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("Evaluate() ok = %v, want %v", ok, tt.wantOK)
			}
			if score != tt.wantScore {
				t.Fatalf("Evaluate() score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}

func TestCELEvaluateNumeric(t *testing.T) {
	eval, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("NewCELEvaluator() error = %v", err)
	}

	tests := []struct {
		name       string
		expression string
		output     map[string]interface{}
		wantScore  float64
		wantOK     bool
	}{
		{
			name:       "integer result",
			expression: "size(output.hazards)",
			output:     map[string]interface{}{"hazards": []interface{}{"a", "b", "c"}},
			wantScore:  3.0,
			wantOK:     true,
		},
		{
			name:       "arithmetic",
			expression: "output.score + 1",
			output:     map[string]interface{}{"score": int64(4)},
			wantScore:  5.0,
			wantOK:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, ok, err := eval.Evaluate(tt.expression, tt.output, nil, nil)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("Evaluate() ok = %v, want %v", ok, tt.wantOK)
			}
			if score != tt.wantScore {
				t.Fatalf("Evaluate() score = %v, want %v", score, tt.wantScore)
			}
		})
	}
}

func TestCELEvaluateErrors(t *testing.T) {
	eval, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("NewCELEvaluator() error = %v", err)
	}

	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "invalid syntax",
			expression: "output.???",
			wantErr:    true,
		},
		{
			name:       "string result not allowed",
			expression: "output.name",
			wantErr:    true,
		},
		{
			name:       "empty expression",
			expression: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := eval.Evaluate(tt.expression, map[string]interface{}{"name": "test"}, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompileError(t *testing.T) {
	eval, err := NewCELEvaluator()
	if err != nil {
		t.Fatalf("NewCELEvaluator() error = %v", err)
	}

	_, _, err = eval.Evaluate("output.???", map[string]interface{}{}, nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}

	msg := CompileError(err)
	if msg == "" {
		t.Fatal("CompileError returned empty string")
	}
	t.Logf("CompileError: %s", msg)
}
