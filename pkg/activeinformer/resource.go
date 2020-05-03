package activeinformer

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource represents an API resource in the cluster.
type Resource interface {
	// GroupVersionKind returns the group/version/kind for the resource.
	GroupVersionKind() schema.GroupVersionKind
	// GroupVersionResource returns the group/version/resource for the resource.
	GroupVersionResource() schema.GroupVersionResource
	// Verbs returns the verbs that this resource understands.
	Verbs() []string
	// Name returns the API resource name for the resource.
	Name() string
	// Categories returns the categories this resource is in.
	Categories() []string
	// IsNamespaced returns true if the resource is namespaced.
	IsNamespaced() bool
}

// Resources is a list of Resource.
type Resources []Resource

// GroupVersionKind returns a resource in the list by group/version/kind.
func (rl Resources) GroupVersionKind(groupVersionKind schema.GroupVersionKind) (Resource, bool) {
	for _, r := range rl {
		if groupVersionKind.String() == r.GroupVersionKind().String() {
			return r, true
		}
	}

	return nil, false
}

// NamespacedScoped returns namespace scoped resources.
func (rl Resources) NamespacedScoped() []Resource {
	var list []Resource

	for _, r := range rl {
		if r.IsNamespaced() {
			list = append(list, r)
		}
	}

	return list
}

// ClusterScoped returns cluster scoped resources.
func (rl Resources) ClusterScoped() []Resource {
	var list []Resource

	for _, r := range rl {
		if !r.IsNamespaced() {
			list = append(list, r)
		}
	}

	return list
}

type resource struct {
	groupVersionKind schema.GroupVersionKind
	verbs            []string
	name             string
	categories       []string
	isNamespaced     bool
}

var _ Resource = &resource{}

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
