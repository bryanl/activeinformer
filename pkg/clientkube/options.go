package clientkube

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testing"

	"github.com/bryanl/clientkube/pkg/cluster"
)

type options struct {
	logger logr.Logger
	store  cluster.Store
}

func currentOptions(list ...Option) options {
	opts := options{
		logger: &testing.NullLogger{},
	}

	for _, o := range list {
		o(&opts)
	}

	return opts
}

type Option func(o *options)

func WithLogger(logger logr.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

func WithStore(store cluster.Store) Option {
	return func(o *options) {
		o.store = store
	}
}
