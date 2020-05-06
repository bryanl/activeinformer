package activeinformer

import (
	"k8s.io/apimachinery/pkg/watch"

	"github.com/bryanl/activeinformer/pkg/kubernetes"
)

type UpdatableWatcher struct {
	source kubernetes.Watch
}

var _ kubernetes.Watch = &UpdatableWatcher{}

func NewUpdatableWatcher(source kubernetes.Watch) *UpdatableWatcher {
	w := UpdatableWatcher{
		source: source,
	}

	return &w
}

func (w *UpdatableWatcher) SetSource(source kubernetes.Watch) {
	w.source = source
}

func (w *UpdatableWatcher) GetSource() kubernetes.Watch {
	return w.source
}

func (w *UpdatableWatcher) Stop() {
	panic("implement me")
}

func (w *UpdatableWatcher) ResultChan() <-chan watch.Event {
	panic("implement me")
}
