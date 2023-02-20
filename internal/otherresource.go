package internal

import "k8s.io/apimachinery/pkg/runtime"

type OtherResource struct {
	runtime.Object
	Name string
}
