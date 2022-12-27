package main

import (
	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"log"
	"os"
)

var (
	outputDir = "manifests"
)

func main() {
	configFile := composeTypes.ConfigFile{"/home/david/portal/docker-compose.yml", nil, nil}
	configFiles := []composeTypes.ConfigFile{configFile}
	configDetails := composeTypes.ConfigDetails{"3.4", "/home/david/portal/", configFiles, make(map[string]string)}
	project, err := composeLoader.Load(configDetails)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	deployments := []apps.Deployment{}
	services := []core.Service{}
	for _, composeService := range project.Services {
		deployment, service := composeServiceToDeploymentAndService(composeService)
		deployments = append(deployments, deployment)
		services = append(services, service)
	}

	err = prepareOutputDir(outputDir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	err = writeManifests(deployments, services)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(0)
}
