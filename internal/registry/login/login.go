package login

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"

	"github.com/fluxcd/image-reflector-controller/internal/registry"
	"github.com/fluxcd/image-reflector-controller/internal/registry/aws"
	"github.com/fluxcd/image-reflector-controller/internal/registry/azure"
	"github.com/fluxcd/image-reflector-controller/internal/registry/gcp"
)

// ImageRegistryProvider analyzes the provided image and returns the identified
// container image registry provider.
func ImageRegistryProvider(image string, ref name.Reference) registry.Provider {
	_, _, ok := aws.ParseImage(image)
	if ok {
		return registry.ProviderAWS
	}
	if gcp.ValidHost(ref.Context().RegistryStr()) {
		return registry.ProviderGCR
	}
	if azure.ValidHost(ref.Context().RegistryStr()) {
		return registry.ProviderAzure
	}
	return registry.ProviderGeneric
}

// ProviderOptions contains options for registry provider login.
type ProviderOptions struct {
	// AwsAutoLogin enables automatic attempt to get credentials for images in
	// ECR.
	AwsAutoLogin bool
	// GcpAutoLogin enables automatic attempt to get credentials for images in
	// GCP.
	GcpAutoLogin bool
	// AzureAutoLogin enables automatic attempt to get credentials for images in
	// ACR.
	AzureAutoLogin bool
}

// Manager is a login manager for various registry providers.
type Manager struct {
	ecr *aws.Client
	gcr *gcp.Client
	acr *azure.Client
}

// NewManager initializes a Manager with default registry clients
// configurations.
func NewManager() *Manager {
	return &Manager{
		ecr: aws.NewClient(),
		gcr: gcp.NewClient(),
		acr: azure.NewClient(),
	}
}

// WithECRClient allows overriding the default ECR client.
func (m *Manager) WithECRClient(c *aws.Client) *Manager {
	m.ecr = c
	return m
}

// WithGCRClient allows overriding the default GCR client.
func (m *Manager) WithGCRClient(c *gcp.Client) *Manager {
	m.gcr = c
	return m
}

// WithACRClient allows overriding the default ACR client.
func (m *Manager) WithACRClient(c *azure.Client) *Manager {
	m.acr = c
	return m
}

// Login performs authentication against a registry and returns the
// authentication material. For generic registry provider, it is no-op.
func (m *Manager) Login(ctx context.Context, image string, ref name.Reference, opts ProviderOptions) (authn.Authenticator, error) {
	switch ImageRegistryProvider(image, ref) {
	case registry.ProviderAWS:
		return m.ecr.Login(ctx, opts.AwsAutoLogin, image)
	case registry.ProviderGCR:
		return m.gcr.Login(ctx, opts.GcpAutoLogin, image, ref)
	case registry.ProviderAzure:
		return m.acr.Login(ctx, opts.AzureAutoLogin, image, ref)
	}
	return nil, nil
}
