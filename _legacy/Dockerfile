FROM golang:1.25 AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/gopalette ./cmd

FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /out/gopalette /app/gopalette
COPY config.yaml /app/config.yaml
COPY static /app/static

EXPOSE 8080
ENTRYPOINT ["/app/gopalette"]
