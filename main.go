package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
)

const usage = `Usage of convoy-build:
  -c, --consul-location absolute filepath of consul source code on your system, takes precedence over the CONVOY_CONSUL_LOCATION env var
  -e, --envoy-version version of envoy to use, defaults to 1.26
  -h, --help prints help information 
`

//go:embed embeddable
var f embed.FS

func main() {
	var consulLocation string
	var envoyVersion string

	err := parseArgs(&consulLocation, &envoyVersion)
	if err != nil {
		log.Fatal(err)
	}

	curDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	defer os.Chdir(curDir)

	consulBytes, err := buildConsul(consulLocation, runtime.GOARCH, curDir)
	if err != nil {
		log.Fatal(err)
	}

	dir, err := buildTempDir(consulBytes)
	if err != nil {
		log.Fatal(err)
	}

	err = buildDockerImage(dir, envoyVersion)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("successfully built convoy image, it is available as \"convoy:local\"")
}

func parseArgs(consulLocation, envoyVersion *string) error {
	flag.StringVar(consulLocation, "consul-location", "", "absolute filepath of consul source code on your system, takes precedence over the CONVOY_CONSUL_LOCATION env var")
	flag.StringVar(consulLocation, "c", "", "absolute filepath of consul source code on your system, takes precedence over the CONVOY_CONSUL_LOCATION env var")
	flag.StringVar(envoyVersion, "envoy-version", "", "envoy version to use, defaults to 1.26")
	flag.StringVar(envoyVersion, "e", "", "envoy version to use, defaults to 1.26")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	if *consulLocation == "" {
		*consulLocation = os.Getenv("CONVOY_CONSUL_LOCATION")
	}

	if *consulLocation == "" {
		return errors.New("consul version must be supplied")
	}

	return nil
}

func buildTempDir(consulBytes []byte) (string, error) {
	dir, err := os.MkdirTemp(os.TempDir(), "convoy-build")
	if err != nil {
		return "", err
	}

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

	err = os.WriteFile(fmt.Sprintf("%s/consul", dir), consulBytes, 0o777)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func buildConsul(consulLocation, goArch, curDir string) ([]byte, error) {
	defer os.Chdir(curDir)

	err := os.Chdir(consulLocation)
	if err != nil {
		return nil, err
	}

	log.Print("building consul")
	cmd := exec.Command("make", "linux")
	err = cmd.Run()
	if err != nil {
		log.Print("failed to build consul")
		return nil, err
	}

	binLocation := fmt.Sprintf("%s/pkg/bin/linux_%s/consul", consulLocation, goArch)

	consulBytes, err := os.ReadFile(binLocation)
	if err != nil {
		return nil, err
	}

	return consulBytes, nil
}

func buildDockerImage(dir, envoyVersion string) error {
	err := os.Chdir(dir)
	if err != nil {
		log.Fatal(err)
	}

	dockerBuildArgs := []string{"build", ".", "-t", "convoy:local"}
	if envoyVersion != "" {
		dockerBuildArgs = append(dockerBuildArgs[:1], append([]string{"--build-arg", fmt.Sprintf("ENVOY_VERSION=v%s-latest", envoyVersion)}, dockerBuildArgs[1:]...)...)
	}
	cmd := exec.Command("docker", dockerBuildArgs...)
	log.Print("building convoy image")
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
