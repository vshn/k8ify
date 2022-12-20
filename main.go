package main

import (
	"fmt"
	composeLoader "github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	apps "k8s.io/api/apps/v1"
	printers "k8s.io/cli-runtime/pkg/printers"
	"os"
)

func main() {
	configFile := composeTypes.ConfigFile{"/home/david/portal/docker-compose.yml", nil, nil}
	configFiles := []composeTypes.ConfigFile{configFile}
	configDetails := composeTypes.ConfigDetails{"3.4", "/home/david/portal/", configFiles, make(map[string]string)}
	project, _ := composeLoader.Load(configDetails)

	deployments := []apps.Deployment{}
	for _, service := range project.Services {
		deployments = append(deployments, serviceToDeployment(service))
	}

	for i, deployment := range deployments {
		fmt.Printf("== Deployment %d ==\n", i)
		yp := printers.YAMLPrinter{}
		err := yp.PrintObj(&deployment, os.Stdout)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
	}

	os.Exit(0)
}
