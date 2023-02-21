package converter

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type OtherResource interface {
	runtime.Object
	GetObjectMeta() metav1.Object
}
