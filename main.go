package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

var dockerRegistryUserID = ""

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

//go:embed embeddable
var f embed.FS

type ErrorDetail struct {
	Message string `json:"message"`
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	consulLocation, err := exec.LookPath("consul")
	if err != nil {
		log.Fatal("could not find 'consul' on your path")
	}

	envoyLocation, err := exec.LookPath("envoy")
	if err != nil {
		log.Fatal("could not find 'envoy' on your path")
	}

	location, err := buildTempDir(consulLocation, envoyLocation)
	if err != nil {
		log.Fatal(err)
	}

	err = imageBuild(cli, location)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
}

func buildTempDir(consulLocation, envoyLocation string) (string, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "convoy-build")
	entryFile, err := f.Open("embeddable/entrypoint.sh")
	if err != nil {
		return "", err
	}

	dockerFile, err := f.Open("embeddable/Dockerfile")
	if err != nil {
		return "", err
	}
	entryFileDst, err := os.Create(fmt.Sprintf("%s/entrypoint.sh", dir))
	if err != nil {
		return "", err
	}

	dockerfileDst, err := os.Create(fmt.Sprintf("%s/Dockerfile", dir))
	if err != nil {
		return "", err
	}

	_, err = io.Copy(entryFileDst, entryFile)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(dockerfileDst, dockerFile)

	if err != nil {
		return "", err
	}

	consulBytes, err := os.ReadFile(consulLocation)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fmt.Sprintf("%s/consul", dir), consulBytes, 0777)
	if err != nil {
		return "", err
	}

	envoyBytes, err := os.ReadFile(envoyLocation)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(fmt.Sprintf("%s/envoy", dir), envoyBytes, 0777)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func imageBuild(dockerClient *client.Client, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	tar, err := archive.TarWithOptions(location, &archive.TarOptions{})
	if err != nil {
		return err
	}

	opts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{"consul-envoy:local"},
		Remove:     true,
	}
	res, err := dockerClient.ImageBuild(ctx, tar, opts)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	err = print(res.Body)
	if err != nil {
		return err
	}

	return nil
}

func print(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		fmt.Println(scanner.Text())
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
