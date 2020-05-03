package activeinformer

import (
	"log"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Store represents the ability to store objects.
type Store interface {
	// Update updates the object given a group/version/resource.
	Update(res schema.GroupVersionResource, object runtime.Object)
	// Update deletes the object given a group/version/resource.
	Delete(res schema.GroupVersionResource, object runtime.Object)
	// List lists objects in the store given a group/version/resource and list options.
	List(res schema.GroupVersionResource, options ListOption) (*unstructured.UnstructuredList, error)
}

type storeKey struct {
	name      string
	namespace string
}

type memoryStoreResData map[storeKey]*unstructured.Unstructured
type memoryStoreData map[schema.GroupVersionResource]memoryStoreResData

// MemoryStore is a memory store. It stores objects in memory.
type MemoryStore struct {
	data memoryStoreData

	mu sync.RWMutex
}

var _ Store = &MemoryStore{}

// NewMemoryStore creates an instance of MemoryStore.
func NewMemoryStore() *MemoryStore {
	s := MemoryStore{
		data: memoryStoreData{},
	}

	return &s
}

// Update updates the object in the memory store.
func (s *MemoryStore) Update(res schema.GroupVersionResource, object runtime.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := object.(*unstructured.Unstructured)
	if !ok {
		log.Printf("store update only works with unstructured objects; got %T", object)
		return
	}

	m, ok := s.data[res]
	if !ok {
		m = memoryStoreResData{}
	}

	m[s.key(u)] = u
	s.data[res] = m
}

// Delete deletes the object from the memory store.
func (s *MemoryStore) Delete(res schema.GroupVersionResource, object runtime.Object) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := object.(*unstructured.Unstructured)
	if !ok {
		log.Printf("store delete only works with unstructured objects; got %T", object)
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
}

// List lists objects in a resource.
// TODO: support all the list option features
func (s *MemoryStore) List(res schema.GroupVersionResource, options ListOption) (*unstructured.UnstructuredList, error) {
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

func (s *MemoryStore) key(u *unstructured.Unstructured) storeKey {
	return storeKey{
		name:      u.GetName(),
		namespace: u.GetNamespace(),
	}
}
