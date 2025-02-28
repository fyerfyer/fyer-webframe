package session

import (
    "context"
    "net/http"
)

// Session 负责session具体数据的操作
type Session interface {
    Get(ctx context.Context, key string) (any, error)
    Set(ctx context.Context, key string, value any) error
    ID() string
    Touch(ctx context.Context) error  // 添加续约功能
}

// Storage 负责session的生命周期管理
type Storage interface {
    Create(ctx context.Context, id string) (Session, error)
    Refresh(ctx context.Context, id string) error
    Find(ctx context.Context, id string) (Session, error)
    Delete(ctx context.Context, id string) error
    GC(ctx context.Context) error  // 添加垃圾回收功能
    Close() error  // 添加关闭功能
}

// Propagator 负责session ID在请求和响应中的传递
type Propagator interface {
    Insert(id string, resp http.ResponseWriter) error
    Extract(req *http.Request) (string, error)
    Remove(resp http.ResponseWriter) error
}