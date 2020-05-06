package clientkube

import (
	"testing"

	logrTesting "github.com/go-logr/logr/testing"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/bryanl/clientkube/pkg/cluster"
)

func TestMemoryStore_Update(t *testing.T) {
	res1 := schema.GroupVersionResource{
		Group:    "group1",
		Version:  "version",
		Resource: "resource",
	}
	res2 := schema.GroupVersionResource{
		Group:    "group2",
		Version:  "version",
		Resource: "resource",
	}

	object1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res1.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name": "object1",
			},
		},
	}
	object2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res1.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name": "object2",
			},
		},
	}
	object3 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res2.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name": "object3",
			},
		},
	}

	tests := []struct {
		name       string
		res        schema.GroupVersionResource
		object     runtime.Object
		data       memoryStoreData
		wantedData memoryStoreData
	}{
		{
			name:   "add object to empty store",
			res:    res1,
			object: object1,
			data:   memoryStoreData{},
			wantedData: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
			},
		},
		{
			name:   "overwrite object",
			res:    res1,
			object: object1,
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
			},
			wantedData: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
			},
		},
		{
			name:   "add object to existing store",
			res:    res1,
			object: object2,
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
			},
			wantedData: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
					storeKey{name: object2.GetName()}: object2,
				},
			},
		},
		{
			name:   "multiple resources",
			res:    res2,
			object: object3,
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
			},
			wantedData: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
				res2: memoryStoreResData{
					storeKey{name: object3.GetName()}: object3,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := NewMemoryStore()
			ms.data = test.data
			ms.Update(test.res, test.object)

			require.Equal(t, test.wantedData, ms.data)
		})
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	res1 := schema.GroupVersionResource{
		Group:    "group1",
		Version:  "version",
		Resource: "resource",
	}

	object1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res1.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name": "object1",
			},
		},
	}

	tests := []struct {
		name       string
		res        schema.GroupVersionResource
		object     runtime.Object
		data       memoryStoreData
		wantedData memoryStoreData
	}{
		{
			name:       "delete with no object",
			res:        res1,
			object:     object1,
			data:       memoryStoreData{},
			wantedData: memoryStoreData{},
		},
		{
			name:   "delete with existing object",
			res:    res1,
			object: object1,
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName()}: object1,
				},
			},
			wantedData: memoryStoreData{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := NewMemoryStore()
			ms.data = test.data
			ms.Delete(test.res, test.object)

			require.Equal(t, test.wantedData, ms.data)
		})
	}
}

func TestMemoryStore_List(t *testing.T) {
	res1 := schema.GroupVersionResource{
		Group:    "group1",
		Version:  "version",
		Resource: "resource",
	}

	res2 := schema.GroupVersionResource{
		Group:    "group2",
		Version:  "version",
		Resource: "resource",
	}

	object1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res1.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name":      "object1",
				"namespace": "default",
			},
		},
	}

	tests := []struct {
		name    string
		res     schema.GroupVersionResource
		options cluster.ListOptions
		object  runtime.Object
		data    memoryStoreData
		wanted  *unstructured.UnstructuredList
		wantErr bool
	}{
		{
			name:    "list resource",
			res:     res1,
			options: cluster.ListOptions{},
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName(), namespace: object1.GetNamespace()}: object1,
				},
			},
			wanted: &unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{
					*object1,
				},
			},
		},
		{
			name:    "list resource with namespace",
			res:     res1,
			options: cluster.ListOptions{Namespace: object1.GetNamespace()},
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName(), namespace: object1.GetNamespace()}: object1,
				},
			},
			wanted: &unstructured.UnstructuredList{
				Items: []unstructured.Unstructured{
					*object1,
				},
			},
		},
		{
			name:    "list resource that doesn't exist",
			res:     res2,
			options: cluster.ListOptions{},
			data: memoryStoreData{
				res1: memoryStoreResData{
					storeKey{name: object1.GetName(), namespace: object1.GetNamespace()}: object1,
				},
			},
			wanted: &unstructured.UnstructuredList{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := NewMemoryStore()
			ms.data = test.data

			actual, err := ms.List(test.res, test.options)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, test.wanted, actual)
		})
	}
}

type storeOp struct {
	eventType watch.EventType
	object    runtime.Object
	res       schema.GroupVersionResource
}

func TestMemoryStore_Watch(t *testing.T) {
	res1 := schema.GroupVersionResource{
		Group:    "group1",
		Version:  "version",
		Resource: "resource",
	}
	res2 := schema.GroupVersionResource{
		Group:    "group2",
		Version:  "version",
		Resource: "resource",
	}

	object1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res1.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name":      "object1",
				"namespace": "ns1",
			},
		},
	}
	object2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res1.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name":      "object2",
				"namespace": "ns2",
			},
		},
	}
	object3 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": res2.GroupVersion().String(),
			"kind":       "Resource",
			"metadata": map[string]interface{}{
				"name": "object3",
			},
		},
	}

	tests := []struct {
		name    string
		ops     []storeOp
		res     schema.GroupVersionResource
		options cluster.ListOptions
		wanted  []watch.Event
	}{
		{
			name: "watch all objects in a group/version/resource",
			ops: []storeOp{
				{
					eventType: watch.Modified,
					object:    object1,
					res:       res1,
				},
				{
					eventType: watch.Deleted,
					object:    object1,
					res:       res1,
				},
				{
					eventType: watch.Modified,
					object:    object3,
					res:       res2,
				},
			},
			res:     res1,
			options: cluster.ListOptions{},
			wanted: []watch.Event{
				{
					Type:   watch.Modified,
					Object: object1,
				},
				{
					Type:   watch.Deleted,
					Object: object1,
				},
			},
		},
		{
			name: "watch all objects in a group/version/resource in a namespace",
			ops: []storeOp{
				{
					eventType: watch.Modified,
					object:    object1,
					res:       res1,
				},
				{
					eventType: watch.Modified,
					object:    object2,
					res:       res1,
				},
				{
					eventType: watch.Deleted,
					object:    object1,
					res:       res1,
				},
			},
			res: res1,
			options: cluster.ListOptions{
				Namespace: object2.GetNamespace(),
			},
			wanted: []watch.Event{
				{
					Type:   watch.Modified,
					Object: object2,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ms := NewMemoryStore(WithLogger(&logrTesting.TestLogger{}))

			w, err := ms.Watch(test.res, test.options)
			require.NoError(t, err)

			var actual []watch.Event

			done := make(chan bool, 1)

			go func() {
				for e := range w.ResultChan() {
					actual = append(actual, e)
				}
				done <- true
			}()

			for _, op := range test.ops {
				switch op.eventType {
				case watch.Modified:
					ms.Update(op.res, op.object)
				case watch.Deleted:
					ms.Delete(op.res, op.object)
				default:
					t.Errorf("unknown event type %s", op.eventType)
				}
			}

			w.Stop()

			<-done

			require.Equal(t, test.wanted, actual)
		})
	}
}
