VERSION 0.7
FROM golang:1.22-bookworm
WORKDIR /workspace

docker-all:
  BUILD --platform=linux/amd64 --platform=linux/arm64 +docker

docker:
  FROM gcr.io/distroless/static:nonroot
  WORKDIR /
  COPY LICENSE /usr/local/share/servfaild/
  ARG TARGETARCH
  COPY (+servfaild/servfaild --GOARCH=${TARGETARCH}) /usr/local/bin/servfaild
  USER 65532:65532
  EXPOSE 5353/udp 5353/tcp
  ENTRYPOINT ["//usr/local/bin/servfaild"]
  ARG VERSION=latest-dev
  SAVE IMAGE --push ghcr.io/noisysockets/servfaild:${VERSION}
  SAVE IMAGE --push ghcr.io/noisysockets/servfaild:latest

servfaild:
  ARG GOOS=linux
  ARG GOARCH=amd64
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -ldflags '-s' -o servfaild main.go
  SAVE ARTIFACT ./servfaild AS LOCAL dist/servfaild-${GOOS}-${GOARCH}

tidy:
  LOCALLY
  RUN go mod tidy
  RUN go fmt ./...

lint:
  FROM golangci/golangci-lint:v1.57.2
  WORKDIR /workspace
  COPY . ./
  RUN golangci-lint run --timeout 5m ./...

test:
  COPY go.mod go.sum ./
  RUN go mod download
  COPY . .
  RUN go test -coverprofile=coverage.out -v ./...
  SAVE ARTIFACT ./coverage.out AS LOCAL coverage.out