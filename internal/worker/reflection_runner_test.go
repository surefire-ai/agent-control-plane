package worker

import (
	"strings"
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestIsReflectionPattern(t *testing.T) {
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
			name: "pattern.type reflection",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "reflection"},
			},
			want: true,
		},
		{
			name: "runner.pattern.type reflection",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "reflection"},
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
			name: "runner.pattern.type non-string",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": 123},
				},
			},
			want: false,
		},
		{
			name: "runner.pattern missing type",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"other": "value"},
				},
			},
			want: false,
		},
		{
			name: "both pattern.type and runner.pattern set",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "reflection"},
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "other"},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReflectionPattern(tt.artifact); got != tt.want {
				t.Errorf("isReflectionPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAugmentPromptForPhase(t *testing.T) {
	base := contract.PromptSpec{
		Name:     "system",
		Template: "You are a helpful assistant.",
		Language: "en",
	}

	tests := []struct {
		name           string
		phase          string
		previousOutput string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "generate phase",
			phase:          "generate",
			previousOutput: "",
			wantContains: []string{
				"You are a helpful assistant.",
				"Generate your best output",
			},
		},
		{
			name:           "critique phase",
			phase:          "critique",
			previousOutput: "some output to review",
			wantContains: []string{
				"You are a helpful assistant.",
				"Review the following output critically",
				"some output to review",
				"\"approved\"",
				"\"needs_revision\"",
			},
		},
		{
			name:           "revise phase",
			phase:          "revise",
			previousOutput: "current output with critique",
			wantContains: []string{
				"You are a helpful assistant.",
				"Revise the following output",
				"current output with critique",
			},
		},
		{
			name:           "unknown phase",
			phase:          "unknown",
			previousOutput: "",
			wantContains: []string{
				"You are a helpful assistant.",
			},
			wantNotContain: []string{
				"Generate your best output",
				"Review the following output",
				"Revise the following output",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := augmentPromptForPhase(base, tt.phase, tt.previousOutput)

			// Verify name and language are preserved.
			if got.Name != base.Name {
				t.Errorf("Name = %q, want %q", got.Name, base.Name)
			}
			if got.Language != base.Language {
				t.Errorf("Language = %q, want %q", got.Language, base.Language)
			}

			// Template should always start with the base template.
			if !strings.HasPrefix(got.Template, base.Template) {
				t.Errorf("Template does not start with base template")
			}

			for _, s := range tt.wantContains {
				if !strings.Contains(got.Template, s) {
					t.Errorf("Template missing expected substring %q", s)
				}
			}
			for _, s := range tt.wantNotContain {
				if strings.Contains(got.Template, s) {
					t.Errorf("Template should not contain %q", s)
				}
			}
		})
	}
}

func TestIsCritiquePositive(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "approved status",
			content: `{"status": "approved", "feedback": "No changes needed."}`,
			want:    true,
		},
		{
			name:    "needs_revision status",
			content: `{"status": "needs_revision", "feedback": "Fix the formatting."}`,
			want:    false,
		},
		{
			name:    "approved with extra whitespace",
			content: `{"status": "  approved  ", "feedback": "Looks good."}`,
			want:    true,
		},
		{
			name:    "invalid json",
			content: `not valid json`,
			want:    false,
		},
		{
			name:    "empty string",
			content: "",
			want:    false,
		},
		{
			name:    "empty object",
			content: `{}`,
			want:    false,
		},
		{
			name:    "status is empty string",
			content: `{"status": ""}`,
			want:    false,
		},
		{
			name:    "status is wrong case",
			content: `{"status": "Approved"}`,
			want:    false,
		},
		{
			name:    "status contains approved as substring",
			content: `{"status": "not approved"}`,
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCritiquePositive(tt.content); got != tt.want {
				t.Errorf("isCritiquePositive() = %v, want %v", got, tt.want)
			}
		})
	}
}
