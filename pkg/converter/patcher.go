package converter

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"maps"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

var (
	emptyHash = sha256.New().Sum(nil)
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

// We sort beforehand to not depend on the order of key-value entries
func hashSecret(secret *core.Secret, hash hash.Hash) {
	sortedKeys := slices.Collect(maps.Keys(secret.StringData))
	slices.Sort(sortedKeys)

	for _, k := range sortedKeys {
		hash.Write([]byte(k))
		hash.Write([]byte(secret.StringData[k]))
	}
}

func hashSecrets(secrets []*core.Secret, hash hash.Hash) {
	for _, secret := range secrets {
		hashSecret(secret, hash)
	}
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
				logrus.Infof("Container %q: %s", container.Name, err.Error())
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
				logrus.Infof("Container %q: %s", container.Name, err.Error())
				continue
			}
			matchingSecrets = append(matchingSecrets, secret)
		}
		if isModifiedImage(container.Image, modifiedImages) {
			modified = true
		}
	}
	if template.Spec.ImagePullSecrets != nil {
		for _, auth := range template.Spec.ImagePullSecrets {
			secret, err := getSecretByName(secrets, auth.Name)
			if err != nil {
				logrus.Infof("ImagePullSecret: %s", err.Error())
				continue
			}
			matchingSecrets = append(matchingSecrets, secret)
		}
	}

	hash := sha256.New()
	hashSecrets(matchingSecrets, hash)
	hashSum := hash.Sum(nil)
	if !bytes.Equal(hashSum, emptyHash) {
		addAnnotations(&template.Annotations, map[string]string{
			"k8ify.restart-trigger-config": fmt.Sprintf("%x", hashSum),
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
