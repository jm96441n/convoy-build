# Convoy-Build

Simple program to construct a docker image, tagged `convoy:local`, that contains both consul
and envoy. 

## Installation

Install via `go install`:
```
go install github.com/jm96441n/convoy-build
```

## Running
`convoy-build` compiles consul from your system and places that into the docker container with envoy. To do this you
need to supply the location that consul should be compiled from, this can be done via either by setting an environment
variable `CONVOY_CONSUL_LOCATION` or using the `--consul-location` or `-c` flag (the cli arg takes
precedence over the env var). You can also specify the envoy version used via the `--envoy-version` or `-e` flag.

Full flag reference
```
  -c, --consul-location absolute filepath of consul source code on your system, takes precedence over the CONVOY_CONSUL_LOCATION env var
  -e, --envoy-version version of envoy to use, defaults to 1.26
  -h, --help prints help information 
```
