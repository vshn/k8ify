package internal

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
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

func writeManifest(obj runtime.Object, destination string) error {
	f, err := os.Create(destination)
	if err != nil {
		return err
	}
	yp := printers.YAMLPrinter{}
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
		logrus.Error(err)
		os.Exit(1)
	}

	for _, deployment := range objects.Deployments {
		err := writeManifest(&deployment, outputDir+"/"+deployment.Name+"-deployment.yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d deployments\n", len(objects.Deployments))

	for _, statefulset := range objects.StatefulSets {
		err := writeManifest(&statefulset, outputDir+"/"+statefulset.Name+"-statefulset.yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d statefulsets\n", len(objects.StatefulSets))

	for _, service := range objects.Services {
		err := writeManifest(&service, outputDir+"/"+service.Name+"-service.yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d services\n", len(objects.Services))

	for _, persistentVolumeClaim := range objects.PersistentVolumeClaims {
		err := writeManifest(&persistentVolumeClaim, outputDir+"/"+persistentVolumeClaim.Name+"-persistentvolumeclaim.yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d persistentVolumeClaims\n", len(objects.PersistentVolumeClaims))

	for _, secret := range objects.Secrets {
		err := writeManifest(&secret, outputDir+"/"+secret.Name+"-secret.yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d secrets\n", len(objects.Secrets))

	for _, ingress := range objects.Ingresses {
		err := writeManifest(&ingress, outputDir+"/"+ingress.Name+"-ingress.yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d ingresses\n", len(objects.Ingresses))

	for _, other := range objects.Others {
		err := writeManifest(&other, outputDir+"/"+other.GetName()+"-"+strings.ToLower(other.GetObjectKind().GroupVersionKind().Kind)+".yaml")
		if err != nil {
			return err
		}
	}
	logrus.Infof("wrote %d other objects\n", len(objects.Others))

	return nil
}
