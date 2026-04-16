package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "windosx.com"
	Version = "v1alpha1"
)

var GroupVersion = schema.GroupVersion{
	Group:   Group,
	Version: Version,
}
