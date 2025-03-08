package orm

// FindOptions 定义查询选项
type FindOptions struct {
	Offset  int
	Limit   int
	OrderBy []OrderBy
}

// FindOption 是FindOptions的构建器选项
type FindOption func(*FindOptions)

// WithLimit 设置查询结果数量限制
func WithLimit(limit int) FindOption {
	return func(o *FindOptions) {
		o.Limit = limit
	}
}

// WithOffset 设置查询结果的偏移量
func WithOffset(offset int) FindOption {
	return func(o *FindOptions) {
		o.Offset = offset
	}
}

// WithOrderBy 设置结果排序方式
func WithOrderBy(orderBy ...OrderBy) FindOption {
	return func(o *FindOptions) {
		o.OrderBy = orderBy
	}
}

// UpdateOptions 定义更新选项
type UpdateOptions struct {
	ReturnOld bool
}

// UpdateOption 是UpdateOptions的构建器选项
type UpdateOption func(*UpdateOptions)

// WithReturnOld 设置是否返回更新前的文档
func WithReturnOld(returnOld bool) UpdateOption {
	return func(o *UpdateOptions) {
		o.ReturnOld = returnOld
	}
}

// InsertOptions 定义插入选项
type InsertOptions struct {
	ReturnID   bool
	IgnoreDups bool
}

// InsertOption 是InsertOptions的构建器选项
type InsertOption func(*InsertOptions)

// WithReturnID 设置是否返回插入后的ID
func WithReturnID(returnID bool) InsertOption {
	return func(o *InsertOptions) {
		o.ReturnID = returnID
	}
}

// WithIgnoreDups 设置是否忽略重复键错误
func WithIgnoreDups(ignoreDups bool) InsertOption {
	return func(o *InsertOptions) {
		o.IgnoreDups = ignoreDups
	}
}

// DeleteOptions 定义删除选项
type DeleteOptions struct {
	Limit int
}

// DeleteOption 是DeleteOptions的构建器选项
type DeleteOption func(*DeleteOptions)

// WithDeleteLimit 设置删除的最大记录数
func WithDeleteLimit(limit int) DeleteOption {
	return func(o *DeleteOptions) {
		o.Limit = limit
	}
}