package ir

import (
	"strconv"
	"strings"

	composeTypes "github.com/compose-spec/compose-go/types"
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
