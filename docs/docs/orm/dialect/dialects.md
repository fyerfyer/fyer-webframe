# Dialect

WebFrame ORM 框架提供了对多种数据库方言的支持，让您能够无缝地在不同数据库系统之间切换而不需要修改应用代码。

## 方言系统概览

WebFrame ORM 通过 `Dialect` 接口统一抽象不同数据库的差异，提供了一致的开发体验：

```go
type Dialect interface {
    // BuildUpsert 构建 UPSERT 语句
    BuildUpsert(builder *strings.Builder, conflictCols []*Column, cols []*Column)

    // Quote 根据数据库方言对标识符(表名、列名等)进行引用
    Quote(name string) string

    // Placeholder 生成参数占位符
    Placeholder(index int) string

    // Concat 字符串连接函数
    Concat(items ...string) string

    // IfNull 处理空值
    IfNull(expr string, defaultVal string) string

    // DDL相关方法
    CreateTableSQL(m *model) string
    AlterTableSQL(m *model, existingTable *model) string
    TableExistsSQL(schema, table string) string
    ColumnType(f *field) string
}
```

方言注册和使用非常简单：

```go
// 注册方言 (框架内部已实现)
RegisterDialect("mysql", &Mysql{})
RegisterDialect("postgresql", &Postgresql{})
RegisterDialect("sqlite", &Sqlite{})

// 使用方言创建数据库连接
db, err := orm.OpenDB("mysql", "user:password@tcp(localhost:3306)/dbname", "mysql")
```

## MySQL 方言

### 连接配置

MySQL 方言使用标准的 DSN (数据源名称) 格式：

```go
// 创建 MySQL 连接
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local",
    "mysql",
)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

常用的 MySQL DSN 选项：

| 参数 | 说明 | 示例 |
|------|------|------|
| `charset` | 字符集 | `charset=utf8mb4` |
| `parseTime` | 自动解析时间类型 | `parseTime=true` |
| `loc` | 时区设置 | `loc=Local` |
| `timeout` | 连接超时时间 | `timeout=10s` |
| `readTimeout` | 读取超时 | `readTimeout=30s` |
| `writeTimeout` | 写入超时 | `writeTimeout=30s` |

### 标识符引用

MySQL 方言使用反引号 (`) 引用标识符：

```sql
-- MySQL生成的SQL示例
SELECT `id`, `name`, `email` FROM `users` WHERE `status` = ?;
```

### 占位符风格

MySQL 使用问号 (`?`) 作为参数占位符：

```go
// 使用MySQL方言的查询
users, err := orm.RegisterSelector[User](db).
    Select(orm.Col("ID"), orm.Col("Name")).
    Where(orm.Col("Status").Eq("active"), orm.Col("Age").Gt(18)).
    GetMulti(ctx)

// 生成的SQL为:
// SELECT `id`, `name` FROM `user` WHERE `status` = ? AND `age` > ?;
// 参数: ["active", 18]
```

### 特有函数

MySQL 方言实现了一些特有的函数：

```go
// 字符串连接
// MySQL: CONCAT(first_name, ' ', last_name)
concat := mysql.Concat("first_name", " ", "last_name")

// NULL值处理
// MySQL: IFNULL(email, 'no email')
ifnull := mysql.IfNull("email", "'no email'")

// 日期格式化
// MySQL: DATE_FORMAT(created_at, '%Y-%m-%d')
dateFormat := mysql.DateFormat("created_at", "%Y-%m-%d")
```

### UPSERT 操作

MySQL 使用 `ON DUPLICATE KEY UPDATE` 语法实现 UPSERT：

```go
// 插入用户，如果存在则更新
result, err := orm.RegisterInserter[User](db).
    Insert(nil, &user).
    Upsert(nil, []*orm.Column{
        orm.Col("Name"),
        orm.Col("Email"),
        orm.Col("Age"),
    }).
    Exec(ctx)

// 生成的SQL类似:
// INSERT INTO `user` (`id`, `name`, `email`, `age`) VALUES (?, ?, ?, ?)
// ON DUPLICATE KEY UPDATE `name` = VALUES(`name`), `email` = VALUES(`email`), `age` = VALUES(`age`);
```

### 类型映射

MySQL 方言针对 Go 类型提供了特定的 SQL 类型映射：

| Go 类型 | MySQL 类型 |
|---------|-----------|
| `bool` | `TINYINT(1)` |
| `int`, `int32` | `INT` |
| `int8` | `TINYINT` |
| `int16` | `SMALLINT` |
| `int64` | `BIGINT` |
| `uint8` | `TINYINT UNSIGNED` |
| `uint16` | `SMALLINT UNSIGNED` |
| `uint`, `uint32` | `INT UNSIGNED` |
| `uint64` | `BIGINT UNSIGNED` |
| `float32` | `FLOAT` |
| `float64` | `DOUBLE` |
| `string` | `VARCHAR(255)` |
| `[]byte` | `BLOB` |
| `time.Time` | `DATETIME` |
| `sql.NullString` | `VARCHAR(255) NULL` |
| `sql.NullInt64` | `BIGINT NULL` |
| `sql.NullFloat64` | `DOUBLE NULL` |
| `sql.NullBool` | `TINYINT(1) NULL` |
| `sql.NullTime` | `DATETIME NULL` |

### 表创建

MySQL 方言生成的建表语句包含特定的表选项：

```sql
CREATE TABLE `user` (
  `id` INT AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL,
  `email` VARCHAR(255) UNIQUE,
  `age` INT NOT NULL DEFAULT 18,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

## PostgreSQL 方言

### 连接配置

PostgreSQL 方言使用标准连接字符串格式：

```go
// 创建 PostgreSQL 连接
db, err := orm.OpenDB(
    "postgres", 
    "postgres://username:password@localhost:5432/dbname?sslmode=disable",
    "postgresql",
)
if err != nil {
    log.Fatal(err)
}
defer db.Close()
```

常用的 PostgreSQL DSN 选项：

| 参数 | 说明 | 示例 |
|------|------|------|
| `sslmode` | SSL模式 | `sslmode=disable` |
| `application_name` | 应用名称 | `application_name=myapp` |
| `connect_timeout` | 连接超时时间 | `connect_timeout=10` |
| `search_path` | schema搜索路径 | `search_path=public,users` |
| `timezone` | 会话时区 | `timezone=UTC` |
| `binary_parameters` | 使用二进制参数 | `binary_parameters=yes` |

### 标识符引用

PostgreSQL 方言使用双引号 (`"`) 引用标识符：

```sql
-- PostgreSQL生成的SQL示例
SELECT "id", "name", "email" FROM "users" WHERE "status" = $1;
```

### 占位符风格

PostgreSQL 使用 `$n` 形式的占位符，其中 n 是从 1 开始的参数索引：

```go
// 使用PostgreSQL方言的查询
users, err := orm.RegisterSelector[User](db).
    Select(orm.Col("ID"), orm.Col("Name")).
    Where(orm.Col("Status").Eq("active"), orm.Col("Age").Gt(18)).
    GetMulti(ctx)

// 生成的SQL为:
// SELECT "id", "name" FROM "user" WHERE "status" = $1 AND "age" > $2;
// 参数: ["active", 18]
```

### 特有函数

PostgreSQL 方言实现了一些特有的函数：

```go
// 字符串连接
// PostgreSQL: first_name || ' ' || last_name
concat := postgresql.Concat("first_name", " ", "last_name")

// NULL值处理
// PostgreSQL: COALESCE(email, 'no email')
ifnull := postgresql.IfNull("email", "'no email'")

// 日期格式化
// PostgreSQL: TO_CHAR(created_at, 'YYYY-MM-DD')
dateFormat := postgresql.DateFormat("created_at", "YYYY-MM-DD")

// JSON提取
// PostgreSQL: data->'info'
jsonExtract := postgresql.JsonExtract("data", "info")

// JSON文本提取
// PostgreSQL: data->>'info'
jsonExtractText := postgresql.JsonExtractText("data", "info")
```

### UPSERT 操作

PostgreSQL 使用 `ON CONFLICT` 语法实现 UPSERT：

```go
// 插入用户，如果存在则更新
result, err := orm.RegisterInserter[User](db).
    Insert(nil, &user).
    Upsert([]*orm.Column{orm.Col("ID")}, []*orm.Column{
        orm.Col("Name"),
        orm.Col("Email"),
        orm.Col("Age"),
    }).
    Exec(ctx)

// 生成的SQL类似:
// INSERT INTO "user" ("id", "name", "email", "age") VALUES ($1, $2, $3, $4)
// ON CONFLICT("id") DO UPDATE SET "name" = EXCLUDED."name", "email" = EXCLUDED."email", "age" = EXCLUDED."age";
```

### 类型映射

PostgreSQL 方言针对 Go 类型提供了特定的 SQL 类型映射：

| Go 类型 | PostgreSQL 类型 |
|---------|---------------|
| `bool` | `BOOLEAN` |
| `int`, `int32` | `INTEGER` |
| `int8` | `SMALLINT` |
| `int16` | `SMALLINT` |
| `int64` | `BIGINT` |
| `uint8` | `SMALLINT` |
| `uint16` | `INTEGER` |
| `uint`, `uint32` | `INTEGER` |
| `uint64` | `BIGINT` |
| `float32` | `REAL` |
| `float64` | `DOUBLE PRECISION` |
| `string` | `TEXT` |
| `[]byte` | `BYTEA` |
| `time.Time` | `TIMESTAMP WITH TIME ZONE` |
| `sql.NullString` | `TEXT NULL` |
| `sql.NullInt64` | `BIGINT NULL` |
| `sql.NullFloat64` | `DOUBLE PRECISION NULL` |
| `sql.NullBool` | `BOOLEAN NULL` |
| `sql.NullTime` | `TIMESTAMP WITH TIME ZONE NULL` |

### Schema 支持

PostgreSQL 方言支持 schema，可在表名查询中指定 schema：

```go
// 检查特定schema下的表是否存在
schemaSQL := postgresql.TableExistsSQL("public", "users")
// 生成的SQL为: SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'users'
```

## SQLite 方言

### 连接配置

SQLite 方言使用文件路径作为连接字符串：

```go
// 创建 SQLite 连接
db, err := orm.OpenDB(
    "sqlite3", 
    "file:database.db?cache=shared&mode=rwc",
    "sqlite",
)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// 使用内存数据库
db, err := orm.OpenDB(
    "sqlite3", 
    ":memory:",
    "sqlite",
)
```

常用的 SQLite DSN 选项：

| 参数 | 说明 | 示例 |
|------|------|------|
| `cache` | 缓存模式 | `cache=shared` |
| `mode` | 打开模式 | `mode=rwc` |
| `_journal_mode` | 日志模式 | `_journal_mode=WAL` |
| `_busy_timeout` | 忙碌超时时间(毫秒) | `_busy_timeout=5000` |
| `_foreign_keys` | 启用外键约束 | `_foreign_keys=1` |

### 标识符引用

SQLite 方言使用双引号 (`"`) 引用标识符：

```sql
-- SQLite生成的SQL示例
SELECT "id", "name", "email" FROM "users" WHERE "status" = ?;
```

### 占位符风格

SQLite 使用问号 (`?`) 作为参数占位符：

```go
// 使用SQLite方言的查询
users, err := orm.RegisterSelector[User](db).
    Select(orm.Col("ID"), orm.Col("Name")).
    Where(orm.Col("Status").Eq("active"), orm.Col("Age").Gt(18)).
    GetMulti(ctx)

// 生成的SQL为:
// SELECT "id", "name" FROM "user" WHERE "status" = ? AND "age" > ?;
// 参数: ["active", 18]
```

### 特有函数

SQLite 方言实现了一些特有的函数：

```go
// 字符串连接
// SQLite: first_name || ' ' || last_name
concat := sqlite.Concat("first_name", " ", "last_name")

// NULL值处理
// SQLite: IFNULL(email, 'no email')
ifnull := sqlite.IfNull("email", "'no email'")

// 日期格式化
// SQLite: strftime('format', created_at)
dateFormat := sqlite.DateFormat("created_at", "format")

// Julian Day
// SQLite: julianday(date_expr)
julianDay := sqlite.JulianDay("date_expr")
```

### UPSERT 操作

SQLite 使用 `ON CONFLICT` 语法实现 UPSERT：

```go
// 插入用户，如果存在则更新
result, err := orm.RegisterInserter[User](db).
    Insert(nil, &user).
    Upsert([]*orm.Column{orm.Col("ID")}, []*orm.Column{
        orm.Col("Name"),
        orm.Col("Email"),
        orm.Col("Age"),
    }).
    Exec(ctx)

// 生成的SQL类似:
// INSERT INTO "user" ("id", "name", "email", "age") VALUES (?, ?, ?, ?)
// ON CONFLICT("id") DO UPDATE SET "name" = EXCLUDED."name", "email" = EXCLUDED."email", "age" = EXCLUDED."age";
```

### 类型映射

SQLite 只有 5 种内部存储类型，对应的类型映射较为简单：

| Go 类型 | SQLite 类型 |
|---------|-----------|
| `bool` | `INTEGER` |
| `int`, `int8`, `int16`, `int32`, `int64` | `INTEGER` |
| `uint`, `uint8`, `uint16`, `uint32`, `uint64` | `INTEGER` |
| `float32`, `float64` | `REAL` |
| `string` | `TEXT` |
| `[]byte` | `BLOB` |
| `time.Time` | `TEXT` |
| `sql.NullString` | `TEXT` |
| `sql.NullInt64` | `INTEGER` |
| `sql.NullFloat64` | `REAL` |
| `sql.NullBool` | `INTEGER` |
| `sql.NullTime` | `TEXT` |

### SQLite 特殊考虑

SQLite 在某些方面与其他数据库系统有所不同：

1. **表修改限制**：SQLite 对 ALTER TABLE 语句有较多限制，不支持直接修改列定义，需要通过创建新表、复制数据、删除旧表、重命名新表的方式实现。

2. **类型系统**：SQLite 使用"亲和类型"系统，而非严格类型，即列声明为一种类型也可以存储其他类型的值。

3. **并发限制**：默认情况下，SQLite 对并发写入有限制，适合低到中等并发场景。可以通过启用 WAL 模式提升并发性能。

## 跨数据库兼容性

WebFrame ORM 的方言系统使您能够编写跨数据库兼容的代码。以下是一些确保代码在不同数据库方言间兼容的建议：

### 1. 避免方言特定的原始 SQL

尽量使用 ORM 的查询构建器而非原始 SQL：

```go
// 推荐: 使用查询构建器 (适用于所有方言)
users, err := orm.RegisterSelector[User](db).
    Select().
    Where(orm.Col("Status").Eq("active")).
    OrderBy(orm.Desc(orm.Col("CreatedAt"))).
    GetMulti(ctx)

// 不推荐: 原始SQL可能不兼容
rows, err := db.Raw(ctx, "SELECT * FROM users WHERE status = 'active' ORDER BY created_at DESC")
```

### 2. 使用内置的函数抽象

利用方言接口提供的函数抽象：

```go
// 获取当前方言
dialect := db.Dialect()

// 使用方言提供的函数
ifnullExpr := dialect.IfNull("email", "'no-email'")
concatExpr := dialect.Concat("first_name", "' '", "last_name")
```

### 3. 表结构考虑

在定义模型时考虑跨数据库兼容性：

```go
type Product struct {
    ID          int64     `orm:"primary_key;auto_increment"`
    Name        string    `orm:"size:255;nullable:false"` // 使用通用类型和约束
    Description string    `orm:"size:1000"`
    CreatedAt   time.Time `orm:"nullable:false"`
    UpdatedAt   time.Time
}
```

## 自定义方言

如果需要支持额外的数据库系统，您可以实现自定义方言：

```go
// 创建自定义方言
type MyDialect struct {
    orm.BaseDialect // 继承基本方言实现
}

// 实现必要的方法
func (d *MyDialect) Quote(name string) string {
    return "「" + name + "」" // 自定义引用风格
}

func (d *MyDialect) Placeholder(index int) string {
    return "?" // 自定义占位符风格
}

// 更多必要方法实现...

// 注册自定义方言
func init() {
    orm.RegisterDialect("mydialect", &MyDialect{})
}

// 使用自定义方言
db, err := orm.OpenDB("mydrivername", "mydsn", "mydialect")
```