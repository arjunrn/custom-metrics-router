.PHONY: generate-all generate-object generate-codegen generate-crd build build.linux build.local build.osx test lint

BINARY  ?= custom-metrics-router
LDFLAGS ?= -X main.version=$(VERSION) -w -s
VERSION ?= $(shell git describe --tags --always --dirty)
IMAGE ?= custom-metrics-router

default: build.local

clean:
	rm -rf build

build.local: build/$(BINARY)
build.linux: build/linux/$(BINARY)
build.osx: build/osx/$(BINARY)

build.docker: build.linux
	docker build -t $(IMAGE):$(VERSION) .

build.push: build.docker
	docker push $(IMAGE):$(VERSION)

build/$(BINARY): go.mod $(SOURCES) $(OPENAPI)
	CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

build/linux/$(BINARY): go.mod $(SOURCES) $(OPENAPI)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/linux/$(BINARY) -ldflags "$(LDFLAGS)" .

build/osx/$(BINARY): go.mod $(SOURCES) $(OPENAPI)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o build/osx/$(BINARY) -ldflags "$(LDFLAGS)" .

test:
	go test ./...

lint:
	golangci-lint -c .golangci.yml run ./...

generate-all: generate-object generate-codegen generate-crd

generate-object:
	go run sigs.k8s.io/controller-tools/cmd/controller-gen object paths=pkg/apis/metricsrouter.io/v1alpha1/types.go

generate-codegen:
	hack/update-codegen.sh

generate-crd:
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:crdVersions=v1 paths=./pkg/apis/... output:crd:dir=deploy
