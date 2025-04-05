# Getting Started

WebFrame ORM 框架提供了强大而灵活的对象关系映射功能，支持多种数据库方言、连接池管理、复杂查询构建和高级特性如分片、缓存等。

## 连接配置

### 初始化数据库连接

WebFrame ORM 提供了多种方式初始化数据库连接：

#### 1. 使用已存在的数据库连接

```go
import (
    "database/sql"
    "github.com/fyerfyer/fyer-webframe/orm"
    _ "github.com/go-sql-driver/mysql"
)

func initDB() (*orm.DB, error) {
    // 先创建标准库的 sql.DB
    sqlDB, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True")
    if err != nil {
        return nil, err
    }
    
    // 使用 orm.Open 创建 ORM 数据库实例
    db, err := orm.Open(sqlDB, "mysql")
    if err != nil {
        return nil, err
    }
    
    return db, nil
}
```

#### 2. 直接使用驱动和 DSN

```go
import (
    "github.com/fyerfyer/fyer-webframe/orm"
    _ "github.com/go-sql-driver/mysql"
)

func initDB() (*orm.DB, error) {
    // 使用 orm.OpenDB 直接创建连接
    db, err := orm.OpenDB(
        "mysql", 
        "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True",
        "mysql",
    )
    if err != nil {
        return nil, err
    }
    
    return db, nil
}
```

### 配置连接池

WebFrame ORM 内置了连接池管理，可以通过选项模式进行配置：

```go
import (
    "time"
    "github.com/fyerfyer/fyer-webframe/orm"
)

func initDBWithPool() (*orm.DB, error) {
    db, err := orm.OpenDB(
        "mysql", 
        "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True",
        "mysql",
        // 配置连接池
        orm.WithConnectionPool(
            orm.WithPoolMaxIdle(10),               // 最大空闲连接数
            orm.WithPoolMaxActive(100),            // 最大活动连接数
            orm.WithPoolMaxIdleTime(time.Minute),  // 连接最大空闲时间
            orm.WithPoolMaxLifetime(time.Hour),    // 连接最大生命周期
        ),
    )
    
    if err != nil {
        return nil, err
    }
    
    return db, nil
}
```

### 支持的数据库方言

WebFrame ORM 支持多种数据库方言：

```go
// MySQL
db, err := orm.Open(sqlDB, "mysql")

// PostgreSQL
db, err := orm.Open(sqlDB, "postgresql")

// SQLite
db, err := orm.Open(sqlDB, "sqlite")
```

### 关闭数据库连接

不再使用数据库时，应当关闭连接以释放资源：

```go
func closeDB(db *orm.DB) {
    if err := db.Close(); err != nil {
        log.Printf("Failed to close database: %v", err)
    }
}
```

## 模型定义

### 基础模型结构

在 WebFrame ORM 中，模型是通过 Go 结构体定义的，结构体字段对应数据库表的列：

```go
type User struct {
    ID        int       `orm:"primary_key;auto_increment"`
    Name      string    `orm:"size:255;index"`
    Email     string    `orm:"size:255;unique"`
    Age       int       `orm:"nullable:false;default:18"`
    CreatedAt time.Time `orm:"nullable:false"`
    UpdatedAt time.Time
    DeletedAt sql.NullTime
}
```

### 结构体标签

ORM 使用 orm 标签来配置字段属性：

```go
// 常用标签选项:
// primary_key: 主键
// auto_increment: 自增
// column_name: 指定列名
// size: 字段大小，如varchar(255)中的255
// nullable: 是否允许为空
// unique: 唯一约束
// index: 创建索引
// default: 默认值
// comment: 字段注释

type Product struct {
    ID          int64         `orm:"primary_key;auto_increment"`
    Name        string        `orm:"column_name:product_name;size:100;nullable:false"`
    Description string        `orm:"size:500"`
    Price       float64       `orm:"nullable:false"`
    Stock       int           `orm:"default:0"`
    IsActive    bool          `orm:"default:true"`
    CategoryID  sql.NullInt64 `orm:"index"`
    CreatedAt   time.Time     `orm:"nullable:false"`
}
```

### 自定义表名

默认情况下，ORM 会使用结构体名称的蛇形命名法作为表名。您可以通过实现 `TableNamer` 接口来自定义表名：

```go
// 方法一：结构体接收者
type Order struct {
    ID        int64
    UserID    int64
    Amount    float64
    Status    string
    CreatedAt time.Time
}

func (Order) TableName() string {
    return "t_orders"
}

// 方法二：指针接收者
type Customer struct {
    ID   int64
    Name string
    Age  int
}

func (*Customer) TableName() string {
    return "t_customers"
}
```

### 模型注册和元数据缓存

WebFrame ORM 会自动缓存模型元数据，提高性能：

```go
// 第一次使用模型时会解析并缓存元数据
user := &User{}
selector := orm.RegisterSelector[User](db).Select().Where(orm.Col("ID").Eq(1))
```

## CRUD 操作

WebFrame ORM 提供了类型安全的 API 进行增删改查操作。

### 查询操作

#### 查询单条记录

```go
func getUserByID(ctx context.Context, db *orm.DB, id int) (*User, error) {
    // 创建选择器
    selector := orm.RegisterSelector[User](db)
    
    // 构建查询
    user, err := selector.
        Select(orm.Col("ID"), orm.Col("Name"), orm.Col("Email"), orm.Col("Age")). // 指定列
        Where(orm.Col("ID").Eq(id)).                                             // 添加条件
        Get(ctx)                                                                 // 执行查询
    
    return user, err
}
```

#### 查询多条记录

```go
func getActiveUsers(ctx context.Context, db *orm.DB) ([]*User, error) {
    selector := orm.RegisterSelector[User](db)
    
    users, err := selector.
        Select().                                        // 选择所有列
        Where(orm.Col("Age").Gt(18)).                    // 年龄大于18
        OrderBy(orm.Desc(orm.Col("CreatedAt"))).         // 按创建时间降序
        Limit(10).                                       // 限制10条
        Offset(0).                                       // 从第一条开始
        GetMulti(ctx)                                    // 执行查询
    
    return users, err
}
```

#### 使用复杂条件

```go
func searchUsers(ctx context.Context, db *orm.DB, namePrefix string, minAge, maxAge int) ([]*User, error) {
    selector := orm.RegisterSelector[User](db)
    
    users, err := selector.
        Select().
        Where(
            orm.Col("Name").Like(namePrefix + "%"),           // LIKE条件
            orm.Col("Age").Between(minAge, maxAge),           // BETWEEN条件
            orm.Col("DeletedAt").IsNull(),                    // IS NULL条件
        ).
        GetMulti(ctx)
    
    return users, err
}
```

#### 聚合查询

```go
func getUserStats(ctx context.Context, db *orm.DB) (int, float64, error) {
    selector := orm.RegisterSelector[User](db)
    
    // 聚合查询 - 获取用户数量和平均年龄
    result, err := selector.
        Select(
            orm.Count("*").As("user_count"),
            orm.Avg("Age").As("avg_age"),
        ).
        Where(orm.Col("DeletedAt").IsNull()).
        Get(ctx)
    
    if err != nil {
        return 0, 0, err
    }
    
    // 处理聚合结果...
    return userCount, avgAge, nil
}
```

#### JOIN 查询

```go
func getUserOrders(ctx context.Context, db *orm.DB, userID int64) ([]*Order, error) {
    selector := orm.RegisterSelector[Order](db)
    
    orders, err := selector.
        Select().
        From(orm.Table("orders")).
        Join(orm.InnerJoin, orm.Table("users")).
        On(orm.Col("orders.user_id").Eq(orm.Col("users.id"))).
        Where(orm.Col("users.id").Eq(userID)).
        GetMulti(ctx)
    
    return orders, err
}
```

### 插入操作

#### 插入单条记录

```go
func createUser(ctx context.Context, db *orm.DB, user *User) (int64, error) {
    inserter := orm.RegisterInserter[User](db)
    
    result, err := inserter.
        Insert(nil, user).  // nil 表示插入所有字段
        Exec(ctx)
    
    if err != nil {
        return 0, err
    }
    
    // 获取自增ID
    id, err := result.LastInsertId()
    return id, err
}
```

#### 插入多条记录

```go
func createMultipleUsers(ctx context.Context, db *orm.DB, users []*User) error {
    inserter := orm.RegisterInserter[User](db)
    
    _, err := inserter.
        Insert(nil, users...).  // 批量插入
        Exec(ctx)
    
    return err
}
```

#### 指定插入字段

```go
func createUserWithSpecificFields(ctx context.Context, db *orm.DB, user *User) error {
    inserter := orm.RegisterInserter[User](db)
    
    _, err := inserter.
        Insert([]string{"Name", "Email", "Age"}, user).  // 只插入指定字段
        Exec(ctx)
    
    return err
}
```

#### Upsert 操作

```go
func upsertUser(ctx context.Context, db *orm.DB, user *User) error {
    inserter := orm.RegisterInserter[User](db)
    
    // MySQL 方言的 UPSERT
    _, err := inserter.
        Insert(nil, user).
        Upsert(nil, []*orm.Column{
            orm.Col("Name"),
            orm.Col("Email"),
            orm.Col("Age"),
        }).
        Exec(ctx)
    
    return err
}
```

### 更新操作

#### 单字段更新

```go
func updateUserAge(ctx context.Context, db *orm.DB, userID, newAge int) error {
    updater := orm.RegisterUpdater[User](db)
    
    _, err := updater.
        Update().
        Set(orm.Col("Age"), newAge).
        Where(orm.Col("ID").Eq(userID)).
        Exec(ctx)
    
    return err
}
```

#### 多字段更新

```go
func updateUser(ctx context.Context, db *orm.DB, userID int, name string, age int) error {
    updater := orm.RegisterUpdater[User](db)
    
    _, err := updater.
        Update().
        Set(orm.Col("Name"), name).
        Set(orm.Col("Age"), age).
        Where(orm.Col("ID").Eq(userID)).
        Exec(ctx)
    
    return err
}
```

#### 批量更新

```go
func updateMultipleUsers(ctx context.Context, db *orm.DB, beforeAge, afterAge int) (int64, error) {
    updater := orm.RegisterUpdater[User](db)
    
    result, err := updater.
        Update().
        Set(orm.Col("Age"), afterAge).
        Where(orm.Col("Age").Eq(beforeAge)).
        Limit(100).  // 限制更新的记录数
        Exec(ctx)
    
    if err != nil {
        return 0, err
    }
    
    // 获取受影响行数
    rowsAffected, err := result.RowsAffected()
    return rowsAffected, err
}
```

#### 使用 Map 批量设置

```go
func updateUserWithMap(ctx context.Context, db *orm.DB, userID int, values map[string]any) error {
    updater := orm.RegisterUpdater[User](db)
    
    _, err := updater.
        Update().
        SetMulti(values).  // 使用映射设置多个字段
        Where(orm.Col("ID").Eq(userID)).
        Exec(ctx)
    
    return err
}
```

### 删除操作

#### 按 ID 删除

```go
func deleteUser(ctx context.Context, db *orm.DB, userID int) error {
    deleter := orm.RegisterDeleter[User](db)
    
    _, err := deleter.
        Delete().
        Where(orm.Col("ID").Eq(userID)).
        Exec(ctx)
    
    return err
}
```

#### 条件删除

```go
func deleteInactiveUsers(ctx context.Context, db *orm.DB, inactiveDays int) (int64, error) {
    deleter := orm.RegisterDeleter[User](db)
    
    threshold := time.Now().AddDate(0, 0, -inactiveDays)
    
    result, err := deleter.
        Delete().
        Where(orm.Col("LastLoginAt").Lt(threshold)).
        Exec(ctx)
    
    if err != nil {
        return 0, err
    }
    
    return result.RowsAffected()
}
```

#### 限制删除数量

```go
func deleteBatchUsers(ctx context.Context, db *orm.DB, condition *orm.Predicate, batchSize int) error {
    deleter := orm.RegisterDeleter[User](db)
    
    _, err := deleter.
        Delete().
        Where(condition).
        Limit(batchSize).  // 限制每次删除的记录数
        Exec(ctx)
    
    return err
}
```

## 事务处理

WebFrame ORM 提供了简便的事务处理机制：

### 使用事务闭包

```go
func transferMoney(ctx context.Context, db *orm.DB, fromID, toID int, amount float64) error {
    // 使用事务闭包
    return db.Tx(ctx, func(tx *orm.Tx) error {
        // 扣款
        updater1 := orm.RegisterUpdater[Account](tx)
        _, err := updater1.
            Update().
            Set(orm.Col("Balance"), orm.Raw("balance - ?", amount)).
            Where(orm.Col("UserID").Eq(fromID), orm.Col("Balance").Gte(amount)).
            Exec(ctx)
        
        if err != nil {
            return err
        }
        
        // 确认扣款成功（受影响行数应为1）
        affRows, err := updater1.RowsAffected()
        if err != nil {
            return err
        }
        if affRows == 0 {
            return errors.New("insufficient balance")
        }
        
        // 入账
        updater2 := orm.RegisterUpdater[Account](tx)
        _, err = updater2.
            Update().
            Set(orm.Col("Balance"), orm.Raw("balance + ?", amount)).
            Where(orm.Col("UserID").Eq(toID)).
            Exec(ctx)
        
        return err
    }, nil) // 第二个参数为 nil，使用默认事务选项
}
```

### 手动管理事务

```go
func complexTransaction(ctx context.Context, db *orm.DB) error {
    // 开启事务
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    // 确保事务最终会被回滚或提交
    defer func() {
        if err != nil {
            tx.RollBack() // 如果有错误，回滚事务
        }
    }()
    
    // 执行第一个操作
    inserter := orm.RegisterInserter[User](tx)
    _, err = inserter.Insert(nil, &User{Name: "New User", Age: 30}).Exec(ctx)
    if err != nil {
        return err
    }
    
    // 执行第二个操作
    updater := orm.RegisterUpdater[User](tx)
    _, err = updater.Update().Set(orm.Col("Age"), 40).Where(orm.Col("Name").Eq("Existing User")).Exec(ctx)
    if err != nil {
        return err
    }
    
    // 提交事务
    return tx.Commit()
}
```

## ORM Client 使用

除了上面介绍的底层 API，WebFrame ORM 还提供了一个名为 `Client` 的高级封装，用于简化数据库操作。

### 创建 Client

从已有的 DB 实例创建 Client：

```go
import "github.com/fyerfyer/fyer-webframe/orm"

func main() {
    // 初始化数据库连接
    db, err := orm.OpenDB(
        "mysql", 
        "user:password@tcp(localhost:3306)/dbname?charset=utf8mb4&parseTime=True",
        "mysql",
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建客户端
    client := db.NewClient()
    defer client.Close()
    
    // 使用客户端进行操作
    // ...
}
```

### 使用 Collection 操作模型

Client 提供了基于集合的 API，可以对特定模型进行操作：

```go
// 定义模型
type User struct {
    ID      int64  `orm:"primary_key;auto_increment"`
    Name    string `orm:"size:255"`
    Email   string `orm:"size:255;unique"`
    Age     int    
    Created time.Time
}

func useCollection(client *orm.Client) {
    ctx := context.Background()
    
    // 获取用户集合
    userCollection := client.Collection(&User{})
    
    // 查找单个用户
    user, err := userCollection.Find(ctx, orm.Col("ID").Eq(123))
    if err != nil {
        log.Printf("Find user error: %v", err)
        return
    }
    userData := user.(*User)
    fmt.Printf("Found user: %s\n", userData.Name)
    
    // 查找多个用户
    users, err := userCollection.FindAll(ctx, orm.Col("Age").Gt(18))
    if err != nil {
        log.Printf("Find users error: %v", err)
        return
    }
    
    for _, u := range users {
        user := u.(*User)
        fmt.Printf("User: %s, Age: %d\n", user.Name, user.Age)
    }
    
    // 创建新用户
    newUser := &User{
        Name:    "New User",
        Email:   "newuser@example.com",
        Age:     25,
        Created: time.Now(),
    }
    
    result, err := userCollection.Insert(ctx, newUser)
    if err != nil {
        log.Printf("Insert user error: %v", err)
        return
    }
    
    id, _ := result.LastInsertId()
    fmt.Printf("Created user with ID: %d\n", id)
    
    // 更新用户
    updates := map[string]interface{}{
        "Name": "Updated Name",
        "Age":  30,
    }
    
    result, err = userCollection.Update(ctx, updates, orm.Col("ID").Eq(id))
    if err != nil {
        log.Printf("Update user error: %v", err)
        return
    }
    
    affected, _ := result.RowsAffected()
    fmt.Printf("Updated %d users\n", affected)
    
    // 删除用户
    result, err = userCollection.Delete(ctx, orm.Col("ID").Eq(id))
    if err != nil {
        log.Printf("Delete user error: %v", err)
        return
    }
    
    affected, _ = result.RowsAffected()
    fmt.Printf("Deleted %d users\n", affected)
}
```

### 使用高级查询选项

Collection 支持使用 FindOptions 进行更复杂的查询：

```go
func advancedQuery(client *orm.Client) {
    ctx := context.Background()
    userCollection := client.Collection(&User{})
    
    // 创建查询选项
    options := orm.FindOptions{
        Limit:  10,
        Offset: 20,
        OrderBy: []orm.OrderBy{
            orm.Desc(orm.Col("Created")),
            orm.Asc(orm.Col("Name")),
        },
    }
    
    // 使用选项查询
    users, err := userCollection.FindWithOptions(
        ctx, 
        options, 
        orm.Col("Age").Between(18, 30),
    )
    if err != nil {
        log.Printf("Advanced query error: %v", err)
        return
    }
    
    fmt.Printf("Found %d users\n", len(users))
}
```

### 事务支持

Client 支持事务操作：

```go
func transactionExample(client *orm.Client) {
    ctx := context.Background()
    
    err := client.Transaction(ctx, func(tc *orm.Client) error {
        // 获取事务中的集合
        userCollection := tc.Collection(&User{})
        
        // 创建用户
        newUser := &User{
            Name:    "Transaction User",
            Email:   "txuser@example.com",
            Age:     28,
            Created: time.Now(),
        }
        
        result, err := userCollection.Insert(ctx, newUser)
        if err != nil {
            return err // 返回错误会导致事务回滚
        }
        
        id, _ := result.LastInsertId()
        
        // 在同一事务中更新用户
        updates := map[string]interface{}{
            "Email": "updated@example.com",
        }
        
        _, err = userCollection.Update(ctx, updates, orm.Col("ID").Eq(id))
        if err != nil {
            return err // 返回错误会导致事务回滚
        }
        
        return nil // 返回nil会提交事务
    })
    
    if err != nil {
        log.Printf("Transaction failed: %v", err)
    } else {
        fmt.Println("Transaction committed successfully")
    }
}
```

### 原始 SQL 查询

Client 也支持执行原始 SQL 查询：

```go
func rawSQL(client *orm.Client) {
    ctx := context.Background()
    
    // 执行原始查询
    rows, err := client.Raw(ctx, "SELECT id, name FROM users WHERE age > ?", 25)
    if err != nil {
        log.Printf("Raw query error: %v", err)
        return
    }
    defer rows.Close()
    
    // 处理结果
    for rows.Next() {
        var id int64
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            log.Printf("Scan error: %v", err)
            continue
        }
        fmt.Printf("ID: %d, Name: %s\n", id, name)
    }
    
    // 执行原始命令
    result, err := client.Exec(ctx, "UPDATE users SET status = ? WHERE age < ?", "inactive", 18)
    if err != nil {
        log.Printf("Raw exec error: %v", err)
        return
    }
    
    affected, _ := result.RowsAffected()
    fmt.Printf("Updated %d users\n", affected)
    
    // 计数查询便捷方法
    count, err := client.Count(ctx, &User{}, orm.Col("Age").Gt(30))
    if err != nil {
        log.Printf("Count error: %v", err)
        return
    }
    fmt.Printf("Users over 30: %d\n", count)
}
```