package registry

// Provider is used to categorize the registry providers.
type Provider int

// Registry providers.
const (
	ProviderGeneric Provider = iota
	ProviderAWS
	ProviderGCR
	ProviderAzure
)
