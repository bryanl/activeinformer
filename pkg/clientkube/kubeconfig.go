package clientkube

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

// FindKubeconfig finds a kubeconfig file. It looks for a KUBECONFIG
// environment variable or in Kubernetes' default location for
// kubeconfig files on disk.
func FindKubeconfig() (string, error) {
	if p := os.Getenv("KUBECONFIG"); p != "" {
		return p, nil
	}

	if home := homedir.HomeDir(); home != "" {
		return filepath.Join(home, ".kube", "config"), nil
	}

	return "", fmt.Errorf("unable to find kubeconfig in env or at default location")
}
