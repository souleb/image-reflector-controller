module github.com/fluxcd/image-reflector-controller/api

go 1.20

require (
	github.com/fluxcd/pkg/apis/acl v0.1.0
	github.com/fluxcd/pkg/apis/meta v1.1.1
	k8s.io/apimachinery v0.27.3
	sigs.k8s.io/controller-runtime v0.15.0
)

// Fix CVE-2022-28948
replace gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/text v0.9.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/utils v0.0.0-20230505201702-9f6742963106 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)
