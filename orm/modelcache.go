package orm

import (
	"reflect"
	"sync"
)

type modelCache struct {
	sync.RWMutex
	models map[reflect.Type]*model
}

func NewModelCache() *modelCache {
	return &modelCache{
		models: make(map[reflect.Type]*model),
	}
}

func (m *modelCache) get(val any) (*model, error) {
	typ := reflect.TypeOf(val)
	m.RLock()
	model, ok := m.models[typ]
	m.RUnlock()
	if ok {
		return model, nil
	}

	m.Lock()
	defer m.Unlock()
	// 双重检查
	if model, ok = m.models[typ]; ok {
		return model, nil
	}

	model, err := parseModel(val)
	if err != nil {
		return nil, err
	}
	m.models[typ] = model
	return model, nil
}
