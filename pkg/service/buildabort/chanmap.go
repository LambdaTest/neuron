package buildabort

import (
	"sync"

	"github.com/LambdaTest/neuron/pkg/core"
	"github.com/LambdaTest/neuron/pkg/lumber"
)

type spawnedPodsMap struct {
	mp     *sync.Map
	logger lumber.Logger
}

// NewMap returns a new sync.Map
func NewMap(logger lumber.Logger) core.SyncMap {
	newMap := new(sync.Map)
	return &spawnedPodsMap{
		mp:     newMap,
		logger: logger,
	}
}

func (m *spawnedPodsMap) Set(key string, value chan struct{}) (chan struct{}, bool) {
	val, loaded := m.mp.LoadOrStore(key, value)
	return val.(chan struct{}), loaded
}

func (m *spawnedPodsMap) Get(key string) (chan struct{}, bool) {
	val, exist := m.mp.LoadAndDelete(key)
	if exist {
		return val.(chan struct{}), exist
	}
	return nil, exist
}

func (m *spawnedPodsMap) CleanUp(key string) {
	abortChan, exist := m.Get(key)
	if exist {
		m.logger.Debugf("found key in map: %s", key)
		close(abortChan)
	}
}
