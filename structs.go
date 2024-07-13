package containers

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerCreateConfig A config wrapper for the creation of a container
type ContainerCreateConfig struct {
	Name             string
	Config           *container.Config
	HostConfig       *container.HostConfig
	NetworkingConfig *network.NetworkingConfig
	Platform         *v1.Platform
}

type ImageBuildOut struct {
	Stream         string         `json:"stream,omitempty"`
	Status         string         `json:"status"`
	Id             string         `json:"id"`
	Progress       string         `json:"progress,omitempty"`
	ProgressDetail ProgressDetail `json:"progressDetail,omitempty,mapstructure,squash"`
}

type ProgressDetail struct {
	Current int `json:"current,omitempty"`
	Total   int `json:"total,omitempty"`
}
