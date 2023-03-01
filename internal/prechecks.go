package internal

import (
	"os"

	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
	"github.com/vshn/k8ify/pkg/ir"
)

const HLINE = "--------------------------------------------------------------------------------"

func ComposeServicePrecheck(composeService composeTypes.ServiceConfig) {
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
}

func VolumesPrecheck(inputs *ir.Inputs) {
	// Collect references to volumes
	references := make(map[string][]string)

	for _, service := range inputs.Services {
		for _, volumeName := range service.VolumeNames() {
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
			logrus.Errorf("WARNING: Volume %q has no size specified!", name)
		}

		// CHECK: Volume defined but not used in any services
		if len(references[name]) < 1 {
			logrus.Errorf("WARNING: Volume %q is defined but not referenced by any workloads", name)
			continue
		}

		// CHECK: Same non-shared volume on multiple services
		if !volume.IsShared() && len(references[name]) > 1 {
			logrus.Errorf("Volume %q is not marked as shared (via the `k8ify.shared` label on the volume), but is used by multiple services.", name)
			os.Exit(1)
		}
	}
}
