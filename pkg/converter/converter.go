package converter

import (
	"fmt"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

func composeServiceStorageToK8s() map[core.ResourceName]resource.Quantity {
	quantity := make(map[core.ResourceName]resource.Quantity)
	quantity["storage"], _ = resource.ParseQuantity("100Mi")
	return quantity
}

func composeServiceVolumesToK8s(serviceName string, serviceVolumes []composeTypes.ServiceVolumeConfig, labels map[string]string) ([]core.Volume, []core.VolumeMount, []core.PersistentVolumeClaim) {
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
				AccessModes: []core.PersistentVolumeAccessMode{core.ReadWriteOnce},
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
		containerPorts = append(containerPorts, core.ContainerPort{
			ContainerPort: int32(port.Target),
		})
		servicePorts = append(servicePorts, core.ServicePort{
			Name: fmt.Sprint(port.Target),
			TargetPort: intstr.IntOrString{
				IntVal: int32(port.Target),
			},
			Port: int32(port.Target),
		})
	}
	return containerPorts, servicePorts
}

func composeServiceToSecret(composeService composeTypes.ServiceConfig, labels map[string]string) core.Secret {
	stringData := make(map[string]string)
	for key, value := range composeService.Environment {
		stringData[key] = *value
	}
	secret := core.Secret{}
	secret.APIVersion = "v1"
	secret.Kind = "Secret"
	secret.Name = composeService.Name + "-env"
	secret.Labels = labels
	secret.StringData = stringData
	return secret
}

func composeServiceToDeployment(
	composeService composeTypes.ServiceConfig,
	containerPorts []core.ContainerPort,
	volumes []core.Volume,
	volumeMounts []core.VolumeMount,
	secretName string,
	labels map[string]string) apps.Deployment {

	replicas := new(int32)
	*replicas = 1

	strategy := apps.DeploymentStrategy{}
	strategy.Type = apps.RecreateDeploymentStrategyType

	container := core.Container{
		Image:        composeService.Image,
		Name:         composeService.Name,
		Ports:        containerPorts,
		VolumeMounts: volumeMounts,
		// We COULD put the environment variables here, but because some of them likely contain sensitive data they are stored in a secret instead
		// Env:          envVars,
		// Reference the secret:
		EnvFrom: []core.EnvFromSource{
			core.EnvFromSource{
				SecretRef: &core.SecretEnvSource{
					LocalObjectReference: core.LocalObjectReference{
						Name: secretName,
					},
				},
			},
		},
	}

	podSpec := core.PodSpec{
		Containers:    []core.Container{container},
		Volumes:       volumes,
		RestartPolicy: core.RestartPolicyAlways,
	}

	templateSpec := core.PodTemplateSpec{
		Spec: podSpec,
	}

	deploymentSpec := apps.DeploymentSpec{
		Replicas: replicas,
		Strategy: strategy,
		Template: templateSpec,
	}

	deployment := apps.Deployment{}
	deployment.Spec = deploymentSpec
	deployment.APIVersion = "apps/v1"
	deployment.Kind = "Deployment"
	deployment.Name = composeService.Name
	deployment.Labels = labels

	return deployment
}

func composeServiceToService(composeService composeTypes.ServiceConfig, servicePorts []core.ServicePort, labels map[string]string) core.Service {
	serviceSpec := core.ServiceSpec{
		Ports:    servicePorts,
		Selector: labels,
	}
	service := core.Service{}
	service.Spec = serviceSpec
	service.APIVersion = "v1"
	service.Kind = "Service"
	service.Name = composeService.Name
	service.Labels = labels
	return service
}

func ComposeServiceToK8s(composeService composeTypes.ServiceConfig) (apps.Deployment, core.Service, []core.PersistentVolumeClaim, core.Secret) {
	labels := make(map[string]string)
	labels["k8ify.service"] = composeService.Name

	volumes, volumeMounts, persistentVolumeClaims := composeServiceVolumesToK8s(composeService.Name, composeService.Volumes, labels)
	containerPorts, servicePorts := composeServicePortsToK8s(composeService.Ports)
	secret := composeServiceToSecret(composeService, labels)
	deployment := composeServiceToDeployment(composeService, containerPorts, volumes, volumeMounts, secret.Name, labels)
	service := composeServiceToService(composeService, servicePorts, labels)

	return deployment, service, persistentVolumeClaims, secret
}
