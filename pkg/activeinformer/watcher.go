package activeinformer

import (
	"k8s.io/apimachinery/pkg/watch"
)

type Watcher struct {
	ch     chan watch.Event
	stopCh chan bool
}

var _ watch.Interface = &Watcher{}

func NewWatcher(ch chan watch.Event, stopCh chan bool) *Watcher {
	w := Watcher{
		ch:     ch,
		stopCh: stopCh,
	}

	return &w
}

func (w Watcher) Stop() {
	w.stopCh <- true
}

func (w Watcher) ResultChan() <-chan watch.Event {
	return w.ch
}
