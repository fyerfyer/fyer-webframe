# Transaction

WebFrame ORM 框架提供了灵活且强大的事务管理功能，支持传统手动管理和闭包式事务两种模式。

## 事务基础

### 事务的关键特性

事务遵循 ACID 原则：

- **原子性 (Atomicity)**：事务中的所有操作作为单个单元工作，要么全部成功，要么全部失败。
- **一致性 (Consistency)**：事务完成后，数据库状态保持一致。
- **隔离性 (Isolation)**：事务执行时不受其他事务干扰。
- **持久性 (Durability)**：一旦事务提交，其结果将永久保存。

## 使用事务

WebFrame ORM 提供了两种使用事务的方式：

### 1. 闭包式事务

闭包式事务是推荐的事务处理方式，它通过闭包函数封装事务逻辑，自动处理提交和回滚，简化了错误处理，减少了代码量：

```go
import (
    "context"
    "github.com/fyerfyer/fyer-webframe/orm"
    "log"
)

func TransferMoney(ctx context.Context, db *orm.DB, fromID, toID int64, amount float64) error {
    return db.Tx(ctx, func(tx *orm.Tx) error {
        // 从源账户扣款
        result, err := orm.RegisterUpdater[Account](tx).
            Update().
            Set(orm.Col("Balance"), orm.Raw("Balance - ?", amount)).
            Where(orm.Col("ID").Eq(fromID), orm.Col("Balance").Gte(amount)).
            Exec(ctx)
        if err != nil {
            return err
        }
        
        // 确认账户余额足够（影响行数应为1）
        rows, err := result.RowsAffected()
        if err != nil {
            return err
        }
        if rows == 0 {
            return ErrInsufficientBalance
        }
        
        // 给目标账户增加金额
        _, err = orm.RegisterUpdater[Account](tx).
            Update().
            Set(orm.Col("Balance"), orm.Raw("Balance + ?", amount)).
            Where(orm.Col("ID").Eq(toID)).
            Exec(ctx)
        if err != nil {
            return err
        }
        
        // 记录转账日志
        _, err = orm.RegisterInserter[TransactionLog](tx).
            Insert(nil, &TransactionLog{
                FromID:    fromID,
                ToID:      toID,
                Amount:    amount,
                CreatedAt: time.Now(),
            }).
            Exec(ctx)
        
        return err
    }, nil) // 第二个参数为nil，使用默认事务选项
}
```

闭包式事务的工作流程：

1. `db.Tx` 开始一个新事务
2. 执行闭包函数中的代码
3. 如果闭包返回 `nil`，事务自动提交
4. 如果闭包返回错误或发生 panic，事务自动回滚

### 2. 手动事务管理

对于需要更精细控制的场景，可以使用手动事务管理：

```go
func ManualTransfer(ctx context.Context, db *orm.DB, fromID, toID int64, amount float64) error {
    // 开始事务
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    
    // 确保事务最终会被提交或回滚
    defer func() {
        if p := recover(); p != nil {
            // 发生panic，回滚事务
            tx.RollBack()
            panic(p) // 重新抛出panic
        } else if err != nil {
            // 发生错误，回滚事务
            tx.RollBack()
        }
    }()
    
    // 执行扣款
    result, err := orm.RegisterUpdater[Account](tx).
        Update().
        Set(orm.Col("Balance"), orm.Raw("Balance - ?", amount)).
        Where(orm.Col("ID").Eq(fromID), orm.Col("Balance").Gte(amount)).
        Exec(ctx)
    if err != nil {
        return err
    }
    
    // 检查是否成功扣款
    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return ErrInsufficientBalance
    }
    
    // 执行入账
    _, err = orm.RegisterUpdater[Account](tx).
        Update().
        Set(orm.Col("Balance"), orm.Raw("Balance + ?", amount)).
        Where(orm.Col("ID").Eq(toID)).
        Exec(ctx)
    if err != nil {
        return err
    }
    
    // 记录转账日志
    _, err = orm.RegisterInserter[TransactionLog](tx).
        Insert(nil, &TransactionLog{
            FromID:    fromID,
            ToID:      toID,
            Amount:    amount,
            CreatedAt: time.Now(),
        }).
        Exec(ctx)
    if err != nil {
        return err
    }
    
    // 提交事务
    return tx.Commit()
}
```

### 事务选项

WebFrame ORM 支持通过 `sql.TxOptions` 配置事务选项：

```go
import (
    "context"
    "database/sql"
    "github.com/fyerfyer/fyer-webframe/orm"
)

func TransferWithIsolation(ctx context.Context, db *orm.DB, fromID, toID int64, amount float64) error {
    // 设置事务选项
    opts := &sql.TxOptions{
        Isolation: sql.LevelSerializable, // 使用可序列化隔离级别
        ReadOnly:  false,                 // 读写事务
    }
    
    return db.Tx(ctx, func(tx *orm.Tx) error {
        // 事务逻辑...
        return nil
    }, opts)
}
```

隔离级别选项包括：

- `sql.LevelDefault`: 使用数据库默认隔离级别
- `sql.LevelReadUncommitted`: 允许读取未提交事务的数据（最低隔离级别）
- `sql.LevelReadCommitted`: 只允许读取已提交事务的数据
- `sql.LevelRepeatableRead`: 确保多次读取相同行的结果一致
- `sql.LevelSerializable`: 最高隔离级别，完全隔离事务

## 在连接池环境中的事务

WebFrame ORM 的事务管理与连接池无缝集成。当在启用连接池的 DB 上开启事务时：

1. 从池中获取一个连接
2. 在该连接上创建事务
3. 事务完成后（提交或回滚），连接会被自动归还给池

```go
// 创建带连接池的 DB
db, err := orm.OpenDB(
    "mysql", 
    "user:password@tcp(localhost:3306)/dbname",
    "mysql",
    orm.WithConnectionPool(
        orm.WithPoolMaxIdle(10),
        orm.WithPoolMaxActive(100),
    ),
)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// 在连接池环境中使用事务
err = db.Tx(ctx, func(tx *orm.Tx) error {
    // 事务操作...
    return nil
}, nil)
```

## 事务中的选择器和构建器

在事务中使用选择器、更新器和插入器时，只需将事务对象作为第一个参数传入：

```go
err = db.Tx(ctx, func(tx *orm.Tx) error {
    // 在事务中查询数据
    user, err := orm.RegisterSelector[User](tx).
        Select().
        Where(orm.Col("ID").Eq(userID)).
        Get(ctx)
    if err != nil {
        return err
    }
    
    // 在事务中更新数据
    _, err = orm.RegisterUpdater[User](tx).
        Update().
        Set(orm.Col("Status"), "active").
        Where(orm.Col("ID").Eq(userID)).
        Exec(ctx)
    if err != nil {
        return err
    }
    
    // 在事务中插入数据
    _, err = orm.RegisterInserter[UserLog](tx).
        Insert(nil, &UserLog{
            UserID:    userID,
            Action:    "activate",
            Timestamp: time.Now(),
        }).
        Exec(ctx)
    
    return err
}, nil)
```

## 使用 Client API 的事务支持

如果使用 Client API，可以通过 `Transaction` 方法实现类似的事务支持：

```go
client := db.NewClient()
defer client.Close()

err := client.Transaction(ctx, func(tc *orm.Client) error {
    // 获取用户集合
    userCollection := tc.Collection(&User{})
    
    // 在事务中执行查询
    user, err := userCollection.Find(ctx, orm.Col("ID").Eq(userID))
    if err != nil {
        return err
    }
    
    // 执行更新
    _, err = userCollection.Update(ctx, map[string]interface{}{
        "Status": "active",
    }, orm.Col("ID").Eq(userID))
    if err != nil {
        return err
    }
    
    // 记录日志
    logCollection := tc.Collection(&UserLog{})
    _, err = logCollection.Insert(ctx, &UserLog{
        UserID:    userID,
        Action:    "activate",
        Timestamp: time.Now(),
    })
    
    return err
})
```