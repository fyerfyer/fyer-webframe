package orm

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Migration 表示一个数据库迁移
type Migration struct {
	ModelName   string    // 模型名称
	TableName   string    // 表名
	Version     int       // 版本号
	CreatedAt   time.Time // 创建时间
	AppliedAt   time.Time // 应用时间
	DDL         string    // DDL语句
	CheckSum    string    // 迁移内容的校验和
}

// MigrationStrategy 定义迁移策略
type MigrationStrategy int

const (
	// CreateOnly 只创建新表，不修改已存在的表
	CreateOnly MigrationStrategy = iota

	// AlterIfNeeded 如有必要修改已存在的表
	AlterIfNeeded

	// DropAndCreateIfChanged 如果表结构改变，删除并重新创建
	DropAndCreateIfChanged

	// ForceRecreate 强制删除并重新创建所有表
	ForceRecreate
)

// MigrateOptions 定义迁移选项
type MigrateOptions struct {
	Strategy           MigrationStrategy // 迁移策略
	CreateMigrationLog bool              // 是否创建迁移日志表
	DryRun             bool              // 是否为试运行模式（不实际执行SQL）
	OnMigrated         func(m *Migration) // 迁移完成后的回调
	Schema             string            // 数据库Schema（仅PostgreSQL等支持schema的数据库有效）
}

// MigrateOption 是构建MigrateOptions的函数选项
type MigrateOption func(*MigrateOptions)

// WithStrategy 设置迁移策略
func WithStrategy(strategy MigrationStrategy) MigrateOption {
	return func(o *MigrateOptions) {
		o.Strategy = strategy
	}
}

// WithDryRun 设置试运行模式
func WithDryRun(dryRun bool) MigrateOption {
	return func(o *MigrateOptions) {
		o.DryRun = dryRun
	}
}

// WithMigrationLog 设置是否创建迁移日志
func WithMigrationLog(create bool) MigrateOption {
	return func(o *MigrateOptions) {
		o.CreateMigrationLog = create
	}
}

// WithMigrationCallback 设置迁移完成后的回调
func WithMigrationCallback(callback func(m *Migration)) MigrateOption {
	return func(o *MigrateOptions) {
		o.OnMigrated = callback
	}
}

// WithSchema 设置数据库Schema
func WithSchema(schema string) MigrateOption {
	return func(o *MigrateOptions) {
		o.Schema = schema
	}
}

// SchemaManager 管理数据库架构迁移
type SchemaManager struct {
	db            *DB
	models        map[string]*model  // 已注册模型的缓存
	registry      *ModelRegistry     // 模型注册表
	migrationLogs map[string]Migration // 迁移日志缓存
	mu            sync.RWMutex
}

// NewSchemaManager 创建一个新的架构管理器
func NewSchemaManager(db *DB) *SchemaManager {
	return &SchemaManager{
		db:            db,
		models:        make(map[string]*model),
		registry:      DefaultModelRegistry,
		migrationLogs: make(map[string]Migration),
	}
}

// MigrateModel 迁移单个模型
func (sm *SchemaManager) MigrateModel(ctx context.Context, val any, opts ...MigrateOption) error {
	// 合并选项
	options := &MigrateOptions{
		Strategy:           AlterIfNeeded,
		CreateMigrationLog: true,
	}
	for _, opt := range opts {
		opt(options)
	}

	// 获取模型元数据
	m, err := sm.db.getModel(val)
	if err != nil {
		return err
	}

	// 获取模型名称
	modelName := reflect.TypeOf(val).String()
	if namer, ok := val.(ModelNamer); ok {
		modelName = namer.ModelName()
	}

	// 检查表是否存在
	tableExists, err := sm.tableExists(ctx, options.Schema, m.table)
	if err != nil {
		return fmt.Errorf("检查表是否存在失败: %w", err)
	}

	var ddl string
	var existingModel *model

	// 根据策略生成DDL
	switch options.Strategy {
	case CreateOnly:
		// 如果表已存在，不做任何操作
		if tableExists {
			return nil
		}
		ddl = sm.db.dialect.CreateTableSQL(m)

	case AlterIfNeeded:
		if tableExists {
			// 获取已存在表的结构
			existingModel, err = sm.getExistingTableModel(ctx, options.Schema, m.table)
			if err != nil {
				return fmt.Errorf("获取已存在表结构失败: %w", err)
			}

			// 比较并生成ALTER TABLE语句
			ddl = sm.db.dialect.AlterTableSQL(m, existingModel)
		} else {
			ddl = sm.db.dialect.CreateTableSQL(m)
		}

	case DropAndCreateIfChanged:
		if tableExists {
			// 获取已存在表的结构
			existingModel, err = sm.getExistingTableModel(ctx, options.Schema, m.table)
			if err != nil {
				return fmt.Errorf("获取已存在表结构失败: %w", err)
			}

			// 表结构是否变化
			if sm.isTableChanged(m, existingModel) {
				// 生成删除和创建表的SQL
				dropSQL := fmt.Sprintf("DROP TABLE %s;", sm.db.dialect.Quote(m.table))
				createSQL := sm.db.dialect.CreateTableSQL(m)
				ddl = dropSQL + "\n" + createSQL
			} else {
				// 表结构没有变化，不需要操作
				return nil
			}
		} else {
			ddl = sm.db.dialect.CreateTableSQL(m)
		}

	case ForceRecreate:
		if tableExists {
			// 无论表结构是否变化，都强制删除并重建
			dropSQL := fmt.Sprintf("DROP TABLE %s;", sm.db.dialect.Quote(m.table))
			createSQL := sm.db.dialect.CreateTableSQL(m)
			ddl = dropSQL + "\n" + createSQL
		} else {
			ddl = sm.db.dialect.CreateTableSQL(m)
		}

	default:
		return errors.New("未知的迁移策略")
	}

	// 如果没有需要执行的DDL，直接返回
	if ddl == "" {
		return nil
	}

	// 记录迁移
	migration := &Migration{
		ModelName: modelName,
		TableName: m.table,
		Version:   1, // 简单实现，实际应基于变更计算
		CreatedAt: time.Now(),
		DDL:       ddl,
		CheckSum:  calculateChecksum(ddl),
	}

	// 执行DDL
	if !options.DryRun {
		if err := sm.executeDDL(ctx, ddl); err != nil {
			return fmt.Errorf("执行DDL失败: %w", err)
		}

		// 记录迁移日志
		if options.CreateMigrationLog {
			migration.AppliedAt = time.Now()
			if err := sm.logMigration(ctx, migration); err != nil {
				return fmt.Errorf("记录迁移日志失败: %w", err)
			}
		}
	}

	// 调用回调
	if options.OnMigrated != nil {
		options.OnMigrated(migration)
	}

	return nil
}

// MigrateAll 迁移所有已注册的模型
func (sm *SchemaManager) MigrateAll(ctx context.Context, opts ...MigrateOption) error {
	sm.mu.RLock()
	registry := sm.registry
	sm.mu.RUnlock()

	// 创建迁移日志表
	options := &MigrateOptions{
		Strategy:           AlterIfNeeded,
		CreateMigrationLog: true,
	}
	for _, opt := range opts {
		opt(options)
	}

	if options.CreateMigrationLog && !options.DryRun {
		if err := sm.createMigrationTable(ctx); err != nil {
			return err
		}
	}

	// 获取所有已注册的模型
	registeredModels := make(map[string]interface{})
	for name, model := range registry.models {
		registeredModels[name] = model
	}

	// 按顺序迁移所有模型
	for _, model := range registeredModels {
		if err := sm.MigrateModel(ctx, model, opts...); err != nil {
			return fmt.Errorf("迁移模型 %T 失败: %w", model, err)
		}
	}

	return nil
}

// isTableChanged 检查表结构是否有变化
func (sm *SchemaManager) isTableChanged(newModel, existingModel *model) bool {
	// 比较列的数量
	if len(newModel.fieldsMap) != len(existingModel.fieldsMap) {
		return true
	}

	// 比较各列的定义
	for name, newField := range newModel.fieldsMap {
		oldField, exists := existingModel.fieldsMap[name]
		if !exists {
			// 新增列
			return true
		}

		// 比较列类型和属性
		if sm.db.dialect.ColumnType(newField) != sm.db.dialect.ColumnType(oldField) ||
			newField.nullable != oldField.nullable ||
			newField.default_ != oldField.default_ ||
			newField.primaryKey != oldField.primaryKey ||
			newField.unique != oldField.unique {
			return true
		}
	}

	// 检查是否有删除的列
	for name := range existingModel.fieldsMap {
		if _, exists := newModel.fieldsMap[name]; !exists {
			// 删除列
			return true
		}
	}

	return false
}

// tableExists 检查表是否存在
func (sm *SchemaManager) tableExists(ctx context.Context, schema, table string) (bool, error) {
	// 生成检查表是否存在的SQL
	query := sm.db.dialect.TableExistsSQL(schema, table)

	// 执行查询
	rows, err := sm.db.queryContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	// 如果有结果，表示表存在
	return rows.Next(), nil
}

// getExistingTableModel 从数据库中获取已存在表的结构
func (sm *SchemaManager) getExistingTableModel(ctx context.Context, schema, table string) (*model, error) {
	// 这个实现是简化的，实际应该从数据库的information_schema中读取表结构
	// 不同数据库有不同的方式获取表结构信息

	// 创建一个空的模型用于存储表结构
	m := &model{
		table:       table,
		fieldsMap:   make(map[string]*field),
		colNameMap:  make(map[string]string),
		dialect:     sm.db.dialect,
	}

	// 根据数据库类型，从系统表中查询列信息
	var query string
	switch sm.db.dialect.(type) {
	case *Mysql:
		query = fmt.Sprintf(`
            SELECT 
                COLUMN_NAME,
                DATA_TYPE,
                IS_NULLABLE,
                COLUMN_DEFAULT,
                CHARACTER_MAXIMUM_LENGTH,
                NUMERIC_PRECISION,
                NUMERIC_SCALE,
                COLUMN_KEY,
                EXTRA
            FROM 
                INFORMATION_SCHEMA.COLUMNS
            WHERE 
                TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s'
        `, table)
	case *Postgresql:
		query = fmt.Sprintf(`
            SELECT 
                column_name,
                data_type,
                is_nullable,
                column_default,
                character_maximum_length,
                numeric_precision,
                numeric_scale
            FROM 
                information_schema.columns
            WHERE 
                table_schema = COALESCE('%s', 'public') AND table_name = '%s'
        `, schema, table)
	case *Sqlite:
		query = fmt.Sprintf(`PRAGMA table_info('%s')`, table)
	default:
		return nil, errors.New("不支持的数据库类型")
	}

	// 执行查询
	rows, err := sm.db.queryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 解析结果并填充模型
	for rows.Next() {
		var colName, dataType, isNullable, columnDefault, columnKey, extra sql.NullString
		var maxLength, precision, scale sql.NullInt64

		// 根据数据库类型处理不同的结果集结构
		switch sm.db.dialect.(type) {
		case *Mysql:
			err = rows.Scan(&colName, &dataType, &isNullable, &columnDefault, &maxLength, &precision, &scale, &columnKey, &extra)
		case *Postgresql:
			err = rows.Scan(&colName, &dataType, &isNullable, &columnDefault, &maxLength, &precision, &scale)
		case *Sqlite:
			// SQLite的PRAGMA table_info结果列是：cid, name, type, notnull, dflt_value, pk
			var cid, notNull, pk sql.NullInt64
			err = rows.Scan(&cid, &colName, &dataType, &notNull, &columnDefault, &pk)
			if notNull.Int64 == 0 {
				isNullable.String = "YES"
			} else {
				isNullable.String = "NO"
			}
			isNullable.Valid = true
			if pk.Int64 == 1 {
				columnKey.String = "PRI"
			}
			columnKey.Valid = true
		}

		if err != nil {
			return nil, err
		}

		// 创建字段
		f := &field{
			colName:    colName.String,
			nullable:   isNullable.String == "YES",
			primaryKey: columnKey.String == "PRI",
			default_:   columnDefault.String,
			autoIncr:   strings.Contains(strings.ToLower(extra.String), "auto_increment"),
			sqlType:    dataType.String,
		}

		if maxLength.Valid {
			f.size = int(maxLength.Int64)
		}

		if precision.Valid {
			f.precision = int(precision.Int64)
		}

		if scale.Valid {
			f.scale = int(scale.Int64)
		}

		// 生成一个对应的字段名
		fieldName := strings.Title(strings.ToLower(colName.String))
		m.fieldsMap[fieldName] = f
		m.colNameMap[colName.String] = fieldName
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return m, nil
}

// executeDDL 执行DDL语句
func (sm *SchemaManager) executeDDL(ctx context.Context, ddl string) error {
	// 处理可能的多条SQL语句
	for _, statement := range strings.Split(ddl, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		// 确保语句以分号结尾
		if !strings.HasSuffix(statement, ";") {
			statement += ";"
		}

		_, err := sm.db.execContext(ctx, statement)
		if err != nil {
			return fmt.Errorf("执行DDL失败: %s, 错误: %w", statement, err)
		}
	}
	return nil
}

// logMigration 记录迁移日志
func (sm *SchemaManager) logMigration(ctx context.Context, m *Migration) error {
	// 确保迁移日志表存在
	if err := sm.createMigrationTable(ctx); err != nil {
		return err
	}

	// 检查是否已存在相同的迁移记录
	query := fmt.Sprintf(`
        SELECT COUNT(*) FROM orm_migration_log 
        WHERE model_name = ? AND table_name = ? AND version = ?
    `)

	rows, err := sm.db.queryContext(ctx, query, m.ModelName, m.TableName, m.Version)
	if err != nil {
		return err
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return err
		}
	}

	// 如果已存在相同的迁移记录，更新它
	if count > 0 {
		query = `
            UPDATE orm_migration_log 
            SET ddl = ?, checksum = ?, applied_at = ?
            WHERE model_name = ? AND table_name = ? AND version = ?
        `
		_, err = sm.db.execContext(ctx, query, m.DDL, m.CheckSum, m.AppliedAt, m.ModelName, m.TableName, m.Version)
	} else {
		// 否则，插入新的记录
		query = `
            INSERT INTO orm_migration_log 
            (model_name, table_name, version, created_at, applied_at, ddl, checksum)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        `
		_, err = sm.db.execContext(ctx, query, m.ModelName, m.TableName, m.Version, m.CreatedAt, m.AppliedAt, m.DDL, m.CheckSum)
	}

	return err
}

// createMigrationTable 创建迁移日志表
func (sm *SchemaManager) createMigrationTable(ctx context.Context) error {
	// 获取建表DDL
	var ddl string

	// 根据数据库类型选择合适的DDL
	switch sm.db.dialect.(type) {
	case *Mysql:
		ddl = `
            CREATE TABLE IF NOT EXISTS orm_migration_log (
                id INT AUTO_INCREMENT PRIMARY KEY,
                model_name VARCHAR(255) NOT NULL,
                table_name VARCHAR(255) NOT NULL,
                version INT NOT NULL,
                created_at DATETIME NOT NULL,
                applied_at DATETIME NOT NULL,
                ddl TEXT NOT NULL,
                checksum VARCHAR(64) NOT NULL,
                INDEX idx_model_table_version (model_name, table_name, version)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
        `
	case *Postgresql:
		ddl = `
            CREATE TABLE IF NOT EXISTS orm_migration_log (
                id SERIAL PRIMARY KEY,
                model_name VARCHAR(255) NOT NULL,
                table_name VARCHAR(255) NOT NULL,
                version INTEGER NOT NULL,
                created_at TIMESTAMP WITH TIME ZONE NOT NULL,
                applied_at TIMESTAMP WITH TIME ZONE NOT NULL,
                ddl TEXT NOT NULL,
                checksum VARCHAR(64) NOT NULL
            );
            CREATE INDEX IF NOT EXISTS idx_model_table_version 
            ON orm_migration_log (model_name, table_name, version);
        `
	case *Sqlite:
		ddl = `
            CREATE TABLE IF NOT EXISTS orm_migration_log (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                model_name TEXT NOT NULL,
                table_name TEXT NOT NULL,
                version INTEGER NOT NULL,
                created_at DATETIME NOT NULL,
                applied_at DATETIME NOT NULL,
                ddl TEXT NOT NULL,
                checksum TEXT NOT NULL
            );
            CREATE INDEX IF NOT EXISTS idx_model_table_version 
            ON orm_migration_log (model_name, table_name, version);
        `
	default:
		return errors.New("不支持的数据库类型")
	}

	// 执行建表语句
	_, err := sm.db.execContext(ctx, ddl)
	return err
}

// calculateChecksum 计算DDL的校验和
// 简化实现，实际可以使用MD5或SHA等算法
func calculateChecksum(ddl string) string {
	h := sha256.New()
	h.Write([]byte(ddl))
	return fmt.Sprintf("%x", h.Sum(nil))
}