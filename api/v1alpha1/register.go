package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	SchemeGroupVersion = schema.GroupVersion{
		Group:   Group,
		Version: Version,
	}

	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Agent{},
		&AgentList{},
		&PromptTemplate{},
		&PromptTemplateList{},
		&KnowledgeBase{},
		&KnowledgeBaseList{},
		&ToolProvider{},
		&ToolProviderList{},
		&MCPServer{},
		&MCPServerList{},
		&AgentPolicy{},
		&AgentPolicyList{},
		&AgentRun{},
		&AgentRunList{},
		&AgentEvaluation{},
		&AgentEvaluationList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
