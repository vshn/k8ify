package provider

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/vshn/k8ify/internal"
	"github.com/vshn/k8ify/pkg/converter"
	"github.com/vshn/k8ify/pkg/util"
	core "k8s.io/api/core/v1"
	"strings"
)

func PatchAppuioCloudscale(provider string, config internal.Config, objects converter.Objects) converter.Objects {
	if provider != "appuio-cloudscale" {
		return objects
	}
	return patchEncryptedPersistentVolumeClaims(config, objects)
}

func patchEncryptedPersistentVolumeClaims(config internal.Config, objects converter.Objects) converter.Objects {
	luksSecrets := []core.Secret{}
	for _, pvc := range objects.PersistentVolumeClaims {
		if pvc.Spec.StorageClassName != nil && (*pvc.Spec.StorageClassName == "ssd-encrypted" || *pvc.Spec.StorageClassName == "bulk-encrypted") {
			name := fmt.Sprintf("%s-luks-key", pvc.Name)
			envVarName := strings.ReplaceAll(name, "-", "_") + "_" + config.Env
			stringData := make(map[string]string)
			stringData["luksKey"] = util.GetEnvValueCaseInsensitive(envVarName)
			if stringData["luksKey"] == "" {
				logrus.Errorf("Volume '%s' is encrypted but no luksKey found. Use command 'pwgen -s 100 1' to generate luksKey and put it into environment variable '%s' (case insensitive). Continuing.", pvc.Name, envVarName)
				continue
			}
			luksSecret := core.Secret{}
			luksSecret.APIVersion = "v1"
			luksSecret.Kind = "Secret"
			luksSecret.Name = name
			luksSecret.Labels = pvc.Labels
			luksSecret.StringData = stringData
			luksSecrets = append(luksSecrets, luksSecret)
		}
	}

	// This is hacky. In case of statefulSets k8ify does not actually generate the PersistentVolumeClaims, these are generated
	// by k8s based on the VolumeClaimTemplates. But we can't template the secrets, hence we pre-generate the secrets
	// under the assumption that the template will be instantiated exactly once.
	for _, statefulSet := range objects.StatefulSets {
		for _, vcTemplate := range statefulSet.Spec.VolumeClaimTemplates {
			if vcTemplate.Spec.StorageClassName != nil && (*vcTemplate.Spec.StorageClassName == "ssd-encrypted" || *vcTemplate.Spec.StorageClassName == "bulk-encrypted") {
				name := fmt.Sprintf("%s-%s-%s-luks-key", vcTemplate.Name, statefulSet.Name, "0")
				envVarName := strings.ReplaceAll(name, "-", "_") + "_" + config.Env
				stringData := make(map[string]string)
				stringData["luksKey"] = util.GetEnvValueCaseInsensitive(envVarName)
				if stringData["luksKey"] == "" {
					logrus.Errorf("Volume '%s' is encrypted but no luksKey found. Use command 'pwgen -s 100 1' to generate luksKey and put it into environment variable '%s' (case insensitive). Continuing.", vcTemplate.Name, envVarName)
					continue
				}
				luksSecret := core.Secret{}
				luksSecret.APIVersion = "v1"
				luksSecret.Kind = "Secret"
				luksSecret.Name = name
				luksSecret.Labels = vcTemplate.Labels
				luksSecret.StringData = stringData
				luksSecrets = append(luksSecrets, luksSecret)
			}
		}
	}

	objects.Secrets = append(objects.Secrets, luksSecrets...)
	return objects
}
