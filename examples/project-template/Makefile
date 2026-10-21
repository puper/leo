BINARY_NAME=server
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD)
LDFLAGS=-ldflags "-X server/internal/pkg/version.version=${VERSION} -X server/internal/pkg/version.buildTime=${BUILD_TIME} -X server/internal/pkg/version.gitCommit=${GIT_COMMIT}"

.PHONY: build run test clean lint docker-build docker-run swagger

build:
	go build -mod=readonly ${LDFLAGS} -o ${BINARY_NAME} .

run: build
	./${BINARY_NAME} serve --config ./config/config.toml

test:
	go test -mod=readonly ./...

clean:
	rm -f ${BINARY_NAME}

lint:
	go vet -mod=readonly ./...

swagger:
	go generate ./internal/swagger

docker-build:
	docker build --build-arg VERSION=${VERSION} --build-arg COMMIT=${GIT_COMMIT} --build-arg BUILD_TIME=${BUILD_TIME} -t ${BINARY_NAME}:${VERSION} .

docker-run:
	docker run --rm -p 8888:8888 --mount type=bind,src="$(CURDIR)/config/config.toml",dst=/app/config/config.toml,readonly ${BINARY_NAME}:${VERSION}
