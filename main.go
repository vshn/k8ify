package main

import (
	"log"
	"os"

	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/util"
)

var (
	defaultConfig = internal.Config{
		OutputDir:    "manifests",
		Env:          "dev",
		Ref:          "",
		IngressPatch: converter.IngressPatch{},
	}
)

func main() {
	code := Main(os.Args)
	os.Exit(code)
}

func Main(args []string) int {
	config := internal.ConfigMerge(defaultConfig, internal.ReadConfig(".k8ify.defaults.yaml"), internal.ReadConfig(".k8ify.local.yaml"))
	if len(args) > 1 {
		config.Env = args[1]
	}
	if len(args) > 2 {
		config.Ref = args[2]
	}

	if config.ConfigFiles == nil || len(config.ConfigFiles) == 0 {
		config.ConfigFiles = []string{"compose.yml", "docker-compose.yml", "compose-" + config.Env + ".yml", "docker-compose-" + config.Env + ".yml"}
	}

	composeConfigFiles := []composeTypes.ConfigFile{}
	for _, configFile := range config.ConfigFiles {
		if _, err := os.Stat(configFile); err == nil {
			composeConfigFiles = append(composeConfigFiles, composeTypes.ConfigFile{
				Filename: configFile,
			})
		}
	}
	configDetails := composeTypes.ConfigDetails{
		ConfigFiles: composeConfigFiles,
		Environment: util.GetEnv(),
	}
	project, err := composeLoader.Load(configDetails)
	if err != nil {
		log.Println(err)
		return 1
	}

	objects := converter.Objects{}

	for _, composeService := range project.Services {
		internal.ComposeServicePrecheck(composeService)
		objects = objects.Append(converter.ComposeServiceToK8s(config.Ref, composeService))
	}

	converter.PatchIngresses(objects.Ingresses, config.IngressPatch)

	err = internal.WriteManifests(config.OutputDir, objects)
	if err != nil {
		log.Println(err)
		return 1
	}

	return 0
}
