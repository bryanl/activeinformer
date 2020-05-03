package activeinformer

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/bryanl/activeinformer/internal/stringutil"
)

// Informer represents a cluster MemoryStoreInformer.
// NOTE: this is a poor name. It's a client as well (sans the resources)
type Informer interface {
	// Start starts the informer. Afterwards, you should call Stop.
	Start(ctx context.Context) error
	// Stop stops the informer.
	Stop() error
	// List list objects from the memory store and falls back to querying the
	// cluster directly if the resource is not synced.
	List(ctx context.Context, res schema.GroupVersionResource, options ListOption) (*unstructured.UnstructuredList, error)
}

// MemoryStoreInformer is an informer that uses a memory store.
type MemoryStoreInformer struct {
	client  Client
	synced  map[schema.GroupVersionResource]bool
	watches map[schema.GroupVersionResource]watch.Interface
	store   Store

	mu  sync.RWMutex
	sem *semaphore.Weighted
}

var _ Informer = &MemoryStoreInformer{}

// NewInformer creates an MemoryStoreInformer.
func NewInformer(client Client) *MemoryStoreInformer {
	maxWorkers := runtime.GOMAXPROCS(0)

	i := MemoryStoreInformer{
		client:  client,
		synced:  map[schema.GroupVersionResource]bool{},
		watches: map[schema.GroupVersionResource]watch.Interface{},
		store:   NewMemoryStore(),
		sem:     semaphore.NewWeighted(int64(maxWorkers)),
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
			inf.setSynced(res, w)

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

	for k := range inf.synced {
		delete(inf.synced, k)
	}

	for k, w := range inf.watches {
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
	options ListOption) (*unstructured.UnstructuredList, error) {

	inf.mu.RLock()
	defer inf.mu.RUnlock()

	if !inf.synced[res] {
		log.Printf("listing using client")
		return inf.client.List(ctx, res, options)
	}

	log.Printf("listing using store")
	return inf.store.List(res, options)
}

func (inf *MemoryStoreInformer) setSynced(res schema.GroupVersionResource, watch watch.Interface) {
	inf.mu.Lock()
	defer inf.mu.Unlock()

	inf.synced[res] = true
	inf.watches[res] = watch
}

func (inf *MemoryStoreInformer) setupWatch(ctx context.Context, res schema.GroupVersionResource) (watch.Interface, error) {
	list, err := inf.client.List(ctx, res, ListOption{})
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}

	for _, object := range list.Items {
		inf.store.Update(res, &object)
	}

	w, err := inf.client.Watch(ctx, res, ListOption{})
	if err != nil {
		return nil, fmt.Errorf("watch: %w", err)
	}

	return w, nil
}

func (inf *MemoryStoreInformer) handleWatch(res schema.GroupVersionResource, w watch.Interface) {
	for event := range w.ResultChan() {
		switch event.Type {
		case watch.Added:
			inf.store.Update(res, event.Object)
		case watch.Modified:
			inf.store.Update(res, event.Object)
		case watch.Deleted:
			inf.store.Delete(res, event.Object)
		default:
			log.Printf("watch for %s doesn't handle %s: %v",
				res.String(), event.Type, spew.Sdump(event.Object))
		}
	}
	log.Printf("watch for %s is ending", res.String())
}
