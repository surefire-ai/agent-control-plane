package runtime

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
)

func JSONValue(value interface{}) apiextensionsv1.JSON {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("null")
	}
	return apiextensionsv1.JSON{Raw: raw}
}

func JSONString(values apiv1alpha1.FreeformObject, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	var result string
	if err := json.Unmarshal(value.Raw, &result); err != nil {
		return ""
	}
	return result
}
