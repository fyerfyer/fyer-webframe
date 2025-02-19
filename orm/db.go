package orm

// DB 是orm用来管理数据库连接和缓存之类持久化内容的结构体
type DB struct {
	model *modelCache // 元数据缓存
}

// DBOption 定义配置项
type DBOption func(*DB)

func NewDB(opts ...DBOption) (*DB, error) {
	db := &DB{
		model: NewModelCache(),
	}
	for _, opt := range opts {
		opt(db)
	}
	return db, nil
}

// 获取元数据
func (db *DB) getModel(val any) (*model, error) {
	return db.model.get(val)
}
