package ir

import (
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/vshn/k8ify/pkg/util"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Inputs struct {
	Services map[string]Service
	Volumes  map[string]Volume
}

func NewInputs() *Inputs {
	return &Inputs{
		Services: make(map[string]Service),
		Volumes:  make(map[string]Volume),
	}
}

func FromCompose(project *composeTypes.Project) *Inputs {
	inputs := NewInputs()

	for _, composeService := range project.Services {
		// `project.Services` is a list, so we use the name as reported by the
		// service
		inputs.Services[composeService.Name] = NewService(composeService.Name, composeService)
	}

	for name, composeVolume := range project.Volumes {
		// `project.Volumes` is a map where the key is the volume name, while
		// `volume.Name` is something else (the name prefixed with `_`???). So
		// we use the key as the name.
		inputs.Volumes[name] = NewVolume(name, composeVolume)
	}

	return inputs
}

// Service provides some k8ify-specific abstractions & utility around Compose
// service configurations.
type Service struct {
	Name string

	raw composeTypes.ServiceConfig
}

func NewService(name string, composeService composeTypes.ServiceConfig) Service {
	return Service{Name: name, raw: composeService}
}

// AsCompose returns the underlying compose config
// TODO(mh): make me obsolete!
func (s *Service) AsCompose() composeTypes.ServiceConfig {
	return s.raw
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

func (s *Service) Volumes(volumes map[string]Volume) ([]Volume, []Volume) {
	rwoVolumes := []Volume{}
	rwxVolumes := []Volume{}
	for _, volumeName := range s.VolumeNames() {
		volume := volumes[volumeName]
		if volume.IsShared() {
			rwxVolumes = append(rwxVolumes, volume)
		} else {
			rwoVolumes = append(rwoVolumes, volume)
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

// Volume provides some k8ify-specific abstractions & utility around Compose
// volume configurations.
type Volume struct {
	Name string

	raw composeTypes.VolumeConfig
}

func NewVolume(name string, composeVolume composeTypes.VolumeConfig) Volume {
	return Volume{
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
