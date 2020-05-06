package clientkube

import (
	"k8s.io/apimachinery/pkg/watch"

	"github.com/bryanl/clientkube/pkg/cluster"
)

type UpdatableWatcher struct {
	source cluster.Watch
}

var _ cluster.Watch = &UpdatableWatcher{}

func NewUpdatableWatcher(source cluster.Watch) *UpdatableWatcher {
	w := UpdatableWatcher{
		source: source,
	}

	return &w
}

func (w *UpdatableWatcher) SetSource(source cluster.Watch) {
	w.source = source
}

func (w *UpdatableWatcher) GetSource() cluster.Watch {
	return w.source
}

func (w *UpdatableWatcher) Stop() {
	panic("implement me")
}

func (w *UpdatableWatcher) ResultChan() <-chan watch.Event {
	panic("implement me")
}
