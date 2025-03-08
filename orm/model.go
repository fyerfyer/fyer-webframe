package orm

import (
	"sync"
)

// ModelRegistry 用于注册和管理模型
type ModelRegistry struct {
	models map[string]interface{}
	mu     sync.RWMutex
}

// NewModelRegistry 创建一个新的模型注册表
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		models: make(map[string]interface{}),
	}
}

// Register 注册一个模型
func (r *ModelRegistry) Register(name string, model interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[name] = model
}

// Get 获取一个模型
func (r *ModelRegistry) Get(name string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.models[name]
	return model, ok
}

// ModelNamer 定义了获取模型名称的接口
type ModelNamer interface {
	ModelName() string
}

// DefaultModelRegistry 是默认的模型注册表
var DefaultModelRegistry = NewModelRegistry()

// Model 定义了模型的基本信息和CRUD操作
type Model struct {
	name      string
	prototype interface{}
	client    *Client
}

// NewModel 创建一个新的模型
func NewModel(name string, prototype interface{}, client *Client) *Model {
	return &Model{
		name:      name,
		prototype: prototype,
		client:    client,
	}
}

// Register 注册模型到默认注册表
func Register(name string, model interface{}) {
	DefaultModelRegistry.Register(name, model)
}

// GetModelName 尝试获取模型的名称
// 首先尝试使用ModelNamer接口
// 如果没有实现该接口，则使用参数中传入的名称
func GetModelName(model interface{}, defaultName string) string {
	if namer, ok := model.(ModelNamer); ok {
		return namer.ModelName()
	}

	// 尝试获取指针的接收者实现
	if modelPtr, ok := (interface{})(&model).(ModelNamer); ok {
		return modelPtr.ModelName()
	}

	return defaultName
}

// ModelRef 模型引用，用于构建查询
type ModelRef struct {
	name     string
	model    interface{}
	registry *ModelRegistry
}

// NewModelRef 创建一个新的模型引用
func NewModelRef(name string, registry *ModelRegistry) *ModelRef {
	model, _ := registry.Get(name)
	return &ModelRef{
		name:     name,
		model:    model,
		registry: registry,
	}
}

// GetName 获取模型名称
func (m *ModelRef) GetName() string {
	return m.name
}

// GetModel 获取模型实例
func (m *ModelRef) GetModel() interface{} {
	return m.model
}

// GetRegistry 获取模型所在的注册表
func (m *ModelRef) GetRegistry() *ModelRegistry {
	return m.registry
}

// ModelOption 定义了模型操作的选项
type ModelOption func(*ModelOptions)

// ModelOptions 模型选项集合
type ModelOptions struct {
	TableName   string
	IgnoreCase  bool
	VersionField string
	PrimaryKey  string
	AutoTimestamp bool
}

// WithTableName 设置表名选项
func WithTableName(tableName string) ModelOption {
	return func(o *ModelOptions) {
		o.TableName = tableName
	}
}

// WithIgnoreCase 设置忽略大小写选项
func WithIgnoreCase(ignoreCase bool) ModelOption {
	return func(o *ModelOptions) {
		o.IgnoreCase = ignoreCase
	}
}

// WithVersionField 设置版本字段选项，用于乐观锁
func WithVersionField(field string) ModelOption {
	return func(o *ModelOptions) {
		o.VersionField = field
	}
}

// WithPrimaryKey 设置主键字段
func WithPrimaryKey(field string) ModelOption {
	return func(o *ModelOptions) {
		o.PrimaryKey = field
	}
}

// WithAutoTimestamp 设置是否自动处理创建/更新时间戳
func WithAutoTimestamp(auto bool) ModelOption {
	return func(o *ModelOptions) {
		o.AutoTimestamp = auto
	}
}

// DefaultModelOptions 返回默认的模型选项
func DefaultModelOptions() *ModelOptions {
	return &ModelOptions{
		IgnoreCase:    false,
		AutoTimestamp: true,
		PrimaryKey:    "id",
	}
}