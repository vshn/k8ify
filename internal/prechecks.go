package internal

import (
	"fmt"
	"log"
	"os"

	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/pkg/ir"
)

var reset = "\033[0m"
var red = "\033[31m"
var hLine = "--------------------------------------------------------------------------------"

func logRed(line string) {
	log.Print(red + line + reset)
}

func ComposeServicePrecheck(composeService composeTypes.ServiceConfig) {
	if composeService.Deploy == nil || composeService.Deploy.Resources.Reservations == nil {
		logRed(hLine)
		logRed(fmt.Sprintf("  Service '%s' does not have any CPU/memory reservations defined.", composeService.Name))
		logRed("  k8ify can generate K8s manifests regardless, but your service will be")
		logRed("  unreliable or not work at all: It may not start at all, be slow to react")
		logRed("  due to insufficient CPU time or get OOM killed due to insufficient memory.")
		logRed("  Please specify CPU and memory reservations like this:")
		logRed("    services:")
		logRed(fmt.Sprintf("      %s:", composeService.Name))
		logRed("        deploy:")
		logRed("          resources:")
		logRed("            reservations:    # Minimum guaranteed by K8s to be always available")
		logRed("              cpus: \"0.2\"    # Number of CPU cores. Quotes are required!")
		logRed("              memory: 256M")
		logRed(hLine)
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
				logRed(fmt.Sprintf("Service %q references volume %q, which is not defined!", service.Name, volumeName))
				os.Exit(1)
			}

			// CHECK: Service is singleton but volume is not
			if service.IsSingleton() != volume.IsSingleton() {
				logRed(fmt.Sprintf("Service %q, Volume %q: `k8ify.singleton` labels must be identical", service.Name, volumeName))
				os.Exit(1)
			}

			references[volumeName] = append(references[volumeName], service.Name)
		}
	}

	for name, volume := range inputs.Volumes {
		// CHECK: No size defined
		if volume.SizeIsMissing() {
			logRed(fmt.Sprintf("WARNING: Volume %q has no size specified!", name))
		}

		// CHECK: Volume defined but not used in any services
		if len(references[name]) < 1 {
			logRed(fmt.Sprintf("WARNING: Volume %q is defined but not referenced by any workloads", name))
			continue
		}

		// CHECK: Same non-shared volume on multiple services
		if !volume.IsShared() && len(references[name]) > 1 {
			logRed(fmt.Sprintf("Volume %q is not marked as shared (via the `k8ify.shared` label on the volume),", name))
			logRed("but is used by multiple services.")
			os.Exit(1)
		}
	}
}
