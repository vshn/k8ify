package converter

import (
	"fmt"
	"log"
	"maps"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/sirupsen/logrus"
	"github.com/vshn/k8ify/pkg/ir"
	"github.com/vshn/k8ify/pkg/util"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

var (
	SecretRefMagic = "ylkBUFN0o29yr4yLCTUZqzgIT6qCIbyj" // magic string to indicate that what follows isn't a value but a reference to a secret
)

func composeServiceVolumesToK8s(
	refSlug string,
	mounts []composeTypes.ServiceVolumeConfig,
	projectVolumes map[string]*ir.Volume,
) (map[string]core.Volume, []core.VolumeMount) {

	volumeMounts := []core.VolumeMount{}
	volumes := make(map[string]core.Volume)

	for _, mount := range mounts {
		if mount.Type != "volume" {
			continue
		}
		name := util.Sanitize(mount.Source)

		volumeMounts = append(volumeMounts, core.VolumeMount{
			MountPath: mount.Target,
			Name:      name,
		})

		volume := projectVolumes[mount.Source]
		if volume.IsShared() {
			volumes[name] = core.Volume{
				Name: name,
				VolumeSource: core.VolumeSource{
					PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
						ClaimName: mount.Source + refSlug,
					},
				},
			}
		}
	}
	return volumes, volumeMounts
}

func composeServicePortsToK8sServicePorts(workload *ir.Service) []core.ServicePort {
	servicePorts := []core.ServicePort{}
	ports := workload.GetPorts()
	// the single k8s service contains the ports of all parts
	for _, part := range workload.GetParts() {
		ports = append(ports, part.GetPorts()...)
	}
	for _, port := range ports {
		servicePorts = append(servicePorts, core.ServicePort{
			Name: fmt.Sprint(port.ServicePort),
			Port: int32(port.ServicePort),
			TargetPort: intstr.IntOrString{
				IntVal: int32(port.ContainerPort),
			},
		})
	}
	return servicePorts
}

func composeServicePortsToK8sContainerPorts(workload *ir.Service) []core.ContainerPort {
	containerPorts := []core.ContainerPort{}
	for _, port := range workload.AsCompose().Ports {
		containerPorts = append(containerPorts, core.ContainerPort{
			ContainerPort: int32(port.Target),
		})
	}
	return containerPorts
}

func composeServiceToSecret(workload *ir.Service, refSlug string, labels map[string]string) *core.Secret {
	stringData := make(map[string]string)
	for key, value := range workload.AsCompose().Environment {
		if value != nil && strings.HasPrefix(*value, SecretRefMagic+":") {
			// we've encountered a reference to another secret (starting with "$_ref_:" in the compose file), ignore
			continue
		}
		if value == nil {
			stringData[key] = ""
		} else {
			stringData[key] = *value
		}
	}
	if len(stringData) == 0 {
		return nil
	}
	secret := core.Secret{}
	secret.APIVersion = "v1"
	secret.Kind = "Secret"
	secret.Name = workload.Name + refSlug + "-env"
	secret.Labels = labels
	secret.Annotations = util.Annotations(workload.Labels(), "Secret")
	secret.StringData = stringData
	return &secret
}

func composeServiceToDeployment(
	workload *ir.Service,
	refSlug string,
	projectVolumes map[string]*ir.Volume,
	labels map[string]string,
) (apps.Deployment, []core.Secret) {

	deployment := apps.Deployment{}
	deployment.APIVersion = "apps/v1"
	deployment.Kind = "Deployment"
	deployment.Name = workload.AsCompose().Name + refSlug
	deployment.Labels = labels
	deployment.Annotations = util.Annotations(workload.Labels(), "Deployment")

	templateSpec, secrets := composeServiceToPodTemplate(
		workload,
		refSlug,
		projectVolumes,
		labels,
		util.ServiceAccountName(workload.AsCompose().Labels),
	)

	deployment.Spec = apps.DeploymentSpec{
		Replicas: composeServiceToReplicas(workload.AsCompose()),
		Strategy: composeServiceToStrategy(workload.AsCompose()),
		Template: templateSpec,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
	}

	return deployment, secrets
}

func composeServiceToStrategy(composeService composeTypes.ServiceConfig) apps.DeploymentStrategy {
	order := getUpdateOrder(composeService)
	var typ apps.DeploymentStrategyType

	switch order {
	case "start-first":
		typ = apps.RollingUpdateDeploymentStrategyType
	default:
		typ = apps.RecreateDeploymentStrategyType
	}

	return apps.DeploymentStrategy{
		Type: typ,
	}
}

func getUpdateOrder(composeService composeTypes.ServiceConfig) string {
	if composeService.Deploy == nil || composeService.Deploy.UpdateConfig == nil {
		return "stop-first"
	}
	return composeService.Deploy.UpdateConfig.Order
}

func composeServiceToStatefulSet(
	workload *ir.Service,
	refSlug string,
	projectVolumes map[string]*ir.Volume,
	volumeClaims []core.PersistentVolumeClaim,
	labels map[string]string,
) (apps.StatefulSet, []core.Secret) {

	statefulset := apps.StatefulSet{}
	statefulset.APIVersion = "apps/v1"
	statefulset.Kind = "StatefulSet"
	statefulset.Name = workload.AsCompose().Name + refSlug
	statefulset.Labels = labels
	statefulset.Annotations = util.Annotations(workload.Labels(), "StatefulSet")

	templateSpec, secrets := composeServiceToPodTemplate(
		workload,
		refSlug,
		projectVolumes,
		labels,
		util.ServiceAccountName(workload.AsCompose().Labels),
	)

	statefulset.Spec = apps.StatefulSetSpec{
		Replicas: composeServiceToReplicas(workload.AsCompose()),
		Template: templateSpec,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		VolumeClaimTemplates: volumeClaims,
	}

	return statefulset, secrets
}

func composeServiceToReplicas(composeService composeTypes.ServiceConfig) *int32 {
	deploy := composeService.Deploy
	if deploy == nil || deploy.Replicas == nil {
		return nil
	}
	// deploy.Replicas is an Uint64, but if you have over 2'000'000'000
	// replicas, you might have different problems :)
	return ptr.To(int32(*deploy.Replicas))
}

func composeServiceToPodTemplate(
	workload *ir.Service,
	refSlug string,
	projectVolumes map[string]*ir.Volume,
	labels map[string]string,
	serviceAccountName string,
) (core.PodTemplateSpec, []core.Secret) {
	container, secret, volumes := composeServiceToContainer(workload, refSlug, projectVolumes, labels)
	containers := []core.Container{container}
	secrets := []core.Secret{}
	if secret != nil {
		secrets = append(secrets, *secret)
	}

	for _, part := range workload.GetParts() {
		c, s, cvs := composeServiceToContainer(part, refSlug, projectVolumes, labels)
		containers = append(containers, c)
		if s != nil {
			secrets = append(secrets, *s)
		}
		maps.Copy(volumes, cvs)
	}

	// make sure the array is sorted to have deterministic output
	keys := make([]string, 0, len(volumes))
	for key := range volumes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	volumesArray := []core.Volume{}
	for _, key := range keys {
		volumesArray = append(volumesArray, volumes[key])
	}

	podSpec := core.PodSpec{
		Containers:         containers,
		RestartPolicy:      core.RestartPolicyAlways,
		Volumes:            volumesArray,
		ServiceAccountName: serviceAccountName,
	}

	return core.PodTemplateSpec{
		Spec: podSpec,
		ObjectMeta: metav1.ObjectMeta{
			Labels:      labels,
			Annotations: util.Annotations(workload.Labels(), "Pod"),
		},
	}, secrets
}

func composeServiceToContainer(
	workload *ir.Service,
	refSlug string,
	projectVolumes map[string]*ir.Volume,
	labels map[string]string,
) (core.Container, *core.Secret, map[string]core.Volume) {
	composeService := workload.AsCompose()
	volumes, volumeMounts := composeServiceVolumesToK8s(
		refSlug, workload.AsCompose().Volumes, projectVolumes,
	)
	livenessProbe, readinessProbe, startupProbe := composeServiceToProbes(workload)
	containerPorts := composeServicePortsToK8sContainerPorts(workload)
	resources := composeServiceToResourceRequirements(composeService)
	secret := composeServiceToSecret(workload, refSlug, labels)
	envFrom := []core.EnvFromSource{}
	if secret != nil {
		envFrom = append(envFrom, core.EnvFromSource{SecretRef: &core.SecretEnvSource{LocalObjectReference: core.LocalObjectReference{Name: secret.Name}}})
	}
	env := []core.EnvVar{}
	for key, value := range workload.AsCompose().Environment {
		if value != nil && strings.HasPrefix(*value, SecretRefMagic+":") {
			// we've encountered a reference to another secret (starting with "$_ref_:" in the compose file)
			refValue := (*value)[len(SecretRefMagic)+1:]
			refStrings := strings.SplitN(refValue, ":", 2)
			if len(refStrings) != 2 {
				logrus.Warnf("Secret reference '$_ref_:%s' has invalid format, should be '$_ref_:SECRETNAME:KEY'. Ignoring.", refValue)
				continue
			}
			env = append(env, core.EnvVar{Name: key, ValueFrom: &core.EnvVarSource{SecretKeyRef: &core.SecretKeySelector{LocalObjectReference: core.LocalObjectReference{Name: refStrings[0]}, Key: refStrings[1]}}})
		}
	}
	return core.Container{
		Name:  composeService.Name + refSlug,
		Image: composeService.Image,
		Ports: containerPorts,
		// We COULD put the environment variables here, but because some of them likely contain sensitive data they are stored in a secret instead
		// Env:          envVars,
		// Reference the secret:
		EnvFrom:         envFrom,
		Env:             env,
		VolumeMounts:    volumeMounts,
		LivenessProbe:   livenessProbe,
		ReadinessProbe:  readinessProbe,
		StartupProbe:    startupProbe,
		Resources:       resources,
		Command:         composeService.Entrypoint, // ENTRYPOINT in Docker == 'entrypoint' in Compose == 'command' in K8s
		Args:            composeService.Command,    // CMD in Docker == 'command' in Compose == 'args' in K8s
		ImagePullPolicy: core.PullAlways,
	}, secret, volumes
}

func serviceSpecToService(refSlug string, workload *ir.Service, serviceSpec core.ServiceSpec, labels map[string]string) core.Service {
	serviceName := workload.Name + refSlug
	// We only add the port numbers to the service name if the ports are exposed directly. This is to ensure backwards compatibility with previous versions of k8ify and to keep things neat (not many people will need to expose ports directly).
	if !serviceSpecIsUnexposedDefault(serviceSpec) {
		for _, port := range serviceSpec.Ports {
			serviceName = fmt.Sprintf("%s-%d", serviceName, port.Port)
		}
	}
	service := core.Service{}
	service.Spec = serviceSpec
	service.APIVersion = "v1"
	service.Kind = "Service"
	service.Name = serviceName
	service.Labels = labels
	service.Annotations = util.Annotations(workload.Labels(), "Service")
	return service
}

// PortConfig only exists to be used as a map key (we can't use core.ServiceSpec)
type PortConfig struct {
	Type                  core.ServiceType
	ExternalTrafficPolicy core.ServiceExternalTrafficPolicy
	HealthCheckNodePort   int32
}

func serviceSpecIsUnexposedDefault(serviceSpec core.ServiceSpec) bool {
	return serviceSpec.Type == "" && serviceSpec.ExternalTrafficPolicy == "" && serviceSpec.HealthCheckNodePort == 0
}

func composeServiceToServices(refSlug string, workload *ir.Service, servicePorts []core.ServicePort, labels map[string]string) []core.Service {
	var services []core.Service
	serviceSpecs := map[PortConfig]core.ServiceSpec{}

	for _, servicePort := range servicePorts {
		portConfig := PortConfig{
			Type:                  util.ServiceType(workload.Labels(), servicePort.Port),
			ExternalTrafficPolicy: util.ServiceExternalTrafficPolicy(workload.Labels(), servicePort.Port),
			HealthCheckNodePort:   util.ServiceHealthCheckNodePort(workload.Labels(), servicePort.Port),
		}
		spec, specExists := serviceSpecs[portConfig]
		if specExists {
			spec.Ports = append(serviceSpecs[portConfig].Ports, servicePort)
			serviceSpecs[portConfig] = spec
		} else {
			serviceSpecs[portConfig] = core.ServiceSpec{
				Selector:              labels,
				Type:                  portConfig.Type,
				ExternalTrafficPolicy: portConfig.ExternalTrafficPolicy,
				HealthCheckNodePort:   portConfig.HealthCheckNodePort,
				Ports:                 []core.ServicePort{servicePort},
			}
		}
	}

	for _, serviceSpec := range serviceSpecs {
		services = append(services, serviceSpecToService(refSlug, workload, serviceSpec, labels))
	}

	return services
}

func composeServiceToIngress(workload *ir.Service, refSlug string, services []core.Service, labels map[string]string, targetCfg ir.TargetCfg) *networking.Ingress {
	var service *core.Service
	for _, s := range services {
		if serviceSpecIsUnexposedDefault(s.Spec) {
			service = &s
		}
	}
	if service == nil {
		return nil
	}

	workloads := []*ir.Service{workload}
	workloads = append(workloads, workload.GetParts()...)

	var ingressRules []networking.IngressRule
	var ingressTLSs []networking.IngressTLS

	for _, w := range workloads {
		for i, port := range w.GetPorts() {
			// we expect the config to be in "k8ify.expose.PORT"
			configPrefix := fmt.Sprintf("k8ify.expose.%d", port.ServicePort)
			ingressConfig := util.SubConfig(w.Labels(), configPrefix, "host")
			if _, ok := ingressConfig["host"]; !ok && i == 0 {
				// for the first port we also accept config in "k8ify.expose"
				ingressConfig = util.SubConfig(w.Labels(), "k8ify.expose", "host")
			}

			if host, ok := ingressConfig["host"]; ok {
				serviceBackendPort := networking.ServiceBackendPort{
					Number: int32(port.ServicePort),
				}

				ingressServiceBackend := networking.IngressServiceBackend{
					Name: service.Name,
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

				ingressRules = append(ingressRules, networking.IngressRule{
					Host:             host,
					IngressRuleValue: ingressRuleValue,
				})

				if targetCfg.IsSubdomainOfAppsDomain(host) {
					// special case: With an empty TLS configuration the ingress uses the cluster-wide apps domain wildcard certificate
					ingressTLSs = append(ingressTLSs, networking.IngressTLS{})
				} else {
					ingressTLSs = append(ingressTLSs, networking.IngressTLS{
						Hosts:      []string{host},
						SecretName: workload.Name + refSlug,
					})
				}
			}
		}
	}

	if len(ingressRules) == 0 {
		return nil
	}

	ingress := networking.Ingress{}
	ingress.APIVersion = "networking.k8s.io/v1"
	ingress.Kind = "Ingress"
	ingress.Name = workload.Name + refSlug
	ingress.Labels = labels
	ingress.Annotations = util.Annotations(workload.Labels(), "Ingress")
	ingress.Spec = networking.IngressSpec{
		Rules: ingressRules,
		TLS:   ingressTLSs,
	}

	return &ingress
}

func composeServiceToProbe(config map[string]string, port intstr.IntOrString) *core.Probe {
	if enabledStr, ok := config["enabled"]; ok {
		if !util.IsTruthy(enabledStr) {
			return nil
		}
	}

	path := ""
	if pathStr, ok := config["path"]; ok {
		path = pathStr
	}

	scheme := core.URISchemeHTTP
	if schemeStr, ok := config["scheme"]; ok {
		if schemeStr == "HTTPS" || schemeStr == "https" {
			scheme = core.URISchemeHTTPS
		}
	}

	periodSeconds := util.ConfigGetInt32(config, "periodSeconds", 30)
	timeoutSeconds := util.ConfigGetInt32(config, "timeoutSeconds", 60)
	initialDelaySeconds := util.ConfigGetInt32(config, "initialDelaySeconds", 0)
	successThreshold := util.ConfigGetInt32(config, "successThreshold", 1)
	failureThreshold := util.ConfigGetInt32(config, "failureThreshold", 3)

	probeHandler := core.ProbeHandler{}
	if path == "" {
		probeHandler.TCPSocket = &core.TCPSocketAction{
			Port: port,
		}
	} else {
		probeHandler.HTTPGet = &core.HTTPGetAction{
			Path:   path,
			Port:   port,
			Scheme: scheme,
		}
	}

	return &core.Probe{
		ProbeHandler:        probeHandler,
		PeriodSeconds:       periodSeconds,
		TimeoutSeconds:      timeoutSeconds,
		InitialDelaySeconds: initialDelaySeconds,
		SuccessThreshold:    successThreshold,
		FailureThreshold:    failureThreshold,
	}
}

func composeServiceToProbes(workload *ir.Service) (*core.Probe, *core.Probe, *core.Probe) {
	composeService := workload.AsCompose()
	if len(composeService.Ports) == 0 {
		return nil, nil, nil
	}
	port := intstr.IntOrString{IntVal: int32(composeService.Ports[0].Target)}
	livenessConfig := util.SubConfig(composeService.Labels, "k8ify.liveness", "path")
	readinessConfig := util.SubConfig(composeService.Labels, "k8ify.readiness", "path")
	startupConfig := util.SubConfig(composeService.Labels, "k8ify.startup", "path")

	// Protect application from overly eager livenessProbe during startup while keeping the startup fast.
	// By default the startupProbe is the same as the livenessProbe except for periodSeconds and failureThreshold
	for k, v := range livenessConfig {
		if _, ok := startupConfig[k]; !ok {
			startupConfig[k] = v
		}
	}
	if _, ok := startupConfig["periodSeconds"]; !ok {
		startupConfig["periodSeconds"] = "10"
	}
	if _, ok := startupConfig["failureThreshold"]; !ok {
		startupConfig["failureThreshold"] = "30" // will try for a total of 300s
	}

	// By default the readinessProbe is disabled.
	if len(readinessConfig) == 0 {
		readinessConfig["enabled"] = "false"
	}

	livenessProbe := composeServiceToProbe(livenessConfig, port)
	readinessProbe := composeServiceToProbe(readinessConfig, port)
	startupProbe := composeServiceToProbe(startupConfig, port)
	return livenessProbe, readinessProbe, startupProbe
}

func composeServiceToResourceRequirements(composeService composeTypes.ServiceConfig) core.ResourceRequirements {
	requestsMap := core.ResourceList{}
	limitsMap := core.ResourceList{}

	if composeService.Deploy != nil {
		if composeService.Deploy.Resources.Reservations != nil {
			// NanoCPU appears to be a misnomer, it's actually a float indicating the number of CPU cores, nothing 'nano'
			cpuRequest, err := strconv.ParseFloat(composeService.Deploy.Resources.Reservations.NanoCPUs, 64)
			if err == nil && cpuRequest > 0 {
				requestsMap["cpu"] = resource.MustParse(fmt.Sprintf("%f", cpuRequest))
				limitsMap["cpu"] = resource.MustParse(fmt.Sprintf("%f", cpuRequest*10.0))
			}
			memRequest := composeService.Deploy.Resources.Reservations.MemoryBytes
			if memRequest > 0 {
				requestsMap["memory"] = resource.MustParse(fmt.Sprintf("%dMi", memRequest/1024/1024))
				limitsMap["memory"] = resource.MustParse(fmt.Sprintf("%dMi", memRequest/1024/1024))
			}
		}
		if composeService.Deploy.Resources.Limits != nil {
			// If there are explicit limits configured we ignore the defaults calculated from the requests
			limitsMap = core.ResourceList{}
			cpuLimit, err := strconv.ParseFloat(composeService.Deploy.Resources.Limits.NanoCPUs, 64)
			if err == nil && cpuLimit > 0 {
				limitsMap["cpu"] = resource.MustParse(fmt.Sprintf("%f", cpuLimit))
			}
			memLimit := composeService.Deploy.Resources.Limits.MemoryBytes
			if memLimit > 0 {
				limitsMap["memory"] = resource.MustParse(fmt.Sprintf("%dMi", memLimit/1024/1024))
			}
		}
	}

	resources := core.ResourceRequirements{
		Requests: requestsMap,
		Limits:   limitsMap,
	}
	return resources
}

func toRefSlug(ref string, resource WithLabels) string {
	if ref == "" || util.IsSingleton(resource.Labels()) {
		return ""
	}

	return ref
}

type WithLabels interface {
	Labels() map[string]string
}

func CallExternalConverter(resourceName string, options map[string]string) (unstructured.Unstructured, error) {
	args := []string{resourceName}
	for k, v := range options {
		if k != "cmd" {
			args = append(args, "--"+k, v)
		}
	}
	cmd := exec.Command(options["cmd"], args...)
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		for _, line := range strings.Split(string(output), "\n") {
			logrus.Error(line)
		}
		return unstructured.Unstructured{}, fmt.Errorf("Could not convert service '%s' using command '%s': %w", resourceName, options["cmd"], err)
	}
	otherResource := unstructured.Unstructured{}
	err = yaml.Unmarshal(output, &otherResource)
	if err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("Could not convert service '%s' using command '%s': %w", resourceName, options["cmd"], err)
	}
	return otherResource, nil
}

func ComposeServiceToK8s(ref string, workload *ir.Service, projectVolumes map[string]*ir.Volume, targetCfg ir.TargetCfg) Objects {
	refSlug := toRefSlug(util.SanitizeWithMinLength(ref, 4), workload)
	labels := make(map[string]string)
	labels["k8ify.service"] = workload.Name
	if refSlug != "" {
		labels["k8ify.ref-slug"] = refSlug
		refSlug = "-" + refSlug
	}

	objects := Objects{}

	if util.Converter(workload.Labels()) != nil {
		otherResource, err := CallExternalConverter(workload.Name+refSlug, util.SubConfig(workload.Labels(), "k8ify.converter", "cmd"))
		if err != nil {
			log.Fatal(err)
		}
		if otherResource.GetLabels() == nil {
			otherResource.SetLabels(labels)
		} else {
			for k, v := range labels {
				otherResource.GetLabels()[k] = v
			}
		}
		annotations := util.Annotations(workload.Labels(), otherResource.GetKind())
		if otherResource.GetAnnotations() == nil {
			otherResource.SetAnnotations(annotations)
		} else {
			for k, v := range annotations {
				otherResource.GetAnnotations()[k] = v
			}
		}
		objects.Others = append([]unstructured.Unstructured{}, otherResource)
		return objects
	}

	servicePorts := composeServicePortsToK8sServicePorts(workload)
	objects.Services = composeServiceToServices(refSlug, workload, servicePorts, labels)

	// Find volumes used by this service and all its parts
	rwoVolumes, rwxVolumes := workload.Volumes(projectVolumes)
	for _, part := range workload.GetParts() {
		rwoV, rwxV := part.Volumes(projectVolumes)
		maps.Copy(rwoVolumes, rwoV)
		maps.Copy(rwxVolumes, rwxV)
	}

	// All shared (rwx) volumes used by the service, no matter if the service is a StatefulSet or a Deployment, must be
	// turned into PersistentVolumeClaims. Note that since these volumes are shared, the same PersistentVolumeClaim might
	// be generated by multiple compose services. Objects.Append() takes care of deduplication.
	pvcs := []core.PersistentVolumeClaim{}
	for _, vol := range rwxVolumes {
		pvcs = append(pvcs, ComposeSharedVolumeToK8s(ref, vol))
	}
	objects.PersistentVolumeClaims = pvcs

	if len(rwoVolumes) > 0 {
		// rwo volumes mean that we can only have one instance of the service, hence StatefulSet is the right choice.
		// Technically we might have multiple instances with a StatefulSet but then every instance gets its own volume,
		// ensuring that each volume remains rwo
		pvcs := []core.PersistentVolumeClaim{}
		for _, vol := range rwoVolumes {
			pvcs = append(pvcs, composeVolumeToPvc(vol.Name, labels, vol))
		}

		statefulset, secrets := composeServiceToStatefulSet(
			workload,
			refSlug,
			projectVolumes,
			pvcs,
			labels,
		)
		objects.StatefulSets = []apps.StatefulSet{statefulset}
		objects.Secrets = secrets
	} else {
		deployment, secrets := composeServiceToDeployment(
			workload,
			refSlug,
			projectVolumes,
			labels,
		)
		objects.Deployments = []apps.Deployment{deployment}
		objects.Secrets = secrets
	}

	ingress := composeServiceToIngress(workload, refSlug, objects.Services, labels, targetCfg)
	if ingress == nil {
		objects.Ingresses = []networking.Ingress{}
	} else {
		objects.Ingresses = []networking.Ingress{*ingress}
	}

	return objects
}

func ComposeSharedVolumeToK8s(ref string, volume *ir.Volume) core.PersistentVolumeClaim {
	refSlug := toRefSlug(util.SanitizeWithMinLength(ref, 4), volume)
	labels := make(map[string]string)
	labels["k8ify.volume"] = volume.Name
	if refSlug != "" {
		labels["k8ify.ref-slug"] = refSlug
		refSlug = "-" + refSlug
	}
	name := volume.Name + refSlug
	pvc := composeVolumeToPvc(name, labels, volume)

	return pvc
}

func composeVolumeToPvc(name string, labels map[string]string, volume *ir.Volume) core.PersistentVolumeClaim {
	name = util.Sanitize(name)
	accessMode := core.ReadWriteOnce
	if volume.IsShared() {
		accessMode = core.ReadWriteMany
	}

	return core.PersistentVolumeClaim{
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
				Requests: core.ResourceList{
					"storage": volume.Size("1G"),
				},
			},
			StorageClassName: util.StorageClass(volume.Labels()),
		},
	}
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
	Others                 []unstructured.Unstructured
}

func (this Objects) Append(other Objects) Objects {
	// Merge PVCs while avoiding duplicates based on the name
	pvcs := this.PersistentVolumeClaims
	nameSet := make(map[string]bool)
	for _, pvc := range pvcs {
		nameSet[pvc.Name] = true
	}
	for _, pvc := range other.PersistentVolumeClaims {
		if !nameSet[pvc.Name] {
			pvcs = append(pvcs, pvc)
			nameSet[pvc.Name] = true
		}
	}

	return Objects{
		Deployments:            append(this.Deployments, other.Deployments...),
		StatefulSets:           append(this.StatefulSets, other.StatefulSets...),
		Services:               append(this.Services, other.Services...),
		PersistentVolumeClaims: pvcs,
		Secrets:                append(this.Secrets, other.Secrets...),
		Ingresses:              append(this.Ingresses, other.Ingresses...),
		Others:                 append(this.Others, other.Others...),
	}
}
