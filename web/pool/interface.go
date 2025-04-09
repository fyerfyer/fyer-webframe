package pool

import (
	"sync"
)

// Poolable 定义可被池化的对象接口
// 实现此接口的对象可以被对象池管理和复用
type Poolable interface {
	// Reset 重置对象状态，清除所有字段，准备复用
	Reset()
}

// Pool 定义通用对象池接口
type Pool[T Poolable] interface {
	// Get 从池中获取一个对象，如果池为空则创建新对象
	Get() T

	// Put 将对象放回池中
	Put(obj T)
}

// ObjectPool 是Pool接口的通用实现
type ObjectPool[T Poolable] struct {
	pool sync.Pool
	// 用于创建新对象的工厂函数
	newFunc func() T
}

// NewObjectPool 创建一个新的对象池
func NewObjectPool[T Poolable](newFunc func() T) *ObjectPool[T] {
	return &ObjectPool[T]{
		pool: sync.Pool{
			New: func() interface{} {
				return newFunc()
			},
		},
		newFunc: newFunc,
	}
}

// Get 从池中获取一个对象
func (p *ObjectPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put 将对象放回池中前先重置
func (p *ObjectPool[T]) Put(obj T) {
	obj.Reset()
	p.pool.Put(obj)
}