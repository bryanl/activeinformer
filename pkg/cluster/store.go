package cluster

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//go:generate mockgen -destination=../mocks/mock_store.go -package mocks github.com/bryanl/clientkube/pkg/cluster Store

// Store represents the ability to store objects.
type Store interface {
	// Add adds an object to a group/version/resource.
	Add(res schema.GroupVersionResource, object runtime.Object)
	// Update updates the object given a group/version/resource.
	Update(res schema.GroupVersionResource, object runtime.Object)
	// Update deletes the object given a group/version/resource.
	Delete(res schema.GroupVersionResource, object runtime.Object)
	// List lists objects in the store given a group/version/resource and list options.
	List(res schema.GroupVersionResource, options ListOptions) (*unstructured.UnstructuredList, error)
	// Watch watches objects in a given group/version/resource for updates.
	Watch(res schema.GroupVersionResource, options ListOptions) (Watch, error)
}
