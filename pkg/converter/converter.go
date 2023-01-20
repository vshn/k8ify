package converter

import (
	"fmt"
	"strconv"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func composeServiceStorageToK8s() map[core.ResourceName]resource.Quantity {
	quantity := make(map[core.ResourceName]resource.Quantity)
	quantity["storage"], _ = resource.ParseQuantity("100Mi")
	return quantity
}

func composeServiceVolumesToK8s(
	serviceName string,
	serviceVolumes []composeTypes.ServiceVolumeConfig,
	labels map[string]string,
	accessMode core.PersistentVolumeAccessMode,
) ([]core.Volume, []core.VolumeMount, []core.PersistentVolumeClaim) {

	volumeMounts := []core.VolumeMount{}
	volumes := []core.Volume{}
	persistentVolumeClaims := []core.PersistentVolumeClaim{}
	for i, volume := range serviceVolumes {
		name := util.Sanitize(volume.Source)
		if len(name) == 0 || strings.HasPrefix(name, "claim") {
			name = fmt.Sprintf("%s-claim%d", serviceName, i)
		}
		volumeMounts = append(volumeMounts, core.VolumeMount{
			MountPath: volume.Target,
			Name:      name,
		})
		volumes = append(volumes, core.Volume{
			Name: name,
			VolumeSource: core.VolumeSource{
				PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
					ClaimName: name,
				},
			},
		})
		persistentVolumeClaims = append(persistentVolumeClaims, core.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "PersistentVolumeClaim",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: labels,
			},
			Spec: core.PersistentVolumeClaimSpec{
				AccessModes: []core.PersistentVolumeAccessMode{accessMode},
				Resources: core.ResourceRequirements{
					Requests: composeServiceStorageToK8s(),
				},
			},
		})
	}
	return volumes, volumeMounts, persistentVolumeClaims
}

func composeServicePortsToK8s(composeServicePorts []composeTypes.ServicePortConfig) ([]core.ContainerPort, []core.ServicePort) {
	containerPorts := []core.ContainerPort{}
	servicePorts := []core.ServicePort{}
	for _, port := range composeServicePorts {
		publishedPort, err := strconv.Atoi(port.Published)
		if err != nil {
			publishedPort = int(port.Target)
		}
		containerPorts = append(containerPorts, core.ContainerPort{
			ContainerPort: int32(port.Target),
		})
		servicePorts = append(servicePorts, core.ServicePort{
			Name: fmt.Sprint(publishedPort),
			Port: int32(publishedPort),
			TargetPort: intstr.IntOrString{
				IntVal: int32(port.Target),
			},
		})
	}
	return containerPorts, servicePorts
}

func composeServiceToSecret(refSlug string, composeService composeTypes.ServiceConfig, labels map[string]string) core.Secret {
	stringData := make(map[string]string)
	for key, value := range composeService.Environment {
		stringData[key] = *value
	}
	secret := core.Secret{}
	secret.APIVersion = "v1"
	secret.Kind = "Secret"
	secret.Name = composeService.Name + refSlug + "-env"
	secret.Labels = labels
	secret.StringData = stringData
	return secret
}

func composeServiceToDeployment(
	refSlug string,
	composeService composeTypes.ServiceConfig,
	containerPorts []core.ContainerPort,
	volumes []core.Volume,
	volumeMounts []core.VolumeMount,
	secretName string,
	labels map[string]string,
) apps.Deployment {

	strategy := apps.DeploymentStrategy{}
	strategy.Type = apps.RecreateDeploymentStrategyType

	deployment := apps.Deployment{}
	deployment.APIVersion = "apps/v1"
	deployment.Kind = "Deployment"
	deployment.Name = composeService.Name + refSlug
	deployment.Labels = labels

	templateSpec := composeServiceToPodTemplate(
		deployment.Name,
		composeService.Image,
		secretName,
		containerPorts,
		volumes,
		volumeMounts,
		labels,
	)

	deployment.Spec = apps.DeploymentSpec{
		Replicas: composeServiceToReplicas(composeService),
		Strategy: strategy,
		Template: templateSpec,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
	}

	return deployment
}

func composeServiceToStatefulSet(
	refSlug string,
	composeService composeTypes.ServiceConfig,
	containerPorts []core.ContainerPort,
	volumes []core.Volume,
	volumeMounts []core.VolumeMount,
	volumeClaims []core.PersistentVolumeClaim,
	secretName string,
	labels map[string]string,
) apps.StatefulSet {

	statefulset := apps.StatefulSet{}
	statefulset.APIVersion = "apps/v1"
	statefulset.Kind = "StatefulSet"
	statefulset.Name = composeService.Name + refSlug
	statefulset.Labels = labels

	templateSpec := composeServiceToPodTemplate(
		statefulset.Name,
		composeService.Image,
		secretName,
		containerPorts,
		volumes,
		volumeMounts,
		labels,
	)

	statefulset.Spec = apps.StatefulSetSpec{
		Replicas: composeServiceToReplicas(composeService),
		Template: templateSpec,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		VolumeClaimTemplates: volumeClaims,
	}

	return statefulset
}

func composeServiceToReplicas(composeService composeTypes.ServiceConfig) *int32 {
	deploy := composeService.Deploy
	if deploy == nil || deploy.Replicas == nil {
		return nil
	}
	// deploy.Replicas is an Uint64, but if you have over 2'000'000'000
	// replicas, you might have different problems :)
	return pointer.Int32(int32(*deploy.Replicas))
}

func composeServiceToPodTemplate(
	name string,
	image string,
	secretName string,
	ports []core.ContainerPort,
	volumes []core.Volume,
	volumeMounts []core.VolumeMount,
	labels map[string]string,
) core.PodTemplateSpec {

	container := core.Container{
		Name:  name,
		Image: image,
		Ports: ports,
		// We COULD put the environment variables here, but because some of them likely contain sensitive data they are stored in a secret instead
		// Env:          envVars,
		// Reference the secret:
		EnvFrom: []core.EnvFromSource{
			{
				SecretRef: &core.SecretEnvSource{
					LocalObjectReference: core.LocalObjectReference{
						Name: secretName,
					},
				},
			},
		},
		VolumeMounts: volumeMounts,
	}

	podSpec := core.PodSpec{
		Containers:    []core.Container{container},
		RestartPolicy: core.RestartPolicyAlways,
		Volumes:       volumes,
	}

	return core.PodTemplateSpec{
		Spec: podSpec,
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
	}
}

func composeServiceToService(refSlug string, composeService composeTypes.ServiceConfig, servicePorts []core.ServicePort, labels map[string]string) core.Service {
	serviceSpec := core.ServiceSpec{
		Ports:    servicePorts,
		Selector: labels,
	}
	service := core.Service{}
	service.Spec = serviceSpec
	service.APIVersion = "v1"
	service.Kind = "Service"
	service.Name = composeService.Name + refSlug
	service.Labels = labels
	return service
}

func composeServiceToIngress(refSlug string, composeService composeTypes.ServiceConfig, service core.Service, labels map[string]string) []networking.Ingress {
	ingresses := []networking.Ingress{}
	for i, port := range service.Spec.Ports {
		// we expect the config to be in "k8ify.expose.PORT"
		configPrefix := fmt.Sprintf("k8ify.expose.%d", port.Port)
		ingressConfig := util.SubConfig(composeService.Labels, configPrefix, "host")
		if _, ok := ingressConfig["host"]; !ok && i == 0 {
			// for the first port we also accept config in "k8ify.expose"
			ingressConfig = util.SubConfig(composeService.Labels, "k8ify.expose", "host")
		}

		if host, ok := ingressConfig["host"]; ok {
			ingress := networking.Ingress{}
			ingress.APIVersion = "networking.k8s.io/v1"
			ingress.Kind = "Ingress"
			ingress.Name = fmt.Sprintf("%s%s-%d", composeService.Name, refSlug, service.Spec.Ports[i].Port)
			ingress.Labels = labels

			serviceBackendPort := networking.ServiceBackendPort{
				Number: service.Spec.Ports[i].Port,
			}

			ingressServiceBackend := networking.IngressServiceBackend{
				Name: composeService.Name,
				Port: serviceBackendPort,
			}

			ingressBackend := networking.IngressBackend{
				Service: &ingressServiceBackend,
			}

			pathType := networking.PathTypePrefix
			path := networking.HTTPIngressPath{
				PathType: &pathType,
				Path:     "/",
				Backend:  ingressBackend,
			}

			httpIngressRuleValue := networking.HTTPIngressRuleValue{
				Paths: []networking.HTTPIngressPath{path},
			}

			ingressRuleValue := networking.IngressRuleValue{
				HTTP: &httpIngressRuleValue,
			}

			ingressRule := networking.IngressRule{
				Host:             host,
				IngressRuleValue: ingressRuleValue,
			}

			ingressTls := networking.IngressTLS{
				Hosts:      []string{host},
				SecretName: fmt.Sprintf("%s-tls", ingress.Name),
			}

			ingressSpec := networking.IngressSpec{
				Rules: []networking.IngressRule{ingressRule},
				TLS:   []networking.IngressTLS{ingressTls},
			}

			ingress.Spec = ingressSpec
			ingresses = append(ingresses, ingress)
		}
	}
	return ingresses
}

func toRefSlug(ref string, composeService composeTypes.ServiceConfig) string {
	if ref == "" {
		return ""
	}
	if singleton, ok := composeService.Labels["k8ify.singleton"]; ok {
		if util.IsTruthy(singleton) {
			return ""
		}
	}
	return ref
}

func ComposeServiceToK8s(ref string, composeService composeTypes.ServiceConfig) Objects {
	refSlug := toRefSlug(util.SanitizeWithMinLength(ref, 3), composeService)
	labels := make(map[string]string)
	labels["k8ify.service"] = composeService.Name
	if refSlug != "" {
		labels["k8ify.ref-slug"] = refSlug
		refSlug = "-" + refSlug
	}

	objects := Objects{}

	secret := composeServiceToSecret(refSlug, composeService, labels)
	objects.Secrets = []core.Secret{secret}

	containerPorts, servicePorts := composeServicePortsToK8s(composeService.Ports)
	service := composeServiceToService(refSlug, composeService, servicePorts, labels)
	objects.Services = []core.Service{service}

	shareStorage := util.IsTruthy(composeService.Labels["k8ify.share-storage"])
	accessMode := core.ReadWriteOnce
	if shareStorage {
		accessMode = core.ReadWriteMany
	}
	volumes, volumeMounts, persistentVolumeClaims := composeServiceVolumesToK8s(
		composeService.Name+refSlug, composeService.Volumes, labels, accessMode,
	)

	if shareStorage || len(volumeMounts) < 1 {
		objects.PersistentVolumeClaims = persistentVolumeClaims
		deployment := composeServiceToDeployment(refSlug,
			composeService,
			containerPorts,
			volumes,
			volumeMounts,
			secret.Name,
			labels,
		)
		objects.Deployments = []apps.Deployment{deployment}

	} else {
		// StatefulSets create their own PVC's via `spec.volumeTemplate`, so we don't include the PVC objects here
		statefulset := composeServiceToStatefulSet(
			refSlug,
			composeService,
			containerPorts,
			volumes,
			volumeMounts,
			persistentVolumeClaims,
			secret.Name,
			labels,
		)
		objects.StatefulSets = []apps.StatefulSet{statefulset}

	}

	ingresses := composeServiceToIngress(refSlug, composeService, service, labels)
	objects.Ingresses = ingresses

	return objects
}

// Objects combines all possible resources the conversion process could produce
type Objects struct {
	// Deployments
	Deployments            []apps.Deployment
	StatefulSets           []apps.StatefulSet
	Services               []core.Service
	PersistentVolumeClaims []core.PersistentVolumeClaim
	Secrets                []core.Secret
	Ingresses              []networking.Ingress
}

func (o Objects) Append(other Objects) Objects {
	return Objects{
		Deployments:            append(o.Deployments, other.Deployments...),
		StatefulSets:           append(o.StatefulSets, other.StatefulSets...),
		Services:               append(o.Services, other.Services...),
		PersistentVolumeClaims: append(o.PersistentVolumeClaims, other.PersistentVolumeClaims...),
		Secrets:                append(o.Secrets, other.Secrets...),
		Ingresses:              append(o.Ingresses, other.Ingresses...),
	}
}
