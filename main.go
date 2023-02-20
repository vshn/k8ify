package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
	"plugin"
	"time"

	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/spf13/pflag"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/ir"
	"github.com/vshn/k8ify/pkg/util"
)

var (
	defaultConfig = internal.Config{
		OutputDir:    "manifests",
		Env:          "dev",
		Ref:          "",
		IngressPatch: converter.IngressPatch{},
	}
	modifiedImages internal.ModifiedImagesFlag
	shellEnvFiles  internal.ShellEnvFilesFlag
	convPlugin     ConverterPlugin
)

func main() {
	pflag.Var(&modifiedImages, "modified-image", "Image that has been modified during the build. Can be repeated.")
	pflag.Var(&shellEnvFiles, "shell-env-file", "Shell environment file ('key=value' format) to be used in addition to the current shell environment. Can be repeated.")

	plug, err := plugin.Open("plugins/plugins.so")
	if err != nil {
		log.Fatal("plugin plugins/plugins.so could not be loaded")
	}
	convPluginSymbol, err := plug.Lookup("ConverterPlugin")
	if err != nil {
		log.Fatal("plugin plugins/plugins.so does not have symbol ConverterPlugin")
	}
	var ok bool
	convPlugin, ok = convPluginSymbol.(ConverterPlugin)
	if !ok {
		log.Fatal("plugin plugins/plugins.so symbol ConverterPlugin has wrong type")
	}

	code := Main(os.Args)
	os.Exit(code)
}

type ConverterPlugin interface {
	ComposeServiceToK8s(ref string, workload *ir.Service, projectVolumes map[string]ir.Volume) (converter.Objects, bool)
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

	// Load the additional shell environment files. This will merge everything into the existing shell environment and
	// all values can later be retrieved using os.Environ()
	for _, shellEnvFile := range shellEnvFiles.Values {
		err := godotenv.Load(shellEnvFile)
		if err != nil {
			log.Println(err)
			return 1
		}
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

	inputs := ir.FromCompose(project)
	internal.VolumesPrecheck(inputs)

	objects := converter.Objects{}
	for _, volume := range inputs.Volumes {
		if v := converter.ComposeVolumeToK8s(config.Ref, &volume); v != nil {
			objects.PersistentVolumeClaims = append(objects.PersistentVolumeClaims, *v)
		}
	}

	for _, service := range inputs.Services {
		internal.ComposeServicePrecheck(service.AsCompose())
		pluginObjects, skipDefaultConverter := convPlugin.ComposeServiceToK8s(config.Ref, &service, inputs.Volumes)
		objects = objects.Append(pluginObjects)
		if !skipDefaultConverter {
			objects = objects.Append(converter.ComposeServiceToK8s(config.Ref, &service, inputs.Volumes))
		}
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
