package activeinformer

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/bryanl/activeinformer/pkg/kubernetes"
)

type resource struct {
	groupVersionKind schema.GroupVersionKind
	verbs            []string
	name             string
	categories       []string
	isNamespaced     bool
}

var _ kubernetes.Resource = &resource{}

func newResource(groupVersion schema.GroupVersion, apiResource metav1.APIResource) *resource {
	r := resource{
		groupVersionKind: schema.GroupVersionKind{
			Group:   groupVersion.Group,
			Version: groupVersion.Version,
			Kind:    apiResource.Kind,
		},
		verbs:        apiResource.Verbs,
		name:         apiResource.Name,
		categories:   apiResource.Categories,
		isNamespaced: apiResource.Namespaced,
	}

	return &r
}

func (r resource) GroupVersionKind() schema.GroupVersionKind {
	return r.groupVersionKind
}

func (r resource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.groupVersionKind.Group,
		Version:  r.groupVersionKind.Version,
		Resource: r.name,
	}
}

func (r resource) Verbs() []string {
	return r.verbs
}

func (r resource) Name() string {
	return r.name
}

func (r resource) Categories() []string {
	return r.categories
}

func (r resource) IsNamespaced() bool {
	return r.isNamespaced
}
