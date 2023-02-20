package main

import (
	"fmt"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/ir"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type SomethingObjectKind struct {
}

func (this SomethingObjectKind) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{Kind: "Something", Version: "v1", Group: "foo"}
}

func (this SomethingObjectKind) SetGroupVersionKind(kind schema.GroupVersionKind) {
}

type Something struct {
	internal.OtherResource
	Foo string
}

func (this Something) GetObjectKind() schema.ObjectKind {
	return SomethingObjectKind{}
}

func (this Something) DeepCopyObject() runtime.Object {
	return this
}

type converterPlugin struct{}

func (this converterPlugin) ComposeServiceToK8s(ref string, workload *ir.Service, projectVolumes map[string]ir.Volume) (converter.Objects, bool) {
	var o converter.Objects
	r := Something{Foo: "bar"}
	r.Name = "PluginGeneratedResource 1"
	o.Others = append(o.Others, r)
	fmt.Printf("%v\n", o.Others)
	return o, false
}

var ConverterPlugin converterPlugin
