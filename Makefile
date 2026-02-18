IMAGE_REPO   ?= quay.io/azaalouk
IMAGE_NAME   ?= agent-registry
IMAGE_TAG    ?= latest
IMAGE        := $(IMAGE_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
OVERLAY      ?= openshift
NAMESPACE    ?= agent-registry
CONTAINER_RT ?= podman

.PHONY: build image push deploy undeploy clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/agent-registry ./cmd/server

image:
	$(CONTAINER_RT) build --platform linux/amd64 -t $(IMAGE) -f deploy/Dockerfile .

push:
	$(CONTAINER_RT) push $(IMAGE)

deploy:
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -k deploy/k8s/overlays/$(OVERLAY)

undeploy:
	kubectl delete -k deploy/k8s/overlays/$(OVERLAY) --ignore-not-found

clean:
	rm -rf bin/
