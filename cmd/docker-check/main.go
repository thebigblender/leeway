package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const (
	smokeContainerName = "leeway-runtime-smoke"
	smokePurposeLabel  = "io.leeway.purpose"
	smokePurposeValue  = "runtime-smoke"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	var mode, target string
	switch {
	case len(args) == 1 && args[0] != "" && args[0] != "start-smoke" && args[0] != "stop-smoke":
		mode, target = "inspect", args[0]
	case len(args) == 2 && (args[0] == "start-smoke" || args[0] == "stop-smoke") && args[1] != "":
		mode, target = args[0], args[1]
	default:
		return fmt.Errorf("usage: %s <container-id> | start-smoke <image-id> | stop-smoke <container-id>", os.Args[0])
	}

	//connect to Docker daemon
	cli, err := client.New(client.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		return fmt.Errorf("open Docker client: %w", err)
	}
	defer cli.Close()

	//abort if docker daemon doesnt respond in over 3 sec
	timeout := 3 * time.Second
	if mode == "stop-smoke" {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	//get a response
	res, err := cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		return fmt.Errorf("ping Docker: %w", err)
	}
	fmt.Printf("Docker reachable (API %s)\n", res.APIVersion)

	switch mode {
	case "start-smoke":
		return startSmoke(ctx, cli, target)
	case "stop-smoke":
		err := stopSmoke(ctx, cli, target)
		if errdefs.IsNotFound(err) {
			fmt.Printf("Smoke container %s is already gone\n", target)
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("Removed smoke container %s\n", target)
		return nil
	}

	result, err := cli.ContainerInspect(ctx, target, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect container %s: %w", target, err)
	}
	if result.Container.State == nil {
		return fmt.Errorf("inspect container %s: Docker returned no state", target)
	}
	fmt.Printf("Container running: %t\n", result.Container.State.Running)
	return nil
}

func startSmoke(ctx context.Context, cli *client.Client, imageID string) error {
	created, err := cli.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name:  smokeContainerName,
		Image: imageID,
		Config: &container.Config{
			Env:    []string{"APP_VERSION=v2"},
			Labels: map[string]string{smokePurposeLabel: smokePurposeValue},
		},
		HostConfig: &container.HostConfig{NetworkMode: "none"},
	})
	if err != nil {
		return fmt.Errorf("create %s: %w", smokeContainerName, err)
	}
	fmt.Printf("Created %s (%s)\n", smokeContainerName, created.ID)

	if _, err := cli.ContainerStart(ctx, created.ID, client.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("start %s (%s): %w", smokeContainerName, created.ID, err)
	}
	fmt.Printf("Started %s\n", smokeContainerName)
	return nil
}

func stopSmoke(ctx context.Context, cli *client.Client, target string) error {
	result, err := cli.ContainerInspect(ctx, target, client.ContainerInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect smoke container %s: %w", target, err)
	}
	info := result.Container
	if info.Config == nil || info.Config.Labels[smokePurposeLabel] != smokePurposeValue {
		return fmt.Errorf("refusing cleanup of %s: missing %s=%s label", target, smokePurposeLabel, smokePurposeValue)
	}
	if info.ID == "" || info.State == nil {
		return fmt.Errorf("inspect smoke container %s: Docker returned incomplete container information", target)
	}

	if info.State.Running {
		graceSeconds := 3
		if _, err := cli.ContainerStop(ctx, info.ID, client.ContainerStopOptions{Timeout: &graceSeconds}); err != nil {
			return fmt.Errorf("stop smoke container %s: %w", info.ID, err)
		}
	}
	if _, err := cli.ContainerRemove(ctx, info.ID, client.ContainerRemoveOptions{}); err != nil {
		return fmt.Errorf("remove smoke container %s: %w", info.ID, err)
	}
	return nil
}
