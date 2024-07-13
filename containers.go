package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
)

// DockerClient ~ The docker client
var DockerClient *client.Client

// InitializeDockerClient ~ Initializes the docker client
func InitializeDockerClient() error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO INITIALIZE DOCKER CLIENT! => " + err.Error())
	}
	DockerClient = cli
	return nil
}

// CloseDockerClient ~ Closes the docker client
func CloseDockerClient() error {
	if DockerClient == nil {
		return errors.New("[ERR:] [DOCKER] => DOCKER CLIENT NOT FOUND")
	}
	closeErr := DockerClient.Close()
	if closeErr != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO CLOSE DOCKER CLIENT => " + closeErr.Error())
	}
	return nil
}

// ListContainers ~ Unused. Lists all containers
func ListContainers() error {
	containers, err := DockerClient.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return errors.New("[ERR] [DOCKER:] => FAILED TO LIST CONTAINERS => " + err.Error())
	}
	for _, container := range containers {
		fmt.Printf("%s %s %s %s \n\n", container.ID, container.ID[:10], container.Image, container.Names[0])
	}
	return nil
}

// BuildImage ~ Builds an image
func BuildImage(path string, imageName string) error {
	buildCtx, buildCtxErr := archive.Tar(path, archive.Uncompressed)
	if buildCtxErr != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO CREATE BUILD CONTEXT FOR IMAGE " + imageName + " => " + buildCtxErr.Error())
	}

	buildOptions := types.ImageBuildOptions{
		Dockerfile:     "Dockerfile",
		PullParent:     true,
		Tags:           []string{imageName},
		Remove:         true,
		NoCache:        true,
		SuppressOutput: false,
	}

	image, imgErr := DockerClient.ImageBuild(context.Background(), buildCtx, buildOptions)
	if imgErr != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO BUILD IMAGE " + imageName + " => " + imgErr.Error())
	}
	for {
		var buildOut ImageBuildOut
		outErr := json.NewDecoder(image.Body).Decode(&buildOut)
		if outErr == io.EOF {
			image.Body.Close()
			break
		}
	}
	return nil
}

// CreateContainer ~ Creates a container
func CreateContainer(config *ContainerCreateConfig) (container.CreateResponse, error) {
	containerRes, err := DockerClient.ContainerCreate(context.Background(),
		config.Config,
		config.HostConfig,
		config.NetworkingConfig,
		config.Platform,
		config.Name,
	)
	if err != nil {
		return containerRes, errors.New("[ERR:] [DOCKER] => FAILED TO CREATE CONTAINER " + config.Name + " => " + err.Error())
	}

	return containerRes, nil
}

// StartContainer ~ Starts a container
func StartContainer(cont container.CreateResponse) error {
	err := DockerClient.ContainerStart(context.Background(), cont.ID, container.StartOptions{})
	if err != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO START CONTAINER WITH ID: " + cont.ID + " => " + err.Error())
	}
	return nil
}

// StopContainer ~ Stops a container
func StopContainer(containerID string) error {
	err := DockerClient.ContainerStop(context.Background(), containerID, container.StopOptions{
		Signal: "SIGTERM",
	})
	if err != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO STOP CONTAINER WITH ID: " + containerID + " => " + err.Error())
	}
	return nil
}

// PurgeContainer ~ Purges a stopped container
func PurgeContainer(containerID string) error {
	removeOptions := container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks:   false,
		Force:         true,
	}
	err := DockerClient.ContainerRemove(context.Background(), containerID, removeOptions)
	if err != nil {
		return errors.New("[ERR:] [DOCKER] => FAILED TO PURGE CONTAINER WITH ID: " + containerID + " => " + err.Error())
	}
	return nil
}

// DeleteImage ~ Deletes an image
func DeleteImage(imageName string) (bool, error) {
	img, _, imgErr := DockerClient.ImageInspectWithRaw(context.Background(), imageName)
	exists := true
	if imgErr != nil {
		exists = false
	}
	if exists {
		_, imgRemoveErr := DockerClient.ImageRemove(context.Background(), img.ID, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if imgRemoveErr != nil {
			return exists, errors.New("[ERR:] [DOCKER] => FAILED TO DELETE IMAGE: " + imageName + " | => " + imgRemoveErr.Error())
		}
	}

	return exists, nil
}

// PruneDanglingImages ~ Prunes all dangling images
func PruneDanglingImages() (image.PruneReport, error) {
	pruneFilters := filters.NewArgs()
	pruneFilters.Add("dangling", "true")

	pruneReport, pruneErr := DockerClient.ImagesPrune(context.Background(), pruneFilters)
	if pruneErr != nil {
		return pruneReport, errors.New("[ERR:] [DOCKER] => FAILED TO PRUNE DANGLING IMAGES  | => " + pruneErr.Error())
	}

	return pruneReport, nil
}

// GetContainerHealthStatus ~ Gets the health status of a container
func GetContainerHealthStatus(containerID string) (string, error) {
	// Starting, Healthy or Unhealthy
	containerJSON, err := DockerClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return "unhealthy", err
	}

	return containerJSON.State.Health.Status, nil
}

// Exec executes a command on a running container
func Exec(containerID string, cmd []string) (string, error) {

	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execIDResp, err := DockerClient.ContainerExecCreate(context.Background(), containerID, execConfig)
	if err != nil {
		return "", errors.New("[ERR:] [DOCKER] => FAILED TO CREATE EXEC ISNTANCE => " + err.Error())
	}

	// Attach to the exec instance
	resp, err := DockerClient.ContainerExecAttach(context.Background(), execIDResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", errors.New("[ERR:] [DOCKER] => FAILED TO ATTACH TO EXEC INSTANCE => " + err.Error())
	}

	defer resp.Close()
	var outBuf, errBuf bytes.Buffer

	// Copy the output of the command to the buffers
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
	if err != nil {
		return "", errors.New("[ERR:] [DOCKER] => FAILED TO COPY EXEC OUTPUT => " + err.Error())
	}

	// Inspect exec instance to get the exit code
	execInspectResp, err := DockerClient.ContainerExecInspect(context.Background(), execIDResp.ID)
	if err != nil {
		return "", errors.New("[ERR:] [DOCKER] => FAILED TO INSPECT EXEC INSTANCE => " + err.Error())
	}

	if execInspectResp.ExitCode != 0 {
		return "", errors.New("[ERR:] [DOCKER] => COMMAND EXITIED WITH CODE " + fmt.Sprint(execInspectResp.ExitCode) + " => " + errBuf.String())
	}

	psOutput := outBuf.String()
	return psOutput, nil
}
