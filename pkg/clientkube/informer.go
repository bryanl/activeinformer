package clientkube

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/bryanl/clientkube/internal/stringutil"
	"github.com/bryanl/clientkube/pkg/cluster"
)

type watchDescriptor struct {
	watch   *UpdatableWatcher
	options cluster.ListOptions
}

// Informer represents a cluster MemoryStoreInformer.
// NOTE: this is a poor name. It's a client as well (sans the resources)
type Informer interface {
	// Start starts the informer. Afterwards, you should call Stop.
	Start(ctx context.Context) error
	// Stop stops the informer.
	Stop() error
	// List list objects from the memory store and falls back to querying the
	// cluster directly if the resource is not synced.
	List(ctx context.Context, res schema.GroupVersionResource, options cluster.ListOptions) (*unstructured.UnstructuredList, error)
}

// MemoryStoreInformer is an informer that uses a memory store.
type MemoryStoreInformer struct {
	client           cluster.Client
	synced           map[schema.GroupVersionResource]bool
	apiWatches       map[schema.GroupVersionResource]cluster.Watch
	watchDescriptors map[schema.GroupVersionResource]watchDescriptor
	store            cluster.Store
	logger           logr.Logger

	mu  sync.RWMutex
	sem *semaphore.Weighted
}

var _ Informer = &MemoryStoreInformer{}
var _ cluster.Client = &MemoryStoreInformer{}

// NewInformer creates an MemoryStoreInformer.
func NewInformer(client cluster.Client, optionList ...Option) *MemoryStoreInformer {
	opts := currentOptions(optionList...)

	maxWorkers := runtime.GOMAXPROCS(0)

	i := MemoryStoreInformer{
		client:           client,
		synced:           map[schema.GroupVersionResource]bool{},
		apiWatches:       map[schema.GroupVersionResource]cluster.Watch{},
		watchDescriptors: map[schema.GroupVersionResource]watchDescriptor{},
		store:            opts.store,
		logger:           opts.logger.WithValues("component", "MemoryStoreInformer"),
		sem:              semaphore.NewWeighted(int64(maxWorkers)),
	}

	if i.store == nil {
		i.store = NewMemoryStore(WithLogger(opts.logger))
	}

	return &i
}

// Start starts the informer. Afterwards, you should call Stop.
func (inf *MemoryStoreInformer) Start(ctx context.Context) error {
	resourceList, err := inf.client.Resources()
	if err != nil {
		return fmt.Errorf("get resources: %w", err)
	}

	var g errgroup.Group

	for i := range resourceList {
		i := i

		// only work with resources that can be watched
		if !stringutil.Contains(resourceList[i].Verbs(), "watch") {
			continue
		}

		g.Go(func() error {
			if err := inf.sem.Acquire(ctx, 1); err != nil {
				return fmt.Errorf("acquire semaphore: %w", err)
			}

			defer inf.sem.Release(1)

			res := resourceList[i].GroupVersionResource()
			w, err := inf.setupWatch(ctx, res)
			if err != nil {
				return fmt.Errorf("setup watch %s: %w", res.String(), err)
			}

			go inf.handleWatch(res, w)
			if err := inf.SetSynced(res, w); err != nil {
				return fmt.Errorf("sync watc %s: %w", res.String(), err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("start res watches: %w", err)
	}

	return nil
}

// Stop stops the informer.
func (inf *MemoryStoreInformer) Stop() error {
	inf.mu.Lock()
	defer inf.mu.Unlock()

	inf.logger.Info("stopping")

	for k := range inf.synced {
		delete(inf.synced, k)
	}

	for k, w := range inf.apiWatches {
		w.Stop()
		delete(inf.synced, k)
	}

	return nil
}

// List list objects from the memory store and falls back to querying the
// cluster directly if the resource is not synced.
func (inf *MemoryStoreInformer) List(
	ctx context.Context,
	res schema.GroupVersionResource,
	options cluster.ListOptions) (*unstructured.UnstructuredList, error) {

	inf.mu.RLock()
	defer inf.mu.RUnlock()

	logger := inf.logger.WithValues("res", res)

	if !inf.isResourceSynced(res) {
		logger.Info("listing using client")
		return inf.client.List(ctx, res, options)
	}

	logger.Info("listing using store")
	return inf.store.List(res, options)
}

func (inf *MemoryStoreInformer) Watch(
	ctx context.Context,
	res schema.GroupVersionResource,
	options cluster.ListOptions) (cluster.Watch, error) {
	var w cluster.Watch

	if !inf.isResourceSynced(res) {
		clientWatch, err := inf.client.Watch(ctx, res, options)
		if err != nil {
			return nil, fmt.Errorf("create watch: %w", err)
		}

		w = clientWatch
	}

	if w == nil {
		storeWatcher, err := inf.store.Watch(res, options)
		if err != nil {
			return nil, fmt.Errorf("create store watcher: %w", err)
		}

		w = storeWatcher
	}

	inf.mu.Lock()
	updatableWatcher := NewUpdatableWatcher(w)
	inf.watchDescriptors[res] = watchDescriptor{
		watch:   updatableWatcher,
		options: options,
	}
	inf.mu.Unlock()

	return updatableWatcher, nil
}

func (inf *MemoryStoreInformer) Resources() (cluster.Resources, error) {
	return inf.client.Resources()
}

func (inf *MemoryStoreInformer) isResourceSynced(res schema.GroupVersionResource) bool {
	return inf.synced[res]
}

func (inf *MemoryStoreInformer) SetSynced(res schema.GroupVersionResource, apiWatch cluster.Watch) error {
	inf.mu.Lock()
	defer inf.mu.Unlock()

	inf.synced[res] = true
	inf.apiWatches[res] = apiWatch

	wd, ok := inf.watchDescriptors[res]
	if ok {
		// there was a watch created before the resource was synced

		// create a new watch from the store.
		storeWatcher, err := inf.store.Watch(res, wd.options)
		if err != nil {
			return fmt.Errorf("create store watcher: %w", err)
		}

		delete(inf.watchDescriptors, res)

		// set the existing watch
		wd.watch.SetSource(storeWatcher)
	}

	return nil
}

func (inf *MemoryStoreInformer) setupWatch(
	ctx context.Context,
	res schema.GroupVersionResource) (cluster.Watch, error) {
	list, err := inf.client.List(ctx, res, cluster.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	for _, object := range list.Items {
		inf.store.Update(res, &object)
	}

	w, err := inf.client.Watch(ctx, res, cluster.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("watch: %w", err)
	}

	return w, nil
}

func (inf *MemoryStoreInformer) handleWatch(res schema.GroupVersionResource, w cluster.Watch) {
	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Added:
			inf.store.Add(res, event.Object)
		case watch.Modified:
			inf.store.Update(res, event.Object)
		case watch.Deleted:
			inf.store.Delete(res, event.Object)
		default:
			inf.logger.Info("unknown watch event type",
				"event-type", event.Type,
				"event", event.Object)
		}
	}

	inf.logger.Info("watch is ending", "res", res)
}
