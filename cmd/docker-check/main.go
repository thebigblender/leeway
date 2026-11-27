package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"os"

	"github.com/moby/moby/client"
)

func main() {
	if len(os.Args) != 2{
		log.Fatalf("usage: %s <container-id>", os.Args[0])
	}
	containerID := os.Args[1]
	//connect to docker
	cli, err := client.New(client.WithHost("unix:///var/run/docker.sock"))
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	//if docker daemon is frozen/not responsive, abort
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	//sends GET /_ping to docker engine. replies with APIVersion
	res, err := cli.Ping(ctx, client.PingOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Docker reachable (API %s)\n", res.APIVersion)

	result, err := cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Container running: %t\n", result.Container.State.Running)
}