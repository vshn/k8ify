package internal

import (
	"os"

	"github.com/vshn/k8ify/pkg/converter"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Config struct {
	OutputDir    string                 `json:"outputDir"`
	Env          string                 `json:"env"`
	Ref          string                 `json:"ref"`
	ConfigFiles  []string               `json:"configFiles"`
	IngressPatch converter.IngressPatch `json:"ingressPatch"`
}

func ReadConfig(fileName string) Config {
	buf, err := os.ReadFile(fileName)
	if err != nil {
		return Config{}
	}

	config := Config{}
	if err := yaml.Unmarshal(buf, &config); err != nil {
		return Config{}
	}

	return config
}

func ConfigMerge(configs ...Config) Config {
	if (len(configs)) == 0 {
		return Config{}
	}
	if len(configs) == 1 {
		return configs[0]
	}
	l := len(configs)
	configs = append(configs[0:l-2], configMergeTwo(configs[l-2], configs[l-1]))
	return ConfigMerge(configs...)
}

func configMergeTwo(config1 Config, config2 Config) Config {
	if config2.OutputDir != "" {
		config1.OutputDir = config2.OutputDir
	}
	if config2.Env != "" {
		config1.Env = config2.Env
	}
	if config2.Ref != "" {
		config1.Ref = config2.Ref
	}
	if config2.ConfigFiles != nil && len(config2.ConfigFiles) > 0 {
		config1.ConfigFiles = config2.ConfigFiles
	}
	if config2.IngressPatch.AddAnnotations != nil {
		if config1.IngressPatch.AddAnnotations == nil {
			config1.IngressPatch.AddAnnotations = config2.IngressPatch.AddAnnotations
		} else {
			for k, v := range config2.IngressPatch.AddAnnotations {
				config1.IngressPatch.AddAnnotations[k] = v
			}
		}
	}

	return config1
}
