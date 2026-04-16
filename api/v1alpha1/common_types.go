package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ConditionedStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type LocalObjectReference struct {
	Name string `json:"name"`
}

type SecretKeyReference struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type JSONSchema map[string]interface{}

type NamedReference struct {
	Name string `json:"name"`
	Ref  string `json:"ref,omitempty"`
}
