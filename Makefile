# Image URL to use for building/pushing image targets
IMG ?= todolist-operator:latest

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary name
BINARY_NAME=manager

.PHONY: all build clean test deps docker-build docker-push install deploy undeploy

all: test build

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v main.go

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Build the docker image
docker-build:
	docker build -t ${IMG} .

# Push the docker image
docker-push:
	docker push ${IMG}

# Install CRDs into the cluster
install:
	kubectl apply -f manifests/todolist-crd.yaml

# Uninstall CRDs from the cluster
uninstall:
	kubectl delete -f manifests/todolist-crd.yaml

# Deploy operator to the cluster
deploy: install
	kubectl apply -f manifests/operator.yaml

# Undeploy operator from the cluster
undeploy:
	kubectl delete -f manifests/operator.yaml

# Run the operator locally
run:
	go run main.go

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run go mod verify
verify:
	go mod verify

# Generate go.sum
tidy:
	go mod tidy

help:
	@echo "Available targets:"
	@echo "  build         - Build the operator binary"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Download dependencies"
	@echo "  docker-build  - Build docker image"
	@echo "  docker-push   - Push docker image"
	@echo "  install       - Install CRDs"
	@echo "  uninstall     - Uninstall CRDs"
	@echo "  deploy        - Deploy operator to cluster"
	@echo "  undeploy      - Remove operator from cluster"
	@echo "  run           - Run operator locally"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  verify        - Verify go.mod"
	@echo "  tidy          - Tidy go.mod"
