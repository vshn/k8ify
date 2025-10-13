package networkpolicy

import (
	"fmt"

	"github.com/vshn/k8ify/pkg/ir"
	"github.com/vshn/k8ify/pkg/provider/targetconfigs"
	"github.com/vshn/k8ify/pkg/util"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func CreateNetworkPoliciesForExposedPorts(targetCfg ir.TargetCfg, refSlug string, workload *ir.ParentService, labels map[string]string, servicePorts []v1.ServicePort) []unstructured.Unstructured {
	var networkPolicies []unstructured.Unstructured
	if scheme, ok := targetCfg[targetconfigs.ExposePlainLoadBalancerSchemeKey]; !ok || scheme != targetconfigs.AppuioCloudscale {
		return networkPolicies
	}

	for _, servicePort := range servicePorts {
		serviceType := util.ServiceType(workload.Labels(), servicePort.Port)
		if serviceType == v1.ServiceTypeLoadBalancer {
			networkPolicy := createCiliumNetworkPolicy(refSlug, workload, labels, servicePort.Port)
			networkPolicies = append(networkPolicies, networkPolicy)
		}
	}
	return networkPolicies
}

func createCiliumNetworkPolicy(refSlug string, workload *ir.ParentService, labels map[string]string, port int32) unstructured.Unstructured {
	portNrAsString := fmt.Sprint(port)
	return unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":   workload.Name + refSlug + "-" + portNrAsString + "-allow-from-world",
				"labels": labels,
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": labels,
				},
				"ingress": []map[string]interface{}{
					{
						"fromEntities": []string{
							"world",
						},
						"toPorts": []map[string]interface{}{
							{
								"ports": []map[string]interface{}{
									{
										"port":     portNrAsString,
										"protocol": "ANY",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
