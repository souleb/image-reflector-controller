package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Based on https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html
const kubeConfigTmpl = `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %[1]s
    server: %[2]s
  name: %[3]s
contexts:
- context:
    cluster: %[3]s
    user: %[4]s
  name: %[3]s
current-context: %[3]s
kind: Config
preferences: {}
users:
- name: %[4]s
  user:
    token: %[5]s
`

func installFlux(ctx context.Context, kubeconfig, installManifest string) error {
	return runCommand(ctx, "./", fmt.Sprintf("kubectl --kubeconfig=%s apply -f %s", kubeconfig, installManifest))
}

// TODO: Maybe remove it, unless needed. Usually flux is installed once and the
// cluster is deleted. But for cases where flux deployment needs to change,
// uninstallation may be required.
func uninstallFlux(ctx context.Context, kubeconfig, installManifest string) error {
	return runCommand(ctx, "./", fmt.Sprintf("kubectl --kubeconfig=%s delete -f %s", kubeconfig, installManifest))
}

// kubeconfigWithClusterAuthToken returns a kubeconfig with the given cluster
// authentication token.
func kubeconfigWithClusterAuthToken(token, caData, endpoint, user, clusterName string) string {
	return fmt.Sprintf(kubeConfigTmpl, caData, endpoint, clusterName, user, token)
}

func getClientToken(ctx context.Context, clusterName string) ([]byte, error) {
	// err := runCommand(ctx, "/tmp/tf-aws", fmt.Sprintf("aws eks get-token --cluster-name %s | jq -r .status.token > token", clusterName))
	err := runCommand(ctx, "build", fmt.Sprintf("aws eks get-token --cluster-name %s | jq -r .status.token > token", clusterName))
	if err != nil {
		return nil, err
	}

	return os.ReadFile("build/token")
}

func runCommand(ctx context.Context, dir, command string) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(timeoutCtx, "bash", "-c", command)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failure to run command %s: %v", string(output), err)
	}
	return nil
}
