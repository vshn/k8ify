package main

import (
	"fmt"
	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/spf13/pflag"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/util"
	"log"
	"os"
	"time"
)

var (
	defaultConfig = internal.Config{
		OutputDir:    "manifests",
		Env:          "dev",
		Ref:          "",
		IngressPatch: converter.IngressPatch{},
	}
	modifiedImages internal.ModifiedImagesFlag
)

func main() {
	pflag.Var(&modifiedImages, "modified-image", "Image that has been modified during the build. Can be repeated.")
	code := Main(os.Args)
	os.Exit(code)
}

func Main(args []string) int {
	err := pflag.CommandLine.Parse(args[1:])
	if err != nil {
		log.Println(err)
		return 1
	}
	plainArgs := pflag.Args()

	config := internal.ConfigMerge(defaultConfig, internal.ReadConfig(".k8ify.defaults.yaml"), internal.ReadConfig(".k8ify.local.yaml"))
	if len(plainArgs) > 0 {
		config.Env = plainArgs[0]
	}
	if len(plainArgs) > 1 {
		config.Ref = plainArgs[1]
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
	forceRestartAnnotation := make(map[string]string)
	forceRestartAnnotation["k8ify.restart-trigger"] = fmt.Sprintf("%d", time.Now().Unix())
	converter.PatchDeployments(objects.Deployments, modifiedImages.Values, forceRestartAnnotation)
	converter.PatchStatefulSets(objects.StatefulSets, modifiedImages.Values, forceRestartAnnotation)

	err = internal.WriteManifests(config.OutputDir, objects)
	if err != nil {
		log.Println(err)
		return 1
	}

	return 0
}
