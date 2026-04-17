//+kubebuilder:rbac:groups=windosx.com,resources=agents,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=agents/status,verbs=get;patch;update
//+kubebuilder:rbac:groups=windosx.com,resources=prompttemplates,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=knowledgebases,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=toolproviders,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=mcpservers,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=agentpolicies,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=agentruns,verbs=get;list;watch
//+kubebuilder:rbac:groups=windosx.com,resources=agentruns/status,verbs=get;patch;update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=create;get;list;watch;patch;update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch
package controller
