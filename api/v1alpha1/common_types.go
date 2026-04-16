package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

type FreeformObject map[string]apiextensionsv1.JSON

type JSONSchema apiextensionsv1.JSON

type NamedReference struct {
	Name string `json:"name"`
	Ref  string `json:"ref,omitempty"`
}
