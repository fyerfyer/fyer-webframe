package pool

import (
	"net/http"
	"sync"
)

// ContextPool 是Context对象池的实现
// 直接使用sync.Pool作为底层存储，避免过度设计
type ContextPool struct {
	pool sync.Pool
}

// CtxOptions 是创建Context时的可选参数
type CtxOptions struct {
	TplEngine     interface{} // 模板引擎
	PoolManager   interface{} // 连接池管理器
	ParamCapacity int         // 参数映射的初始容量
}

// NewContextPool 创建一个新的Context对象池
// factory 函数负责创建新的Context实例
func NewContextPool(factory func(opts CtxOptions) interface{}, opts CtxOptions) *ContextPool {
	return &ContextPool{
		pool: sync.Pool{
			New: func() interface{} {
				return factory(opts)
			},
		},
	}
}

// Get 从池中获取一个Context对象
func (p *ContextPool) Get() interface{} {
	return p.pool.Get()
}

// Put 将Context对象放回池中
// ctx必须是实现了Reset方法的对象
func (p *ContextPool) Put(ctx interface{}) {
	if resetter, ok := ctx.(Poolable); ok {
		resetter.Reset()
		p.pool.Put(ctx)
	}
}

// DefaultContextPool 全局默认的Context对象池
var DefaultContextPool *ContextPool

// InitDefaultContextPool 初始化默认的Context对象池
func InitDefaultContextPool(factory func(opts CtxOptions) interface{}, opts CtxOptions) {
	DefaultContextPool = NewContextPool(factory, opts)
}

// AcquireContext 从默认池获取Context
// req和resp用于设置获取的Context的请求和响应
// 如果默认池未初始化则会panic
func AcquireContext(req *http.Request, resp http.ResponseWriter) interface{} {
	if DefaultContextPool == nil {
		panic("DefaultContextPool is not initialized")
	}
	ctx := DefaultContextPool.Get()

	// 假设ctx有SetRequest和SetResponse方法
	// 这里使用类型断言来调用这些方法
	if setter, ok := ctx.(interface {
		SetRequest(*http.Request)
		SetResponse(http.ResponseWriter)
	}); ok {
		setter.SetRequest(req)
		setter.SetResponse(resp)
	}

	return ctx
}

// ReleaseContext 将Context放回默认池
func ReleaseContext(ctx interface{}) {
	if DefaultContextPool == nil {
		panic("DefaultContextPool is not initialized")
	}
	DefaultContextPool.Put(ctx)
}