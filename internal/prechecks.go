package internal

import (
	"os"

	"github.com/vshn/k8ify/pkg/util"

	"github.com/sirupsen/logrus"
	"github.com/vshn/k8ify/pkg/ir"
)

const HLINE = "--------------------------------------------------------------------------------"

func ComposeServicePrecheck(inputs *ir.Inputs) {
	for _, service := range inputs.Services {
		composeService := service.AsCompose()
		if composeService.Deploy == nil || composeService.Deploy.Resources.Reservations == nil {
			logrus.Error(HLINE)
			logrus.Errorf("  Service '%s' does not have any CPU/memory reservations defined.", composeService.Name)
			logrus.Error("  k8ify can generate K8s manifests regardless, but your service will be")
			logrus.Error("  unreliable or not work at all: It may not start at all, be slow to react")
			logrus.Error("  due to insufficient CPU time or get OOM killed due to insufficient memory.")
			logrus.Error("  Please specify CPU and memory reservations like this:")
			logrus.Error("    services:")
			logrus.Errorf("      %s:", composeService.Name)
			logrus.Error("        deploy:")
			logrus.Error("          resources:")
			logrus.Error("            reservations:    # Minimum guaranteed by K8s to be always available")
			logrus.Error(`              cpus: "0.2"    # Number of CPU cores. Quotes are required!`)
			logrus.Error("              memory: 256M")
			logrus.Error(HLINE)
		}
		parentSingleton := util.IsSingleton(composeService.Labels)
		for _, part := range service.GetParts() {
			partSingleton := util.IsSingleton(part.AsCompose().Labels)
			if partSingleton && !parentSingleton {
				logrus.Errorf("Singleton compose service '%s' can't be part of non-singleton compose service '%s'", part.Name, service.Name)
				os.Exit(1)
			}
			if !partSingleton && parentSingleton {
				logrus.Errorf("Non-singleton compose service '%s' can't be part of singleton compose service '%s'", part.Name, service.Name)
				os.Exit(1)
			}
		}
		environmentValues := service.AsCompose().Environment
		for key, value := range environmentValues {
			if value == nil {
				logrus.Warnf("Service '%s' has environment variable '%s' with value nil. There may be a problem with your compose file(s). Please use empty string \"\" values instead.", service.Name, key)
			}
		}
		serviceMonitorConfig := ir.ServiceMonitorConfigPointer(service.Labels())
		if serviceMonitorConfig != nil {
			_, err := ir.ServiceMonitorBasicAuthConfigPointer(service.Labels())
			if err != nil {
				logrus.Error(err.Error())
				os.Exit(1)
			}
		}
	}
}

func VolumesPrecheck(inputs *ir.Inputs) {
	// Collect references to volumes
	references := make(map[string][]string)

	for _, service := range inputs.Services {

		// conditions must be met not only for volumes of the parent but also all volumes of the parts
		allVolumes := make(map[string]bool) // set semantics (eliminate duplicates)
		for _, volumeName := range service.VolumeNames() {
			allVolumes[volumeName] = true
		}
		for _, part := range service.GetParts() {
			for _, volumeName := range part.VolumeNames() {
				allVolumes[volumeName] = true
			}
		}

		for volumeName := range allVolumes {
			volume, ok := inputs.Volumes[volumeName]

			// CHECK: Volume does not exist
			if !ok {
				logrus.Errorf("Service %q references volume %q, which is not defined!", service.Name, volumeName)
				os.Exit(1)
			}

			// CHECK: Service is singleton but volume is not
			if service.IsSingleton() != volume.IsSingleton() {
				logrus.Errorf("Service %q, Volume %q: `k8ify.singleton` labels must be identical", service.Name, volumeName)
				os.Exit(1)
			}

			references[volumeName] = append(references[volumeName], service.Name)
		}
	}

	for name, volume := range inputs.Volumes {
		// CHECK: No size defined
		if volume.SizeIsMissing() {
			logrus.Warnf("Volume %q has no size specified!", name)
		}

		// CHECK: Volume defined but not used in any services
		if len(references[name]) < 1 {
			logrus.Warnf("Volume %q is defined but not referenced by any workloads", name)
			continue
		}

		// CHECK: Same non-shared volume on multiple services
		if !volume.IsShared() && len(references[name]) > 1 {
			logrus.Errorf("Volume %q is not marked as shared (via the `k8ify.shared` label on the volume), but is used by multiple services.", name)
			os.Exit(1)
		}
	}
}

func DomainLengthPrecheck(inputs *ir.Inputs) {
	maxExposeLength := inputs.TargetCfg.MaxExposeLength()
	for _, service := range inputs.Services {
		for _, domain := range util.SubConfig(service.Labels(), "k8ify.expose", "default") {
			if !inputs.TargetCfg.IsSubdomainOfAppsDomain(domain) && len(domain) > maxExposeLength {
				logrus.Errorf("Service '%s' is supposed to be exposed on domain '%s' which is longer than %d characters. This likely won't work due to certificate common name length restrictions.", service.Name, domain, maxExposeLength)
				logrus.Errorf("To fix this you can use the cluster's appsDomain wildcard certificate (compose file option 'x-targetCfg.appsDomain') or adjust this check ('x-targetCfg.maxExposeLength').")
				os.Exit(1)
			}
		}
	}
}
