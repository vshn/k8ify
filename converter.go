package main

import (
	"fmt"
	composeTypes "github.com/compose-spec/compose-go/types"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
)

func serviceVolumesToVolumes(serviceName string, serviceVolumes []composeTypes.ServiceVolumeConfig) ([]core.Volume, []core.VolumeMount) {
	volumeMounts := []core.VolumeMount{}
	volumes := []core.Volume{}
	for i, volume := range serviceVolumes {
		name := fmt.Sprintf("%s-claim%d", serviceName, i)
		volumeMounts = append(volumeMounts, core.VolumeMount{
			MountPath: volume.Target,
			Name: name,
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

func servicePortsToContainerPorts(servicePorts []composeTypes.ServicePortConfig) []core.ContainerPort {
	containerPorts := []core.ContainerPort{}
	for _, port := range servicePorts {
		containerPorts = append(containerPorts, core.ContainerPort{
			ContainerPort: int32(port.Target),
		})
	}
	return containerPorts
}

func serviceEnvironmentToEnvVars(serviceEnvironmentMapping composeTypes.MappingWithEquals) []core.EnvVar {
	envVars := []core.EnvVar{}
	for key, value := range serviceEnvironmentMapping {
		envVars = append(envVars, core.EnvVar{
			Name: key,
			Value: *value,
		})
	}
	return envVars
}

func serviceToDeployment(service composeTypes.ServiceConfig) apps.Deployment {
	replicas := new(int32)
	*replicas = 1

	strategy := apps.DeploymentStrategy{}
	strategy.Type = apps.RecreateDeploymentStrategyType

	volumes, volumeMounts := serviceVolumesToVolumes(service.Name, service.Volumes)
	containerPorts := servicePortsToContainerPorts(service.Ports)
	envVars := serviceEnvironmentToEnvVars(service.Environment)

	container := core.Container{
		Image: service.Image,
		Name: service.Name,
		Ports: containerPorts,
		VolumeMounts: volumeMounts,
		Env: envVars,
	}

	podSpec := core.PodSpec{
		Containers: []core.Container{container},
		Volumes: volumes,
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
	deployment.Name = service.Name

	return deployment
}
