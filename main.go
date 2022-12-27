package main

import (
	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"log"
	"os"
)

var (
	outputDir = "manifests"
)

func main() {
	configFile := composeTypes.ConfigFile{
		Filename: "docker-compose.yml",
	}
	configDetails := composeTypes.ConfigDetails{
		WorkingDir:  ".",
		ConfigFiles: []composeTypes.ConfigFile{configFile},
		Environment: make(map[string]string),
	}
	project, err := composeLoader.Load(configDetails)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	deployments := []apps.Deployment{}
	services := []core.Service{}
	persistentVolumeClaims := []core.PersistentVolumeClaim{}
	for _, composeService := range project.Services {
		deployment, service, servicePersistentVolumeClaims := converter.ComposeServiceToK8s(composeService)
		deployments = append(deployments, deployment)
		services = append(services, service)
		persistentVolumeClaims = append(persistentVolumeClaims, servicePersistentVolumeClaims...)
	}

	err = internal.WriteManifests(outputDir, deployments, services, persistentVolumeClaims)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
