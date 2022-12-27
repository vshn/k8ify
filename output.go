package main

import (
	"io/ioutil"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"log"
	"os"
	"strings"
)

func prepareOutputDir(outputDir string) error {
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	files, err := ioutil.ReadDir(outputDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
			err = os.Remove(outputDir + "/" + file.Name())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func writeManifest(yp *printers.YAMLPrinter, obj runtime.Object, destination string) error {
	f, err := os.Create(destination)
	if err != nil {
		return err
	}
	err = yp.PrintObj(obj, f)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}

func writeManifests(deployments []apps.Deployment, services []core.Service, persistentVolumeClaims []core.PersistentVolumeClaim) error {
	yp := printers.YAMLPrinter{}

	for _, deployment := range deployments {
		err := writeManifest(&yp, &deployment, outputDir+"/"+deployment.Name+"-deployment.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d deployments\n", len(deployments))

	for _, service := range services {
		err := writeManifest(&yp, &service, outputDir+"/"+service.Name+"-service.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d services\n", len(services))

	for _, persistentVolumeClaim := range persistentVolumeClaims {
		err := writeManifest(&yp, &persistentVolumeClaim, outputDir+"/"+persistentVolumeClaim.Name+"-persistentvolumeclaim.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d persistentVolumeClaims\n", len(services))

	return nil
}
