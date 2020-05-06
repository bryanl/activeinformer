package kubernetes

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

//go:generate mockgen -destination=../mocks/mock_client.go -package mocks github.com/bryanl/activeinformer/pkg/kubernetes Client

// Client represents a Kubernetes cluster client.
type Client interface {
	List(ctx context.Context, res schema.GroupVersionResource, options ListOptions) (*unstructured.UnstructuredList, error)
	Watch(ctx context.Context, res schema.GroupVersionResource, options ListOptions) (Watch, error)
	Resources() (Resources, error)
}
