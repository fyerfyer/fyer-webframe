# Data Migration

WebFrame ORM 提供了强大的数据迁移功能，帮助您自动创建和更新数据库表结构以匹配 Go 结构体定义。

## 自动迁移

自动迁移是 WebFrame ORM 的核心功能之一，它能够根据定义的 Go 结构体模型自动创建和更新数据库表结构。

### 基础用法

最简单的自动迁移方式是使用 `AutoMigrate` 方法：

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/fyerfyer/fyer-webframe/orm"
)

// 用户模型
type User struct {
    ID        int       `orm:"primary_key;auto_increment"`
    Name      string    `orm:"size:255;index"`
    Email     string    `orm:"size:255;unique"`
    Age       int       `orm:"nullable:false;default:18"`
    CreatedAt time.Time `orm:"nullable:false"`
    UpdatedAt time.Time
    DeletedAt sql.NullTime
}

func main() {
    // 连接数据库
    db, err := orm.OpenDB("mysql", "user:password@tcp(localhost:3306)/testdb", "mysql")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 执行自动迁移
    err = db.MigrateModel(context.Background(), &User{})
    if err != nil {
        log.Fatal(err)
    }

    log.Println("Migration completed successfully")
}
```

### 自动迁移多个模型

可以同时迁移多个模型：

```go
// 定义多个模型
type Product struct {
    ID          int     `orm:"primary_key;auto_increment"`
    Name        string  `orm:"size:255;nullable:false"`
    Price       float64 `orm:"nullable:false"`
    Description string  `orm:"size:1000"`
}

type Category struct {
    ID   int    `orm:"primary_key;auto_increment"`
    Name string `orm:"size:255;unique"`
}

// 迁移多个模型
err = db.MigrateAll(context.Background(), orm.WithStrategy(orm.AlterIfNeeded))
```

### 迁移选项

`MigrateModel` 和 `MigrateAll` 方法支持多种选项来自定义迁移行为：

```go
// 使用选项进行迁移
err = db.MigrateModel(
    context.Background(),
    &User{},
    // 指定迁移策略
    orm.WithStrategy(orm.AlterIfNeeded),
    // 启用迁移日志
    orm.WithMigrationLog(true),
    // 指定数据库Schema (仅PostgreSQL等支持schema的数据库有效)
    orm.WithSchema("public"),
    // 迁移完成后回调
    orm.WithMigrationCallback(func(m *orm.Migration) {
        log.Printf("Migrated model: %s, version: %d", m.ModelName, m.Version)
    }),
)
```

### 注册模型并自动迁移

WebFrame ORM 允许在注册模型同时进行自动迁移：

```go
// 注册模型并自动迁移
db.RegisterModel("User", &User{}, orm.WithStrategy(orm.AlterIfNeeded))

// 注册多个模型
db.RegisterModels(map[string]interface{}{
    "User":     &User{},
    "Product":  &Product{},
    "Category": &Category{},
}, orm.WithStrategy(orm.AlterIfNeeded))
```

### 试运行模式

试运行模式允许查看迁移将执行的 SQL 语句，而不实际执行它们：

```go
// 使用试运行模式
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithDryRun(true),
    orm.WithMigrationCallback(func(m *orm.Migration) {
        // 此回调会包含将要执行的 DDL 语句
        log.Printf("Would execute: %s", m.DDL)
    }),
)
```

## 迁移策略

WebFrame ORM 提供了多种迁移策略，以满足不同的开发和部署场景需求。每种策略都有其特定的用途和行为。

### 1. CreateOnly 策略

`CreateOnly` 策略仅创建新表，不修改已存在的表。这是最安全的策略，适合生产环境。

```go
// 使用 CreateOnly 策略
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithStrategy(orm.CreateOnly),
)
```

**适用场景**：
- 生产环境中安全地添加新表
- 当您不希望修改任何现有表结构时

### 2. AlterIfNeeded 策略

`AlterIfNeeded` 策略会根据需要修改现有表结构，如添加列或修改列定义。

```go
// 使用 AlterIfNeeded 策略
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithStrategy(orm.AlterIfNeeded),
)
```

**适用场景**：
- 开发环境中逐步更新表结构
- 需要在不丢失数据的情况下更新表结构时

### 3. DropAndCreateIfChanged 策略

`DropAndCreateIfChanged` 策略在检测到表结构发生变化时，会删除并重新创建表。这可能导致数据丢失。

```go
// 使用 DropAndCreateIfChanged 策略
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithStrategy(orm.DropAndCreateIfChanged),
)
```

**适用场景**：
- 开发环境中频繁更改表结构
- 测试环境中需要保持表结构与模型完全一致

### 4. ForceRecreate 策略

`ForceRecreate` 策略强制删除并重新创建所有表，无论表结构是否变化。这将导致数据丢失。

```go
// 使用 ForceRecreate 策略
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithStrategy(orm.ForceRecreate),
)
```

**适用场景**：
- 开发环境中完全重置数据库
- 单元测试，需要每次测试前重置数据库状态

### 策略选择指南

| 策略 | 数据安全性 | 适用环境 | 备注 |
|------|------------|----------|------|
| `CreateOnly` | 最高 | 生产环境 | 只创建新表，不修改现有表 |
| `AlterIfNeeded` | 中等 | 开发/测试 | 根据需要修改表结构，尝试保留数据 |
| `DropAndCreateIfChanged` | 低 | 开发 | 表结构变化时删除并重建表 |
| `ForceRecreate` | 最低 | 开发/测试 | 强制重建所有表，总是删除数据 |

## 迁移日志

WebFrame ORM 可以维护迁移操作的日志，记录每次模式变更，这对于追踪数据库历史变更非常有用。

### 启用迁移日志

```go
// 启用迁移日志
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithMigrationLog(true),
)
```

启用后，ORM 会创建一个 `migration_logs` 表来跟踪所有迁移操作，包含以下信息：

- **模型名称**：被迁移的模型
- **表名**：对应的数据库表
- **版本号**：迁移版本
- **创建时间**：迁移记录创建时间
- **应用时间**：迁移实际执行时间
- **DDL 语句**：执行的 DDL 语句
- **校验和**：DDL 语句的校验和，用于检测变化

### 迁移回调

使用回调函数可以在迁移操作完成后执行自定义逻辑：

```go
// 使用迁移回调
err = db.MigrateModel(
    context.Background(),
    &User{},
    orm.WithMigrationCallback(func(m *orm.Migration) {
        // 记录迁移信息
        log.Printf("Migrated: %s (v%d) at %v", m.ModelName, m.Version, m.AppliedAt)
        log.Printf("DDL: %s", m.DDL)
    }),
)
```

## 高级迁移场景

### 处理不同的数据库方言

每种数据库方言在迁移时生成的 DDL 语句可能不同。WebFrame ORM 会根据您使用的方言生成适当的 SQL：

```go
// MySQL 方言
dbMySQL, _ := orm.OpenDB("mysql", "user:password@tcp(localhost:3306)/testdb", "mysql")
err = dbMySQL.MigrateModel(context.Background(), &User{})

// PostgreSQL 方言 
dbPG, _ := orm.OpenDB("postgres", "postgres://user:password@localhost/testdb", "postgresql")
err = dbPG.MigrateModel(context.Background(), &User{})

// SQLite 方言
dbSQLite, _ := orm.OpenDB("sqlite3", "file:test.db", "sqlite")
err = dbSQLite.MigrateModel(context.Background(), &User{})
```

### 处理复杂的表结构变更

某些复杂的表结构变更（如重命名列、更改列类型）可能无法通过自动迁移处理。在这种情况下，您可以：

1. 使用原始 SQL 执行：

```go
// 执行复杂的变更
result, err := db.Exec(context.Background(), 
    "ALTER TABLE users RENAME COLUMN old_name TO new_name")
```

2. 分步执行迁移：

```go
// 1. 创建新结构的表
err = db.MigrateModel(context.Background(), &NewModel{})

// 2. 迁移数据
_, err = db.Exec(context.Background(), 
    "INSERT INTO new_table SELECT * FROM old_table")

// 3. 删除旧表
_, err = db.Exec(context.Background(), "DROP TABLE old_table")
```

## 迁移限制

理解自动迁移的一些限制是很重要的：

1. **无法智能处理所有类型的变更**：重命名表/列、更改主键等可能需要手动处理。

2. **不同数据库引擎有不同限制**：例如，SQLite 对 ALTER TABLE 操作有严格限制。

3. **不能处理复杂的数据转换**：如果列类型变更需要数据转换，您需要手动处理。

4. **无法回滚**：WebFrame ORM 当前不提供自动回滚迁移的功能，因此备份非常重要。
