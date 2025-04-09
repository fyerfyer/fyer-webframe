package pool

import (
	"bytes"
	"sync"
)

// ResponseBuffer 可池化的响应缓冲区
type ResponseBuffer struct {
	Buffer *bytes.Buffer
}

// Reset 实现Poolable接口，重置缓冲区内容以便复用
func (b *ResponseBuffer) Reset() {
	b.Buffer.Reset()
}

// NewResponseBuffer 创建新的响应缓冲区
func NewResponseBuffer(initialSize int) *ResponseBuffer {
	return &ResponseBuffer{
		Buffer: bytes.NewBuffer(make([]byte, 0, initialSize)),
	}
}

// ResponseBufferPool 响应缓冲区的对象池
type ResponseBufferPool struct {
	pool       sync.Pool
	bufferSize int
}

// NewResponseBufferPool 创建响应缓冲区池
func NewResponseBufferPool(bufferSize int) *ResponseBufferPool {
	if bufferSize <= 0 {
		bufferSize = 4096 // 默认4KB
	}

	return &ResponseBufferPool{
		bufferSize: bufferSize,
		pool: sync.Pool{
			New: func() interface{} {
				return NewResponseBuffer(bufferSize)
			},
		},
	}
}

// Get 从池中获取响应缓冲区
func (p *ResponseBufferPool) Get() *ResponseBuffer {
	return p.pool.Get().(*ResponseBuffer)
}

// Put 将响应缓冲区归还池中
func (p *ResponseBufferPool) Put(b *ResponseBuffer) {
	if b == nil {
		return
	}
	b.Reset()
	p.pool.Put(b)
}

// 预定义的响应缓冲区池
var (
	// SmallBufferPool 小响应缓冲区池 (2KB)
	SmallBufferPool = NewResponseBufferPool(2 * 1024)

	// DefaultBufferPool 默认响应缓冲区池 (8KB)
	DefaultBufferPool = NewResponseBufferPool(8 * 1024)

	// LargeBufferPool 大响应缓冲区池 (32KB)
	LargeBufferPool = NewResponseBufferPool(32 * 1024)
)

// AcquireBuffer 从默认池获取响应缓冲区
func AcquireBuffer() *ResponseBuffer {
	return DefaultBufferPool.Get()
}

// ReleaseBuffer 将响应缓冲区返还默认池
func ReleaseBuffer(buf *ResponseBuffer) {
	DefaultBufferPool.Put(buf)
}

// AcquireBufferSize 根据预期大小获取合适的缓冲区
func AcquireBufferSize(size int) *ResponseBuffer {
	if size <= 2*1024 {
		return SmallBufferPool.Get()
	} else if size <= 8*1024 {
		return DefaultBufferPool.Get()
	} else {
		return LargeBufferPool.Get()
	}
}

// ReleaseBufferSize 根据预期大小释放缓冲区
func ReleaseBufferSize(buf *ResponseBuffer, size int) {
	if buf == nil {
		return
	}

	if size <= 2*1024 {
		SmallBufferPool.Put(buf)
	} else if size <= 8*1024 {
		DefaultBufferPool.Put(buf)
	} else {
		LargeBufferPool.Put(buf)
	}
}