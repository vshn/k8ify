package converter

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

func addAnnotations(annotations *map[string]string, addAnnotations map[string]string) {
	if *annotations == nil {
		*annotations = make(map[string]string)
	}
	for k, v := range addAnnotations {
		(*annotations)[k] = v
	}
}

func isModifiedImage(image string, modifiedImages []string) bool {
	for _, modifiedImage := range modifiedImages {
		// check if we have an exact match
		if image == modifiedImage {
			return true
		}
		// check if we have a suffix match. For this we must ensure that the modifiedImage string starts with a "/",
		// otherwise we might get nonsensical partial matches
		if !strings.HasPrefix(modifiedImage, "/") {
			modifiedImage = "/" + modifiedImage
		}
		if strings.HasSuffix(image, modifiedImage) {
			return true
		}
	}
	return false
}

func getSecretByName(secrets []core.Secret, name string) (*core.Secret, error) {
	for i := range secrets {
		if secrets[i].Name == name {
			return &secrets[i], nil
		}
	}
	return nil, fmt.Errorf("secret %q not found", name)
}

// As environments could potentionally be very long we concatenate everything and hash at the end
// We sort beforehand to not depend on the order of key-value entries
func hashSecret(secret *core.Secret) uint32 {
	sortedKeys := make([]string, 0, len(secret.StringData))
	for k := range secret.StringData {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// Concatenate all key-value pairs
	hashInput := []byte{}
	for _, k := range sortedKeys {
		hashInput = append(hashInput, []byte(k)...)
		hashInput = append(hashInput, secret.Data[k]...)
	}

	hash := fnv.New32a()
	hash.Write([]byte(hashInput))
	return hash.Sum32()
}

// We use XOR to not depend on the order of the secrets and not cause an overflow
func hashSecrets(secrets []*core.Secret) uint32 {
	var hash uint32 = 0
	for _, secret := range secrets {
		hash = hash ^ hashSecret(secret)
	}
	return hash
}

func patchPodTemplate(template *core.PodTemplateSpec, secrets []core.Secret, modifiedImages []string, forceRestartAnnotation map[string]string) {
	matchingSecrets := []*core.Secret{}
	modified := false
	for _, container := range template.Spec.Containers {
		for _, env := range container.EnvFrom {
			if env.SecretRef == nil {
				continue
			}
			secret, err := getSecretByName(secrets, env.SecretRef.Name)
			if err != nil {
				continue
			}
			matchingSecrets = append(matchingSecrets, secret)
		}
		for _, env := range container.Env {
			if env.ValueFrom == nil || env.ValueFrom.SecretKeyRef == nil {
				continue
			}
			secret, err := getSecretByName(secrets, env.ValueFrom.SecretKeyRef.Name)
			if err != nil {
				continue
			}
			matchingSecrets = append(matchingSecrets, secret)
		}
		if isModifiedImage(container.Image, modifiedImages) {
			modified = true
		}
	}
	hash := hashSecrets(matchingSecrets)
	if hash != 0 {
		addAnnotations(&template.Annotations, map[string]string{
			"k8ify.restart-trigger-config": fmt.Sprint(hash),
		})
	}
	if modified {
		addAnnotations(&template.Annotations, forceRestartAnnotation)
	}
}

func PatchDeployments(deployments []apps.Deployment, modifiedImages []string, secrets []core.Secret, forceRestartAnnotation map[string]string) {
	// don't use 'range', getting a pointer to an array element does not work with 'range'
	for i := 0; i < len(deployments); i++ {
		patchPodTemplate(&deployments[i].Spec.Template, secrets, modifiedImages, forceRestartAnnotation)
	}
}

func PatchStatefulSets(statefulSets []apps.StatefulSet, modifiedImages []string, secrets []core.Secret, forceRestartAnnotation map[string]string) {
	// don't use 'range', getting a pointer to an array element does not work with 'range'
	for i := 0; i < len(statefulSets); i++ {
		patchPodTemplate(&statefulSets[i].Spec.Template, secrets, modifiedImages, forceRestartAnnotation)
	}
}
