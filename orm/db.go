package orm

import "database/sql"

// DB 是orm用来管理数据库连接和缓存之类持久化内容的结构体
type DB struct {
	model *modelCache // 元数据缓存
	sqlDB *sql.DB     // 数据库连接
}

// DBOption 定义配置项
type DBOption func(*DB) error

//func NewDB(opts ...DBOption) (*DB, error) {
//	db := &DB{
//		model: NewModelCache(),
//	}
//	for _, opt := range opts {
//		opt(db)
//	}
//	return db, nil
//}

// getModel 获取元数据
func (db *DB) getModel(val any) (*model, error) {
	return db.model.get(val)
}

func Open(db *sql.DB, opts ...DBOption) (*DB, error) {
	d := &DB{
		model: NewModelCache(),
		sqlDB: db,
	}

	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}

	return d, nil
}

func OpenDB(driver, dsn string, opts ...DBOption) (*DB, error) {
	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	return Open(sqlDB, opts...)
}
