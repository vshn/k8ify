package converter

import (
	"strings"

	apps "k8s.io/api/apps/v1"
	networking "k8s.io/api/networking/v1"
)

func PatchIngresses(ingresses []networking.Ingress, ingressPatch IngressPatch) {
	// don't use 'range', getting a pointer to an array element does not work with 'range'
	for i := 0; i < len(ingresses); i++ {
		PatchIngress(&ingresses[i], ingressPatch)
	}
}

func PatchIngress(ingress *networking.Ingress, ingressPatch IngressPatch) {
	addAnnotations(&ingress.Annotations, ingressPatch.AddAnnotations)
}

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

func PatchDeployments(deployments []apps.Deployment, modifiedImages []string, forceRestartAnnotation map[string]string) {
	// don't use 'range', getting a pointer to an array element does not work with 'range'
	for i := 0; i < len(deployments); i++ {
		for _, container := range deployments[i].Spec.Template.Spec.Containers {
			if isModifiedImage(container.Image, modifiedImages) {
				addAnnotations(&deployments[i].Spec.Template.Annotations, forceRestartAnnotation)
				break
			}
		}
	}
}

func PatchStatefulSets(statefulSets []apps.StatefulSet, modifiedImages []string, forceRestartAnnotation map[string]string) {
	// don't use 'range', getting a pointer to an array element does not work with 'range'
	for i := 0; i < len(statefulSets); i++ {
		for _, container := range statefulSets[i].Spec.Template.Spec.Containers {
			if isModifiedImage(container.Image, modifiedImages) {
				addAnnotations(&statefulSets[i].Spec.Template.Annotations, forceRestartAnnotation)
				break
			}
		}
	}
}
