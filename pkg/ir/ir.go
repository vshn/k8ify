package ir

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/v2/types"
	prometheusTypes "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/vshn/k8ify/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Inputs struct {
	Services  map[string]*Service
	Volumes   map[string]*Volume
	TargetCfg TargetCfg
}

func NewInputs() *Inputs {
	return &Inputs{
		Services: make(map[string]*Service),
		Volumes:  make(map[string]*Volume),
	}
}

func FromCompose(project *composeTypes.Project) *Inputs {
	inputs := NewInputs()

	// first find out all the regular ("parent") services
	for _, composeService := range project.Services {
		if util.PartOf(composeService.Labels) != nil {
			continue
		}
		// `project.Services` is a list, so we use the name as reported by the
		// service
		inputs.Services[composeService.Name] = NewService(composeService.Name, composeService)
	}

	// then find all the parts that belong to a parent service and attach them to their parents
	for _, composeService := range project.Services {
		partOf := util.PartOf(composeService.Labels)
		if partOf == nil {
			continue
		}
		parent, ok := inputs.Services[*partOf]
		if ok {
			service := NewService(composeService.Name, composeService)
			parent.AddPart(service)
			continue
		}

		if partOfServiceConf, exists := project.Services[*partOf]; exists {
			recursivePart := *(util.PartOf(partOfServiceConf.Labels))
			logrus.Errorf("Service %s is configured to be partOf Service %s, but Service %s already is partOf Service %s, This is not supported. Please annotate Service %s to be partOf Service %s to have all of them in one pod",
				composeService.Name,
				*partOf,
				*partOf,
				recursivePart,
				composeService.Name,
				recursivePart,
			)
			os.Exit(1)
		} else {
			logrus.Errorf("Service %s is configured to be partOf Service %s. However, the Service %s does not exists. Check for typos or missing configuration.",
				composeService.Name,
				*partOf,
				*partOf,
			)
			os.Exit(1)
		}
	}

	for name, composeVolume := range project.Volumes {
		// `project.CollectVolumes` is a map where the key is the volume name, while
		// `volume.Name` is something else (the name prefixed with `_`???). So
		// we use the key as the name.
		inputs.Volumes[name] = NewVolume(name, composeVolume)
	}

	if targetCfg, ok := project.Extensions["x-targetCfg"]; ok {
		if targetCfgMap, ok := targetCfg.(map[string]interface{}); ok {
			inputs.TargetCfg = targetCfgMap
		}
	}

	return inputs
}

// Service provides some k8ify-specific abstractions & utility around Compose
// service configurations.
type Service struct {
	Name string

	raw composeTypes.ServiceConfig

	parts []*Service
}

func NewService(name string, composeService composeTypes.ServiceConfig) *Service {
	return &Service{Name: name, raw: composeService, parts: make([]*Service, 0)}
}

// AsCompose returns the underlying compose config
// TODO(mh): make me obsolete!
func (s *Service) AsCompose() composeTypes.ServiceConfig {
	return s.raw
}

func (s *Service) AddPart(part *Service) {
	s.parts = append(s.parts, part)
}

func (s *Service) GetParts() []*Service {
	return s.parts
}

// VolumeNames lists the names of all volumes that are mounted by this service
func (s *Service) VolumeNames() []string {
	names := []string{}

	for _, mount := range s.raw.Volumes {
		if mount.Type != "volume" {
			// We don't support anything else (yet)
			continue
		}

		names = append(names, mount.Source)
	}

	return names
}

func (s *Service) Volumes(volumes map[string]*Volume) (map[string]*Volume, map[string]*Volume) {
	rwoVolumes := make(map[string]*Volume)
	rwxVolumes := make(map[string]*Volume)
	for _, volumeName := range s.VolumeNames() {
		volume := volumes[volumeName]
		if volume.IsShared() {
			rwxVolumes[volume.Name] = volume
		} else {
			rwoVolumes[volume.Name] = volume
		}
	}
	return rwoVolumes, rwxVolumes
}

func (s *Service) IsSingleton() bool {
	return util.IsSingleton(s.raw.Labels)
}
func (s *Service) Labels() map[string]string {
	return s.raw.Labels
}

type PublishedPort struct {
	ServicePort   uint16
	ContainerPort uint16
}

func (s *Service) GetPorts() []PublishedPort {
	var publishedPorts []PublishedPort
	for _, port := range s.raw.Ports {
		publishedPort := PublishedPort{
			ServicePort:   uint16(port.Target), // fall-back
			ContainerPort: uint16(port.Target),
		}
		// port.Published can contain a range. Since we can't use this range for k8s we always use the start of the range instead.
		portRange := strings.Split(port.Published, "-")
		if len(portRange) > 0 {
			p, err := strconv.ParseUint(portRange[0], 10, 16)
			if err == nil {
				publishedPort.ServicePort = uint16(p)
			}
		}
		publishedPorts = append(publishedPorts, publishedPort)
	}
	return publishedPorts
}

// Volume provides some k8ify-specific abstractions & utility around Compose
// volume configurations.
type Volume struct {
	Name string

	raw composeTypes.VolumeConfig
}

func NewVolume(name string, composeVolume composeTypes.VolumeConfig) *Volume {
	return &Volume{
		Name: name,
		raw:  composeVolume,
	}
}

func (v *Volume) IsShared() bool {
	return util.IsShared(v.raw.Labels)
}
func (v *Volume) IsSingleton() bool {
	return util.IsSingleton(v.raw.Labels)
}
func (v *Volume) Labels() map[string]string {
	return v.raw.Labels
}

func (v *Volume) Size(fallback string) resource.Quantity {
	return util.StorageSize(v.raw.Labels, fallback)
}
func (v *Volume) SizeIsMissing() bool {
	return util.StorageSizeRaw(v.raw.Labels) == nil
}

type TargetCfg map[string]interface{}

func (t TargetCfg) appsDomain() *string {
	if value, ok := t["appsDomain"]; ok {
		if domain, ok := value.(string); ok {
			if strings.HasPrefix(domain, "*.") {
				domain = domain[1:]
			}
			if !strings.HasPrefix(domain, ".") {
				domain = "." + domain
			}
			if len(domain) < 2 {
				return nil
			}
			return &domain
		}
	}
	return nil
}

func (t TargetCfg) IsSubdomainOfAppsDomain(domain string) bool {
	appsDomain := t.appsDomain()
	if appsDomain == nil || domain == "" {
		return false
	}
	domainComponents := strings.Split(domain, ".")
	if len(domainComponents) < 2 {
		return false
	}
	return domainComponents[0]+*appsDomain == domain
}

func (t TargetCfg) MaxExposeLength() int {
	if value, ok := t["maxExposeLength"]; ok {
		if length, ok := value.(int); ok {
			return length
		}
	}
	return 63
}

// ServiceMonitorConfig An intermediate struct that makes it easier to access all needed config values
// in one place for the ServiceMonitor.
// We did not use prometheus.ServiceMonitor directly, because then the name would be: serviceMonitor.Endpoints[0].name
type ServiceMonitorConfig struct {
	Interval     *string
	Path         *string
	Scheme       *string
	EndpointName *string
}

// ServiceMonitorConfigPointer Parses the config values for serviceMonitor
func ServiceMonitorConfigPointer(labels map[string]string) *ServiceMonitorConfig {
	enabled := util.GetBoolean(labels, "k8ify.prometheus.serviceMonitor")
	if !enabled {
		return nil
	}
	return &ServiceMonitorConfig{
		Interval:     util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.interval")),
		Path:         util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.path")),
		Scheme:       util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.scheme")),
		EndpointName: util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.name")),
	}
}

type ServiceMonitorBasicAuthConfig struct {
	Username string
	Password string
}

func ServiceMonitorBasicAuthConfigPointer(labels map[string]string) (*ServiceMonitorBasicAuthConfig, error) {
	enabled := util.GetBoolean(labels, "k8ify.prometheus.serviceMonitor.endpoint.basicAuth")
	if !enabled {
		return nil, nil
	}
	username := util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.basicAuth.username")
	password := util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.basicAuth.password")
	if util.IsBlank(username) || util.IsBlank(password) {
		return nil, fmt.Errorf("username or password is blank, this is not allowed. username had length %d, password had length %d",
			len(util.OrEmptyString(username)),
			len(util.OrEmptyString(password)))
	}

	return &ServiceMonitorBasicAuthConfig{
		Username: *username,
		Password: *password,
	}, nil
}

type ServiceMonitorTlsConfig struct {
	Ca                 *string
	Cert               *string
	KeySecretValue     *string
	InsecureSkipVerify *bool
	MaxVersion         *prometheusTypes.TLSVersion
	MinVersion         *prometheusTypes.TLSVersion
	ServerName         *string
}

var (
	tlsVersion10 = string(prometheusTypes.TLSVersion10)
	tlsVersion11 = string(prometheusTypes.TLSVersion11)
	tlsVersion12 = string(prometheusTypes.TLSVersion12)
	tlsVersion13 = string(prometheusTypes.TLSVersion13)
)

func parseTlsVersion(string *string) (*prometheusTypes.TLSVersion, error) {
	if string == nil {
		return nil, nil
	}
	switch *string {
	case tlsVersion10:
		return util.GetPointer(prometheusTypes.TLSVersion10), nil
	case tlsVersion11:
		return util.GetPointer(prometheusTypes.TLSVersion11), nil
	case tlsVersion12:
		return util.GetPointer(prometheusTypes.TLSVersion12), nil
	case tlsVersion13:
		return util.GetPointer(prometheusTypes.TLSVersion13), nil
	default:
		return nil, fmt.Errorf("unknown TLSVersion: %v", *string)
	}
}

func ServiceMonitorTlsConfigPointer(labels map[string]string) (*ServiceMonitorTlsConfig, *[]error) {
	enabled := util.GetBoolean(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig")
	if !enabled {
		return nil, nil
	}
	maxTlsVersion, errMaxVersion := parseTlsVersion(
		util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.maxVersion")),
	)
	minTlsVersion, errMinVersion := parseTlsVersion(
		util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.minVersion")),
	)

	errors := util.FilterNilErrors([]error{errMaxVersion, errMinVersion})
	if len(errors) > 0 {
		return nil, &errors
	}

	return &ServiceMonitorTlsConfig{
		Ca:                 util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.ca")),
		Cert:               util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.cert")),
		KeySecretValue:     util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.keySecretValue")),
		InsecureSkipVerify: util.FilterBlankBool(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.insecureSkipVerify")),
		MaxVersion:         maxTlsVersion,
		MinVersion:         minTlsVersion,
		ServerName:         util.FilterBlank(util.GetOptional(labels, "k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.serverName")),
	}, nil
}
