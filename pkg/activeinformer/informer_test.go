package activeinformer

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/go-logr/stdr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/bryanl/activeinformer/pkg/kubernetes"
	"github.com/bryanl/activeinformer/pkg/mocks"
)

func TestMemoryStoreInformer_Watch(t *testing.T) {
	res := schema.GroupVersionResource{
		Group:    "group",
		Version:  "version",
		Resource: "resource",
	}

	stdLog := log.New(os.Stderr, "", log.LstdFlags)
	loggerOption := WithLogger(stdr.New(stdLog))

	tests := []struct {
		name         string
		options      func(ctrl *gomock.Controller, options kubernetes.ListOptions) []Option
		withInformer func(informer *MemoryStoreInformer)
		listOptions  kubernetes.ListOptions
		initClient   func(ctrl *gomock.Controller, options kubernetes.ListOptions) kubernetes.Client
	}{
		{
			name: "watch for unsynced resource uses client for watch",
			options: func(ctrl *gomock.Controller, options kubernetes.ListOptions) []Option {
				return []Option{
					loggerOption,
				}
			},
			initClient: func(ctrl *gomock.Controller, options kubernetes.ListOptions) kubernetes.Client {
				w := mocks.NewMockWatch(ctrl)
				client := mocks.NewMockClient(ctrl)
				client.EXPECT().Watch(gomock.Any(), res, options).Return(w, nil)

				return client
			},
			listOptions: kubernetes.ListOptions{},
		},
		{
			name: "watch for synced resource builds watch from store",
			options: func(ctrl *gomock.Controller, options kubernetes.ListOptions) []Option {
				s := mocks.NewMockStore(ctrl)
				s.EXPECT().
					Watch(res, options)

				return []Option{
					loggerOption,
					WithStore(s),
				}
			},
			withInformer: func(informer *MemoryStoreInformer) {
				require.NoError(t, informer.SetSynced(res, nil))
			},
			initClient: func(ctrl *gomock.Controller, options kubernetes.ListOptions) kubernetes.Client {
				client := mocks.NewMockClient(ctrl)
				return client
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()

			ctrl := gomock.NewController(t)

			client := test.initClient(ctrl, test.listOptions)
			options := test.options(ctrl, test.listOptions)

			msi := NewInformer(client, options...)
			if fn := test.withInformer; fn != nil {
				fn(msi)
			}

			_, err := msi.Watch(ctx, res, test.listOptions)

			require.NoError(t, err)
		})
	}
}
