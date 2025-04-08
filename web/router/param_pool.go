package router

import (
	"sync"
)

// ParamPool 路由参数对象池，用于减少参数map分配带来的GC压力
type ParamPool struct {
	pool sync.Pool
}

// NewParamPool 创建一个新的参数对象池
func NewParamPool() *ParamPool {
	return &ParamPool{
		pool: sync.Pool{
			New: func() interface{} {
				// 创建一个新的参数映射
				return make(map[string]string)
			},
		},
	}
}

// Get 从池中获取一个参数映射
func (p *ParamPool) Get() map[string]string {
	// 从对象池获取一个map并类型断言
	params := p.pool.Get().(map[string]string)
	return params
}

// Put 将参数映射归还到池中以便复用
func (p *ParamPool) Put(params map[string]string) {
	// 清空map的所有键值对，避免内存泄漏和数据污染
	for k := range params {
		delete(params, k)
	}
	// 将清空后的map放回池中
	p.pool.Put(params)
}

// DefaultParamPool 全局默认的参数池实例，便于包内共享使用
var DefaultParamPool = NewParamPool()

// AcquireParams 获取一个参数映射（从默认池）
func AcquireParams() map[string]string {
	return DefaultParamPool.Get()
}

// ReleaseParams 释放一个参数映射（归还到默认池）
func ReleaseParams(params map[string]string) {
	DefaultParamPool.Put(params)
}