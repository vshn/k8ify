package internal

import "github.com/vshn/k8ify/pkg/converter"

type Config struct {
	OutputDir    string                 `json:"outputDir"`
	Env          string                 `json:"env"`
	ConfigFiles  []string               `json:"configFiles"`
	IngressPatch converter.IngressPatch `json:"ingressPatch"`
}
