package activeinformer

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/bryanl/activeinformer/pkg/kubernetes"
)

type storeKey struct {
	name      string
	namespace string
}

type event struct {
	watch.Event

	Resource schema.GroupVersionResource
}

type memoryStoreResData map[storeKey]*unstructured.Unstructured
type memoryStoreData map[schema.GroupVersionResource]memoryStoreResData

// MemoryStore is a memory store. It stores objects in memory.
type MemoryStore struct {
	data memoryStoreData

	updateCh chan event
	watchers map[string]chan event

	logger logr.Logger

	mu sync.RWMutex
}

var _ kubernetes.Store = &MemoryStore{}

// NewMemoryStore creates an instance of MemoryStore.
func NewMemoryStore(optionList ...Option) *MemoryStore {
	opts := currentOptions(optionList...)

	s := MemoryStore{
		data:     memoryStoreData{},
		updateCh: make(chan event, 100),
		watchers: map[string]chan event{},
		logger:   opts.logger.WithValues("component", "MemoryStore"),
	}

	return &s
}

// Update updates the object in the memory store.
// TODO: create Add
func (s *MemoryStore) Update(res schema.GroupVersionResource, object runtime.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := object.(*unstructured.Unstructured)
	if !ok {
		s.logger.Info("store update only works with unstructured objects",
			"got", fmt.Sprintf("%T", object))
		return
	}

	m, ok := s.data[res]
	if !ok {
		m = memoryStoreResData{}
	}

	m[s.key(u)] = u
	s.data[res] = m

	s.sendUpdate(res, object, watch.Modified)
}

func (s *MemoryStore) sendUpdate(res schema.GroupVersionResource, object runtime.Object, eventType watch.EventType) {
	for _, ch := range s.watchers {
		ch <- event{
			Event: watch.Event{
				Type:   eventType,
				Object: object.DeepCopyObject(),
			},
			Resource: res,
		}
	}
}

func (s *MemoryStore) onUpdate(fn func(ch <-chan event)) {
	id := rand.String(16)
	ch := make(chan event)

	s.mu.Lock()
	s.watchers[id] = ch
	s.mu.Unlock()

	fn(ch)

	s.mu.Lock()
	delete(s.watchers, id)
	s.mu.Unlock()
}

// Delete deletes the object from the memory store.
func (s *MemoryStore) Delete(res schema.GroupVersionResource, object runtime.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := object.(*unstructured.Unstructured)
	if !ok {
		s.logger.Info("store update only works with unstructured objects",
			"got", fmt.Sprintf("%T", object))
		return
	}

	m := s.data[res]
	if m == nil {
		return
	}

	delete(m, s.key(u))

	s.data[res] = m

	if len(s.data[res]) == 0 {
		delete(s.data, res)
	}

	s.sendUpdate(res, u, watch.Deleted)
}

// List lists objects in a resource.
// TODO: support all the list option features
func (s *MemoryStore) List(res schema.GroupVersionResource, options kubernetes.ListOptions) (*unstructured.UnstructuredList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := &unstructured.UnstructuredList{}

	m, ok := s.data[res]
	if !ok {
		return list, nil
	}

	if options.Namespace == "" {
		for _, v := range m {
			list.Items = append(list.Items, *v)
		}

		return list, nil
	}

	for _, v := range m {
		if v.GetNamespace() == options.Namespace {
			list.Items = append(list.Items, *v)
		}
	}

	return list, nil
}

func (s *MemoryStore) Watch(res schema.GroupVersionResource, options kubernetes.ListOptions) (kubernetes.Watch, error) {
	ch := make(chan watch.Event)
	stopCh := make(chan bool, 1)
	w := NewWatcher(ch, stopCh)
	setupCh := make(chan bool, 1)

	s.logger.Info("memory store watch",
		"schema", res,
		"options", options)

	go func() {
		s.onUpdate(func(updateCh <-chan event) {
			setupCh <- true
			done := false
			for !done {
				select {
				case <-stopCh:
					done = true
				case e := <-updateCh:
					if res.String() == e.Resource.String() && s.isListOptionMatch(e.Object, options) {
						ch <- e.Event
					}
				}
			}

			close(ch)
			close(stopCh)
		})
	}()

	<-setupCh

	return w, nil
}

func (s *MemoryStore) isListOptionMatch(object runtime.Object, options kubernetes.ListOptions) bool {
	accessor, err := meta.Accessor(object)
	if err != nil {
		return false
	}

	// check namespace
	if n := options.Namespace; n != "" && n != accessor.GetNamespace() {
		return false

	}

	return true
}

func (s *MemoryStore) key(u *unstructured.Unstructured) storeKey {
	return storeKey{
		name:      u.GetName(),
		namespace: u.GetNamespace(),
	}
}
