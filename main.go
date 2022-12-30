package main

import (
	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"log"
	"os"
)

var (
	hardcodedConfig = internal.Config{
		OutputDir: "manifests",
		Env:       "test",
		Ref:       "",
		IngressPatch: converter.IngressPatch{
			AddAnnotations: map[string]string{"cert-manager.io/cluster-issuer": "letsencrypt-production"},
		},
	}
)

func main() {
	config := hardcodedConfig

	if len(os.Args) > 1 {
		config.Env = os.Args[1]
	}
	if len(os.Args) > 2 {
		config.Ref = os.Args[2]
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
		log.Fatal(err)
		os.Exit(1)
	}

	deployments := []apps.Deployment{}
	services := []core.Service{}
	persistentVolumeClaims := []core.PersistentVolumeClaim{}
	secrets := []core.Secret{}
	ingresses := []networking.Ingress{}
	for _, composeService := range project.Services {
		deployment, service, servicePersistentVolumeClaims, secret, serviceIngresses := converter.ComposeServiceToK8s(config.Ref, composeService)
		deployments = append(deployments, deployment)
		services = append(services, service)
		persistentVolumeClaims = append(persistentVolumeClaims, servicePersistentVolumeClaims...)
		secrets = append(secrets, secret)
		ingresses = append(ingresses, serviceIngresses...)
	}

	converter.PatchIngresses(ingresses, config.IngressPatch)

	err = internal.WriteManifests(config.OutputDir, deployments, services, persistentVolumeClaims, secrets, ingresses)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
