package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestEinoADKRunnerExecutesSingleModel(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"{\"summary\":\"hazard identified\",\"hazards\":[{\"type\":\"electrical\",\"riskLevel\":\"high\"}]}"}}]}`))
	}))
	defer server.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
				"payload": map[string]interface{}{
					"text": "inspect line 3",
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Output["model"] != "planner" {
		t.Fatalf("expected model in output, got %#v", result.Output)
	}
	if result.Output["summary"] != "hazard identified" {
		t.Fatalf("expected summary in output, got %#v", result.Output)
	}
	if result.Runtime == nil || result.Runtime.Runner != "EinoADKRunner" {
		t.Fatalf("expected EinoADKRunner in runtime info, got %#v", result.Runtime)
	}
	if len(result.Artifacts) < 2 {
		t.Fatalf("expected chat completion artifacts, got %#v", result.Artifacts)
	}
}

func TestEinoADKRunnerDiagnosticWhenNoModel(t *testing.T) {
	runner := EinoADKRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Runner:  contract.ArtifactRunner{Kind: "EinoADKRunner"},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Output["validatedModels"] != 0 {
		t.Fatalf("expected validatedModels=0, got %#v", result.Output)
	}
	if result.Output["task"] != "identify_hazard" {
		t.Fatalf("expected task in output, got %#v", result.Output)
	}
}

func TestEinoADKRunnerDiagnosticWhenNoBaseURL(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	runner := EinoADKRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1"},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Output["validatedModels"] != 1 {
		t.Fatalf("expected validatedModels=1, got %#v", result.Output)
	}
}

func TestEinoADKRunnerExecutesGraphWithSingleNode(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"{\"summary\":\"inspection complete\",\"hazards\":[]}"}}]}`))
	}))
	defer server.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"name":     "classify_task",
							"kind":     "llm",
							"modelRef": "planner",
						},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Runtime == nil || result.Runtime.Runner != "EinoADKRunner" {
		t.Fatalf("expected EinoADKRunner in runtime info, got %#v", result.Runtime)
	}

	// Check that the graph node output is in the state.
	classifyOutput, ok := result.Output["classify_task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected classify_task output, got %#v", result.Output)
	}
	if classifyOutput["kind"] != "llm" {
		t.Fatalf("expected llm kind, got %#v", classifyOutput)
	}
	if classifyOutput["model"] != "planner" {
		t.Fatalf("expected planner model, got %#v", classifyOutput)
	}
}

func TestEinoADKRunnerExecutesGraphWithLinearChain(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	t.Setenv("MODEL_EXTRACTOR_API_KEY", "test-secret")

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var response string
		if callCount == 1 {
			response = `{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"{\"task_classified\":\"hazard_identification\"}"}}]}`
		} else {
			response = `{"id":"chatcmpl-2","choices":[{"message":{"role":"assistant","content":"{\"summary\":\"extraction complete\",\"facts\":[\"water near electrical\"]}"}}]}`
		}
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"name":     "classify_task",
							"kind":     "llm",
							"modelRef": "planner",
						},
						map[string]interface{}{
							"name":     "extract_facts",
							"kind":     "llm",
							"modelRef": "extractor",
						},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "classify_task"},
						map[string]interface{}{"from": "classify_task", "to": "extract_facts"},
						map[string]interface{}{"from": "extract_facts", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner":   {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
					"extractor": {Provider: "openai", Model: "gpt-4.1-mini", BaseURL: server.URL},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 model calls, got %d", callCount)
	}

	// Both node outputs should be present.
	if _, ok := result.Output["classify_task"].(map[string]interface{}); !ok {
		t.Fatalf("expected classify_task output, got %#v", result.Output)
	}
	if _, ok := result.Output["extract_facts"].(map[string]interface{}); !ok {
		t.Fatalf("expected extract_facts output, got %#v", result.Output)
	}
}

func TestEinoADKRunnerFallsBackToSingleModelWhenNoGraph(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"{\"summary\":\"done\"}"}}]}`))
	}))
	defer server.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput:    map[string]interface{}{"task": "test"},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "Hello."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Output["model"] != "planner" {
		t.Fatalf("expected planner model in output, got %#v", result.Output)
	}
}

func TestEinoADKRunnerRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := EinoADKRunner{}
	_, err := runner.Run(ctx, RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput:    map[string]interface{}{},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestGraphBuilderReturnsNilForEmptyGraph(t *testing.T) {
	runnable, err := buildGraph(
		context.Background(),
		contract.CompiledArtifact{},
		contract.WorkerRuntimeInfo{},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("buildGraph returned error: %v", err)
	}
	if runnable != nil {
		t.Fatalf("expected nil runnable for empty graph, got %v", runnable)
	}
}

func TestGraphBuilderBuildsSingleNodeGraph(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"{\"result\":\"ok\"}"}}]}`))
	}))
	defer server.Close()

	artifact := contract.CompiledArtifact{
		Runner: contract.ArtifactRunner{
			Models: map[string]contract.ModelConfig{
				"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
			},
			Prompts: map[string]contract.PromptSpec{
				"system": {Template: "Hello."},
			},
			Graph: map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"name":     "my_llm",
						"kind":     "llm",
						"modelRef": "planner",
					},
				},
			},
		},
	}
	runtimeInfo := contract.WorkerRuntimeInfo{
		Models: map[string]contract.WorkerModelRuntime{
			"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL, APIKeyEnv: "MODEL_PLANNER_API_KEY"},
		},
	}

	runnable, err := buildGraph(context.Background(), artifact, runtimeInfo, EinoOpenAIInvoker{}, nil, nil)
	if err != nil {
		t.Fatalf("buildGraph returned error: %v", err)
	}
	if runnable == nil {
		t.Fatal("expected non-nil runnable")
	}

	output, err := runnable.Invoke(context.Background(), map[string]interface{}{"task": "test"})
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if _, ok := output["my_llm"]; !ok {
		t.Fatalf("expected my_llm in output, got %#v", output)
	}
}

func TestPromptRendererReturnsTemplateAsIs(t *testing.T) {
	prompt := contract.PromptSpec{
		Template: "You are an EHS assistant.",
	}
	result, err := renderPrompt(prompt, map[string]interface{}{"task": "test"})
	if err != nil {
		t.Fatalf("renderPrompt returned error: %v", err)
	}
	if result != "You are an EHS assistant." {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestPromptRendererRendersGoTemplate(t *testing.T) {
	prompt := contract.PromptSpec{
		Template: "Task: {{.task}}",
		Variables: []contract.PromptVariableSpec{
			{Name: "task", Required: true},
		},
	}
	result, err := renderPrompt(prompt, map[string]interface{}{"task": "identify_hazard"})
	if err != nil {
		t.Fatalf("renderPrompt returned error: %v", err)
	}
	if result != "Task: identify_hazard" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestRenderUserMessageUsesDirectString(t *testing.T) {
	result := renderUserMessage(map[string]interface{}{"message": "hello world"})
	if result != "hello world" {
		t.Fatalf("expected direct string, got %q", result)
	}
}

func TestRenderUserMessageSerializesJSON(t *testing.T) {
	result := renderUserMessage(map[string]interface{}{"task": "test", "count": 3})
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}
	if parsed["task"] != "test" {
		t.Fatalf("unexpected result: %q", result)
	}
}

func TestGraphSurvivesJSONRoundTrip(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","choices":[{"message":{"role":"assistant","content":"{\"result\":\"ok\"}"}}]}`))
	}))
	defer server.Close()

	// Build an artifact the way the compiler would: typed graph nodes.
	original := contract.CompiledArtifact{
		Kind: contract.CompiledArtifactKind,
		Runtime: contract.ArtifactRuntime{
			Engine:      "eino",
			RunnerClass: "adk",
		},
		Runner: contract.ArtifactRunner{
			Models: map[string]contract.ModelConfig{
				"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
			},
			Prompts: map[string]contract.PromptSpec{
				"system": {Template: "Test."},
			},
			Graph: map[string]interface{}{
				"nodes": []interface{}{
					map[string]interface{}{
						"name":     "step_1",
						"kind":     "llm",
						"modelRef": "planner",
					},
					map[string]interface{}{
						"name":     "step_2",
						"kind":     "llm",
						"modelRef": "planner",
					},
				},
				"edges": []interface{}{
					map[string]interface{}{"from": "START", "to": "step_1"},
					map[string]interface{}{"from": "step_1", "to": "step_2"},
					map[string]interface{}{"from": "step_2", "to": "END"},
				},
			},
		},
	}

	// Simulate the round-trip: marshal to JSON then unmarshal back.
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	roundTripped, err := contract.ParseCompiledArtifact(string(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	// Verify graph survived.
	graphDef := roundTripped.Runner.Graph
	nodesRaw, ok := graphDef["nodes"].([]interface{})
	if !ok {
		t.Fatalf("expected []interface{} for nodes, got %T: %#v", graphDef["nodes"], graphDef)
	}
	if len(nodesRaw) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodesRaw))
	}

	// Verify buildGraph works on the round-tripped artifact.
	runnable, err := buildGraph(context.Background(), roundTripped, contract.WorkerRuntimeInfo{
		Models: map[string]contract.WorkerModelRuntime{
			"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL, APIKeyEnv: "MODEL_PLANNER_API_KEY"},
		},
	}, EinoOpenAIInvoker{}, nil, nil)
	if err != nil {
		t.Fatalf("buildGraph on round-tripped artifact: %v", err)
	}
	if runnable == nil {
		t.Fatal("expected non-nil runnable from round-tripped artifact")
	}
}
