package kubernetes

import "k8s.io/apimachinery/pkg/watch"

//go:generate mockgen -destination=../mocks/mock_watch.go -package mocks github.com/bryanl/activeinformer/pkg/kubernetes Watch

type Watch interface {
	watch.Interface
}
