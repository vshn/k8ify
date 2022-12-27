package main

import (
	"fmt"
	composeTypes "github.com/compose-spec/compose-go/types"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func composeServiceVolumesToVolumes(serviceName string, serviceVolumes []composeTypes.ServiceVolumeConfig) ([]core.Volume, []core.VolumeMount) {
	volumeMounts := []core.VolumeMount{}
	volumes := []core.Volume{}
	for i, volume := range serviceVolumes {
		name := fmt.Sprintf("%s-claim%d", serviceName, i)
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
	}
	return volumes, volumeMounts
}

func composeServicePortsToContainerPorts(composeServicePorts []composeTypes.ServicePortConfig) ([]core.ContainerPort, []core.ServicePort) {
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

func composeServiceEnvironmentToEnvVars(composeServiceEnvironmentMapping composeTypes.MappingWithEquals) []core.EnvVar {
	envVars := []core.EnvVar{}
	for key, value := range composeServiceEnvironmentMapping {
		envVars = append(envVars, core.EnvVar{
			Name:  key,
			Value: *value,
		})
	}
	return envVars
}

func composeServiceToDeploymentAndService(composeService composeTypes.ServiceConfig) (apps.Deployment, core.Service) {
	replicas := new(int32)
	*replicas = 1

	strategy := apps.DeploymentStrategy{}
	strategy.Type = apps.RecreateDeploymentStrategyType

	volumes, volumeMounts := composeServiceVolumesToVolumes(composeService.Name, composeService.Volumes)
	containerPorts, servicePorts := composeServicePortsToContainerPorts(composeService.Ports)
	envVars := composeServiceEnvironmentToEnvVars(composeService.Environment)

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

	return deployment, service
}
