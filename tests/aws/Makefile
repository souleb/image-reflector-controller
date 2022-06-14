GO_TEST_PATH ?= ./...
GO_TEST_ARGS ?=
TEST_TIMEOUT ?= 20m
FLUX_MANIFEST_URL ?= https://github.com/fluxcd/flux2/releases/latest/download/install.yaml

IMG ?= fluxcd/image-reflector-controller

# Builds manifests and run the tests.
test: build-manifests
	go test -timeout $(TEST_TIMEOUT) -v $(GO_TEST_PATH) $(GO_TEST_ARGS)

# Build the manifests required in the test.
build-manifests:
	mkdir -p build/flux
	curl -Lo build/flux/install.yaml ${FLUX_MANIFEST_URL}
	cp kustomization.yaml build/flux
	cd build/flux && kustomize edit set image fluxcd/image-reflector-controller=${IMG}
	kustomize build build/flux > build/flux.yaml

# Delete all the build files.
distclean:
	rm -r build/
