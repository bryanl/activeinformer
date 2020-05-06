package kubernetes

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ListOptions wraps metav1.ListOptions and adds a Namespace key.
type ListOptions struct {
	metav1.ListOptions

	// Namespace is the namespace to scope the returned objects.
	Namespace string
}
