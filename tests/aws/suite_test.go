/*
Copyright 2022 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-multierror"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	imagev1 "github.com/fluxcd/image-reflector-controller/api/v1beta1"
)

const (
	// eksTerraformPath is the path to the terraform working directory
	// containing the terraform configurations.
	eksTerraformPath = "./terraform/eks"
	// kubeconfigPath is the path where the cluster kubeconfig is written to and
	// used from.
	kubeconfigPath = "./build/kubeconfig"
	// fluxInstallManifestPath is the flux installation manifest file path. It
	// is generated before running the Go test.
	fluxInstallManifestPath = "./build/flux.yaml"
)

var (
	// kubeClient is the K8s client of the test cluster.
	kubeClient client.Client

	// retain flag to prevent destroy and retaining the created infrastructure.
	retain = flag.Bool("retain", false, "retain the infrastructure for debugging purposes")

	// existing flag to use existing infrastructure terraform state.
	existing = flag.Bool("existing", false, "use existing infrastructure state for debugging purposes")

	// verbose
	verbose = flag.Bool("verbose", false, "verbose output of the environment setup")
)

func TestMain(m *testing.M) {
	flag.Parse()
	ctx := context.TODO()

	opts := envSetupOptions{
		Retain:   *retain,
		Existing: *existing,
		Verbose:  *verbose,
		BeforeFunc: func() {
			log.Println("Installing flux")
			installFlux(ctx, kubeconfigPath, fluxInstallManifestPath)
		},
		AfterFunc: func() {
			log.Println("Uninstalling flux")
			uninstallFlux(ctx, kubeconfigPath, fluxInstallManifestPath)
		},
		CreateKubeconfig: func(state map[string]*tfjson.StateOutput, kcPath string) error {
			clusterName := state["eks_cluster_name"].Value.(string)
			eksHost := state["eks_cluster_endpoint"].Value.(string)
			eksClusterArn := state["eks_cluster_endpoint"].Value.(string)
			eksCa := state["eks_cluster_ca_certificate"].Value.(string)
			eksToken, err := getClientToken(ctx, clusterName)
			if err != nil {
				return fmt.Errorf("failed to obtain auth token: %w", err)
			}

			kubeconfigYaml := kubeconfigWithClusterAuthToken(string(eksToken), eksCa, eksHost, eksClusterArn, eksClusterArn)

			f, err := os.Create(kcPath)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(f, kubeconfigYaml)
			return err
		},
	}

	// Set up the environment and set the global kube client against the test
	// cluster.
	exitVal, err := setupEnvironmentAndRunTests(ctx, m, opts, &kubeClient)
	if err != nil {
		log.Printf("Received an error while running setup: %v", err)
		os.Exit(1)
	}

	os.Exit(exitVal)
}

// actionFunc is an action that's run during the life span of the test suite
// for configuring the environment.
type actionFunc func()

// createKubeconfig create a kubeconfig for the target cluster and writes to
// the given path using the contextual values from the infrastructure state.
type createKubeconfig func(state map[string]*tfjson.StateOutput, kcPath string) error

type envSetupOptions struct {
	Retain           bool
	Existing         bool
	Verbose          bool
	BeforeFunc       actionFunc
	AfterFunc        actionFunc
	CreateKubeconfig createKubeconfig
}

func setupEnvironmentAndRunTests(ctx context.Context, m *testing.M, opts envSetupOptions, kclient *client.Client) (exitVal int, err error) {
	// Prepare build environment.
	cwd, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get the current working directory: %w", err)
	}
	buildDir := filepath.Join(cwd, "build")
	if err := os.MkdirAll(buildDir, os.ModePerm); err != nil {
		return 0, fmt.Errorf("failed to create build directory: %w", err)
	}

	// Find or download terraform binary.
	i := install.NewInstaller()
	execPath, err := i.Ensure(ctx, []src.Source{
		&fs.AnyVersion{
			Product: &product.Terraform,
		},
		&releases.LatestVersion{
			Product:    product.Terraform,
			InstallDir: buildDir,
		},
	})
	if err != nil {
		return 0, fmt.Errorf("terraform exec path not found: %w", err)
	}
	log.Println("Terraform binary: ", execPath)

	tf, err := tfexec.NewTerraform(eksTerraformPath, execPath)
	if err != nil {
		return 0, fmt.Errorf("could not create terraform instance: %w", err)
	}

	if opts.Verbose {
		tf.SetStdout(os.Stdout)
		tf.SetStderr(os.Stderr)
	}

	log.Println("Init Terraform")
	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return 0, fmt.Errorf("error running init: %w", err)
	}

	// Destroy the infrastructure before exiting if -retain flag is false.
	if !opts.Retain {
		defer func() {
			log.Println("Tearing down test infrastructure")
			if ferr := tf.Destroy(ctx); ferr != nil {
				err = multierror.Append(fmt.Errorf("could not destroy infrastructure: %w", ferr), err)
			}
		}()
	}

	// Exit the test when existing state is found if -existing flag is false.
	if !opts.Existing {
		log.Println("Checking for an empty Terraform state")
		state, err := tf.Show(ctx)
		if err != nil {
			return 0, fmt.Errorf("could not read state: %v", err)
		}
		if state.Values != nil {
			log.Println("Found existing resources, likely from previous unsuccessful run, cleaning up...")
			return 0, fmt.Errorf("expected an empty state but got existing resources")
		}
	}

	// Apply Terraform, read the output values and construct kubeconfig.
	log.Println("Applying Terraform")
	err = tf.Apply(ctx)
	if err != nil {
		return 0, fmt.Errorf("error running apply: %v", err)
	}
	state, err := tf.Show(ctx)
	if err != nil {
		return 0, fmt.Errorf("could not read state: %v", err)
	}
	outputs := state.Values.Outputs
	opts.CreateKubeconfig(outputs, kubeconfigPath)

	// Create kube client.
	kubeCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return 0, fmt.Errorf("failed to build rest config: %w", err)
	}
	scheme := runtime.NewScheme()
	err = imagev1.AddToScheme(scheme)
	if err != nil {
		return 0, fmt.Errorf("failed to add to scheme: %w", err)
	}
	*kclient, err = client.New(kubeCfg, client.Options{Scheme: scheme})
	if err != nil {
		return 0, fmt.Errorf("failed to create new client: %w", err)
	}

	if opts.BeforeFunc != nil {
		opts.BeforeFunc()
	}

	// Run tests.
	result := m.Run()

	if opts.AfterFunc != nil {
		opts.AfterFunc()
	}

	return result, nil
}
