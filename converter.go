package main

import (
	"fmt"
	composeTypes "github.com/compose-spec/compose-go/types"
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

func composeServiceVolumesToK8s(serviceName string, serviceVolumes []composeTypes.ServiceVolumeConfig) ([]core.Volume, []core.VolumeMount, []core.PersistentVolumeClaim) {
	volumeMounts := []core.VolumeMount{}
	volumes := []core.Volume{}
	persistentVolumeClaims := []core.PersistentVolumeClaim{}
	for i, volume := range serviceVolumes {
		name := sanitize(volume.Source)
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
			TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
			ObjectMeta: metav1.ObjectMeta{Name: name},
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

func composeServiceEnvironmentToK8s(composeServiceEnvironmentMapping composeTypes.MappingWithEquals) []core.EnvVar {
	envVars := []core.EnvVar{}
	for key, value := range composeServiceEnvironmentMapping {
		envVars = append(envVars, core.EnvVar{
			Name:  key,
			Value: *value,
		})
	}
	return envVars
}

func composeServiceToK8s(composeService composeTypes.ServiceConfig) (apps.Deployment, core.Service, []core.PersistentVolumeClaim) {
	replicas := new(int32)
	*replicas = 1

	strategy := apps.DeploymentStrategy{}
	strategy.Type = apps.RecreateDeploymentStrategyType

	volumes, volumeMounts, persistentVolumeClaims := composeServiceVolumesToK8s(composeService.Name, composeService.Volumes)
	containerPorts, servicePorts := composeServicePortsToK8s(composeService.Ports)
	envVars := composeServiceEnvironmentToK8s(composeService.Environment)

	container := core.Container{
		Image:        composeService.Image,
		Name:         composeService.Name,
		Ports:        containerPorts,
		VolumeMounts: volumeMounts,
		Env:          envVars,
	}

	podSpec := core.PodSpec{
		Containers: []core.Container{container},
		Volumes:    volumes,
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

	serviceSpec := core.ServiceSpec{
		Ports: servicePorts,
	}

	service := core.Service{}
	service.Spec = serviceSpec
	service.APIVersion = "v1"
	service.Kind = "Service"
	service.Name = composeService.Name

	return deployment, service, persistentVolumeClaims
}
