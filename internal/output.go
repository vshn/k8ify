package internal

import (
	"log"
	"os"
	"strings"

	"github.com/vshn/k8ify/pkg/converter"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
)

func prepareOutputDir(outputDir string) error {
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		return err
	}

	files, err := os.ReadDir(outputDir)
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

func WriteManifests(outputDir string, objects converter.Objects) error {
	err := prepareOutputDir(outputDir)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	yp := printers.YAMLPrinter{}

	for _, deployment := range objects.Deployments {
		err := writeManifest(&yp, &deployment, outputDir+"/"+deployment.Name+"-deployment.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d deployments\n", len(objects.Deployments))

	for _, statefulset := range objects.StatefulSets {
		err := writeManifest(&yp, &statefulset, outputDir+"/"+statefulset.Name+"-statefulset.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d statefulsets\n", len(objects.StatefulSets))

	for _, service := range objects.Services {
		err := writeManifest(&yp, &service, outputDir+"/"+service.Name+"-service.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d services\n", len(objects.Services))

	for _, persistentVolumeClaim := range objects.PersistentVolumeClaims {
		err := writeManifest(&yp, &persistentVolumeClaim, outputDir+"/"+persistentVolumeClaim.Name+"-persistentvolumeclaim.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d persistentVolumeClaims\n", len(objects.PersistentVolumeClaims))

	for _, secret := range objects.Secrets {
		err := writeManifest(&yp, &secret, outputDir+"/"+secret.Name+"-secret.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d secrets\n", len(objects.Secrets))

	for _, ingress := range objects.Ingresses {
		err := writeManifest(&yp, &ingress, outputDir+"/"+ingress.Name+"-ingress.yaml")
		if err != nil {
			return err
		}
	}
	log.Printf("wrote %d ingresses\n", len(objects.Ingresses))

	return nil
}
