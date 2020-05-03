package activeinformer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		options ListOption
		object  runtime.Object
		data    memoryStoreData
		wanted  *unstructured.UnstructuredList
		wantErr bool
	}{
		{
			name:    "list resource",
			res:     res1,
			options: ListOption{},
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
			options: ListOption{Namespace: object1.GetNamespace()},
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
			options: ListOption{},
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
