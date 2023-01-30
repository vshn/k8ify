package internal

import (
	"fmt"
	composeTypes "github.com/compose-spec/compose-go/types"
	"log"
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
