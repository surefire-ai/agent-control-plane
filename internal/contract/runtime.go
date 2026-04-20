package contract

import "fmt"

const (
	RuntimeEngineEino = "eino"
	RunnerClassADK    = "adk"
)

type RuntimeSpec struct {
	Engine      string
	RunnerClass string
}

type RuntimeIdentity struct {
	Engine      string
	RunnerClass string
}

func DefaultRuntimeIdentity() RuntimeIdentity {
	return RuntimeIdentity{
		Engine:      RuntimeEngineEino,
		RunnerClass: RunnerClassADK,
	}
}

func RuntimeIdentityFromSpec(spec RuntimeSpec) RuntimeIdentity {
	identity := DefaultRuntimeIdentity()
	if spec.Engine != "" {
		identity.Engine = spec.Engine
	}
	if spec.RunnerClass != "" {
		identity.RunnerClass = spec.RunnerClass
	}
	return identity
}

func RuntimeIdentityFromMap(values map[string]interface{}) RuntimeIdentity {
	identity := DefaultRuntimeIdentity()
	if engine := runtimeString(values, "engine"); engine != "" {
		identity.Engine = engine
	}
	if runnerClass := runtimeString(values, "runnerClass"); runnerClass != "" {
		identity.RunnerClass = runnerClass
	}
	return identity
}

func (i RuntimeIdentity) ValidateSupported() error {
	if i.Engine != RuntimeEngineEino {
		return fmt.Errorf("unsupported runtime engine %q", i.Engine)
	}
	if i.RunnerClass != RunnerClassADK {
		return fmt.Errorf("unsupported runner class %q for runtime engine %q", i.RunnerClass, i.Engine)
	}
	return nil
}

func runtimeString(values map[string]interface{}, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	output, _ := value.(string)
	return output
}
