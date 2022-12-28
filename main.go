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
	outputDir   = "manifests"
	env         = "prod"
	configFiles = [4]string{"compose.yml", "docker-compose.yml", "compose-" + env + ".yml", "docker-compose-" + env + ".yml"}
)

func main() {
	composeConfigFiles := []composeTypes.ConfigFile{}
	for _, configFile := range configFiles {
		if _, err := os.Stat(configFile); err == nil {
			composeConfigFiles = append(composeConfigFiles, composeTypes.ConfigFile{
				Filename: configFile,
			})
		}
	}
	configDetails := composeTypes.ConfigDetails{
		ConfigFiles: composeConfigFiles,
		Environment: util.GetEnv(env + "_"),
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
		deployment, service, servicePersistentVolumeClaims, secret, serviceIngresses := converter.ComposeServiceToK8s(composeService)
		deployments = append(deployments, deployment)
		services = append(services, service)
		persistentVolumeClaims = append(persistentVolumeClaims, servicePersistentVolumeClaims...)
		secrets = append(secrets, secret)
		ingresses = append(ingresses, serviceIngresses...)
	}

	err = internal.WriteManifests(outputDir, deployments, services, persistentVolumeClaims, secrets, ingresses)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
