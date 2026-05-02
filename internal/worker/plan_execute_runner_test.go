package worker

import (
	"strings"
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestIsPlanExecutePattern(t *testing.T) {
	tests := []struct {
		name     string
		artifact contract.CompiledArtifact
		want     bool
	}{
		{
			name:     "empty artifact",
			artifact: contract.CompiledArtifact{},
			want:     false,
		},
		{
			name: "pattern.type plan_execute",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "plan_execute"},
			},
			want: true,
		},
		{
			name: "runner.pattern.type plan_execute",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "plan_execute"},
				},
			},
			want: true,
		},
		{
			name: "pattern.type react",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "react"},
			},
			want: false,
		},
		{
			name: "runner.pattern.type workflow",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "workflow"},
				},
			},
			want: false,
		},
		{
			name: "runner.pattern.type non-string",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": 42},
				},
			},
			want: false,
		},
		{
			name: "pattern.type takes precedence over runner",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "plan_execute"},
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "react"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlanExecutePattern(tt.artifact); got != tt.want {
				t.Errorf("isPlanExecutePattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAugmentPlanPrompt(t *testing.T) {
	base := contract.PromptSpec{
		Name:     "system",
		Template: "You are an assistant.",
		Language: "en",
	}

	tests := []struct {
		name            string
		base            contract.PromptSpec
		phase           string
		maxSteps        int32
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:     "plan phase",
			base:     base,
			phase:    "plan",
			maxSteps: 5,
			wantContains: []string{
				"You are an assistant.",
				"PLANNING phase",
				"JSON array",
				"max 5",
			},
		},
		{
			name:     "execute phase",
			base:     base,
			phase:    "execute",
			maxSteps: 3,
			wantContains: []string{
				"You are an assistant.",
				"EXECUTION phase",
			},
		},
		{
			name:     "unknown phase is still augmented",
			base:     base,
			phase:    "unknown",
			maxSteps: 5,
			wantContains: []string{
				"You are an assistant.",
			},
			wantNotContains: []string{
				"PLANNING phase",
				"EXECUTION phase",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := augmentPlanPrompt(tt.base, tt.phase, tt.maxSteps)
			if got.Name != tt.base.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.base.Name)
			}
			if got.Language != tt.base.Language {
				t.Errorf("Language = %q, want %q", got.Language, tt.base.Language)
			}
			for _, s := range tt.wantContains {
				if !strings.Contains(got.Template, s) {
					t.Errorf("template missing expected substring %q", s)
				}
			}
			for _, s := range tt.wantNotContains {
				if strings.Contains(got.Template, s) {
					t.Errorf("template should not contain %q", s)
				}
			}
			// Template should always be longer than the base.
			if len(got.Template) <= len(base.Template) {
				t.Error("expected augmented prompt to be longer than base")
			}
		})
	}
}

func TestParsePlanSteps(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		parsed    map[string]interface{}
		wantLen   int
		wantErr   bool
		wantFirst planStep
	}{
		{
			name:    "from parsed steps field",
			content: "ignored",
			parsed: map[string]interface{}{
				"steps": []interface{}{
					map[string]interface{}{"id": "step-1", "description": "Analyze", "action": "extract"},
					map[string]interface{}{"id": "step-2", "description": "Summarize", "action": "produce"},
				},
			},
			wantLen: 2,
			wantFirst: planStep{
				ID:          "step-1",
				Description: "Analyze",
				Action:      "extract",
			},
		},
		{
			name:    "direct JSON array",
			content: `[{"id":"step-1","description":"Analyze input","action":"extract facts"},{"id":"step-2","description":"Generate output","action":"produce answer"}]`,
			parsed:  nil,
			wantLen: 2,
			wantFirst: planStep{
				ID:          "step-1",
				Description: "Analyze input",
				Action:      "extract facts",
			},
		},
		{
			name: "JSON array extracted from surrounding text",
			content: `Here is the plan you requested:

[{"id":"step-1","description":"Analyze the data","action":"run analysis"},{"id":"step-2","description":"Generate report","action":"write report"}]

Let me know if this helps!`,
			parsed:  nil,
			wantLen: 2,
			wantFirst: planStep{
				ID:          "step-1",
				Description: "Analyze the data",
				Action:      "run analysis",
			},
		},
		{
			name:    "JSON array in markdown code block not parsed (no brackets in plain text path)",
			content: "Some text\n```json\n[{\"id\":\"s1\",\"description\":\"step one\"}]\n```\nmore text",
			parsed:  nil,
			wantLen: 1,
			wantFirst: planStep{
				ID:          "s1",
				Description: "step one",
			},
		},
		{
			name:    "single step array",
			content: `[{"id":"only","description":"Do everything","action":"comprehensive"}]`,
			parsed:  nil,
			wantLen: 1,
			wantFirst: planStep{
				ID:          "only",
				Description: "Do everything",
				Action:      "comprehensive",
			},
		},
		{
			name:    "parsed steps skips non-map entries",
			content: "ignored",
			parsed: map[string]interface{}{
				"steps": []interface{}{
					"not a map",
					map[string]interface{}{"id": "s1", "description": "valid step"},
				},
			},
			wantLen: 1,
			wantFirst: planStep{
				ID:          "s1",
				Description: "valid step",
			},
		},
		{
			name:    "parsed steps skips entries with empty id and description",
			content: "ignored",
			parsed: map[string]interface{}{
				"steps": []interface{}{
					map[string]interface{}{"id": "", "description": "", "action": "noop"},
				},
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "unparseable content returns error",
			content: "this is not json at all",
			parsed:  nil,
			wantLen: 0,
			wantErr: true,
		},
		{
			name:    "empty array returns nil steps no error",
			content: "[]",
			parsed:  nil,
			wantLen: 0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps, err := parsePlanSteps(tt.content, tt.parsed)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(steps) != tt.wantLen {
				t.Fatalf("len(steps) = %d, want %d", len(steps), tt.wantLen)
			}
			if tt.wantLen > 0 && steps[0] != tt.wantFirst {
				t.Errorf("steps[0] = %+v, want %+v", steps[0], tt.wantFirst)
			}
		})
	}
}

func TestBuildStepInput(t *testing.T) {
	tests := []struct {
		name            string
		originalInput   map[string]interface{}
		step            planStep
		previousResults []planStepResult
		wantKeys        []string
		wantStepID      string
		wantPrevCount   int
	}{
		{
			name:          "first step with no previous results",
			originalInput: map[string]interface{}{"task": "analyze", "payload": "data"},
			step: planStep{
				ID:          "step-1",
				Description: "Analyze the data",
				Action:      "extract",
			},
			previousResults: nil,
			wantKeys:        []string{"task", "payload", "_plan_step"},
			wantStepID:      "step-1",
			wantPrevCount:   0,
		},
		{
			name:          "second step with one previous result",
			originalInput: map[string]interface{}{"task": "analyze"},
			step: planStep{
				ID:          "step-2",
				Description: "Generate report",
				Action:      "summarize",
			},
			previousResults: []planStepResult{
				{
					Step:    planStep{ID: "step-1", Description: "Analyze"},
					Output:  "analysis result",
					Success: true,
				},
			},
			wantKeys:      []string{"task", "_plan_step", "_previous_steps"},
			wantStepID:    "step-2",
			wantPrevCount: 1,
		},
		{
			name:          "empty original input",
			originalInput: map[string]interface{}{},
			step: planStep{
				ID:          "step-1",
				Description: "Do something",
			},
			previousResults: nil,
			wantKeys:        []string{"_plan_step"},
			wantStepID:      "step-1",
			wantPrevCount:   0,
		},
		{
			name:          "multiple previous results",
			originalInput: map[string]interface{}{"task": "multi"},
			step: planStep{
				ID:          "step-3",
				Description: "Final step",
			},
			previousResults: []planStepResult{
				{Step: planStep{ID: "step-1"}, Output: "out1", Success: true},
				{Step: planStep{ID: "step-2"}, Output: nil, Success: false, Error: "failed"},
			},
			wantKeys:      []string{"task", "_plan_step", "_previous_steps"},
			wantStepID:    "step-3",
			wantPrevCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStepInput(tt.originalInput, tt.step, tt.previousResults)

			// Check expected keys are present.
			for _, key := range tt.wantKeys {
				if _, ok := got[key]; !ok {
					t.Errorf("missing expected key %q", key)
				}
			}

			// Check _plan_step content.
			stepMap, ok := got["_plan_step"].(map[string]interface{})
			if !ok {
				t.Fatal("_plan_step is not a map")
			}
			if stepMap["id"] != tt.wantStepID {
				t.Errorf("_plan_step[id] = %v, want %q", stepMap["id"], tt.wantStepID)
			}

			// Check _previous_steps presence and length.
			if tt.wantPrevCount == 0 {
				if _, ok := got["_previous_steps"]; ok {
					t.Error("unexpected _previous_steps key for first step")
				}
			} else {
				prev, ok := got["_previous_steps"].([]map[string]interface{})
				if !ok {
					t.Fatalf("_previous_steps has wrong type: %T", got["_previous_steps"])
				}
				if len(prev) != tt.wantPrevCount {
					t.Errorf("len(_previous_steps) = %d, want %d", len(prev), tt.wantPrevCount)
				}
				// Verify first previous step has expected fields.
				if prev[0]["step_id"] == nil {
					t.Error("previous step missing step_id")
				}
			}

			// Verify original input keys are preserved.
			for k, v := range tt.originalInput {
				if got[k] != v {
					t.Errorf("originalInput[%q] = %v, want %v", k, got[k], v)
				}
			}
		})
	}
}
