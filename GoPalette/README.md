# Kratos Project Template

## Install Kratos
```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```
## Create a service
```
# Create a template project
kratos new server

cd server
# Add a proto template
kratos proto add api/server/server.proto
# Generate the proto code
kratos proto client api/server/server.proto
# Generate the source code of service by proto file
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```
## Generate other auxiliary files by Makefile
```
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
# Generate all files
make all
```
## Automated Initialization (wire)
```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## Docker
```bash
# build
docker build -t <your-docker-image-name> .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -v </path/to/your/configs>:/data/conf <your-docker-image-name>
```

## Observability

The local and production Compose stacks include Prometheus, Grafana, Jaeger, Loki, and Alloy.

| Component | Local port |
| --- | --- |
| Grafana | `3000` |
| Prometheus | `9091` |
| Loki | `3100` |
| Alloy | `12345` |
| Jaeger | `16686` |

Grafana provisions Prometheus and Loki automatically. Use Explore with Loki query `{compose_project="gopalette"}` to inspect aggregated container logs. If you start Compose with a custom project name, set `GOPALETTE_LOGS_PROJECT` to the same value so Alloy filters the right containers.
