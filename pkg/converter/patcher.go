package converter

import (
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
