# Select Build

WebFrame ORM 查询构建系统提供了一种类型安全、灵活且直观的方式来构建数据库查询。

## 选择器

选择器（Selector）是 WebFrame ORM 中用于构建 SELECT 查询的核心组件，它提供了类型安全的 API 用于构建各种数据库查询。

### 创建选择器

```go
import "github.com/fyerfyer/fyer-webframe/orm"

func getAllUsers(ctx context.Context, db *orm.DB) ([]*User, error) {
    // 创建一个选择器
    selector := orm.RegisterSelector[User](db)
    
    // 构建查询
    users, err := selector.Select().GetMulti(ctx)
    
    return users, err
}
```

### 指定查询列

可以使用 `Select` 方法指定要查询的特定列：

```go
// 查询所有列 (SELECT * FROM users)
selector := orm.RegisterSelector[User](db).Select()

// 查询特定列 (SELECT id, name, email FROM users)
selector := orm.RegisterSelector[User](db).Select(
    orm.Col("ID"),
    orm.Col("Name"),
    orm.Col("Email"),
)
```

### 别名支持

可以为列和表设置别名：

```go
// SELECT id AS user_id, name AS user_name FROM users
selector := orm.RegisterSelector[User](db).Select(
    orm.Col("ID").As("user_id"),
    orm.Col("Name").As("user_name"),
)

// 使用表别名
selector := orm.RegisterSelector[User](db).
    From(orm.Table("users").As("u")).
    Select(orm.Col("ID").As("user_id"))
```

### 原始 SQL 表达式

支持原始 SQL 表达式：

```go
// SELECT CONCAT(first_name, ' ', last_name) AS full_name FROM users
selector := orm.RegisterSelector[User](db).Select(
    orm.Raw("CONCAT(first_name, ' ', last_name) AS full_name"),
)
```

### 执行查询

选择器提供了两种主要的执行方法：

```go
// 获取单个记录
user, err := selector.Get(ctx)

// 获取多条记录
users, err := selector.GetMulti(ctx)
```

### 子查询

选择器可以用作子查询：

```go
// 创建子查询
subQuery := orm.RegisterSelector[User](db).
    Select(orm.Col("DepartmentID")).
    Where(orm.Col("Role").Eq("manager")).
    AsSubQuery("manager_depts")

// 在主查询中使用
employees := orm.RegisterSelector[Employee](db).
    Select().
    Where(orm.Col("DepartmentID").In(subQuery)).
    GetMulti(ctx)
```

## 条件构建

WHERE 条件是查询的关键部分，WebFrame ORM 提供了丰富的条件构建 API。

### 基本比较操作

```go
// 等于 (WHERE id = 1)
selector := orm.RegisterSelector[User](db).Where(orm.Col("ID").Eq(1))

// 大于 (WHERE age > 18)
selector := orm.RegisterSelector[User](db).Where(orm.Col("Age").Gt(18))

// 小于 (WHERE age < 30)
selector := orm.RegisterSelector[User](db).Where(orm.Col("Age").Lt(30))

// 大于等于 (WHERE age >= 18)
selector := orm.RegisterSelector[User](db).Where(orm.Col("Age").Gte(18))

// 小于等于 (WHERE age <= 65)
selector := orm.RegisterSelector[User](db).Where(orm.Col("Age").Lte(65))
```

### NULL 值检查

```go
// IS NULL (WHERE last_login IS NULL)
selector := orm.RegisterSelector[User](db).Where(orm.Col("LastLogin").IsNull())

// IS NOT NULL (WHERE email IS NOT NULL)
selector := orm.RegisterSelector[User](db).Where(orm.Col("Email").NotNull())
```

### LIKE 操作符

```go
// LIKE (WHERE name LIKE 'John%')
selector := orm.RegisterSelector[User](db).Where(orm.Col("Name").Like("John%"))

// NOT LIKE (WHERE name NOT LIKE '%test%')
selector := orm.RegisterSelector[User](db).Where(orm.Col("Name").NotLike("%test%"))
```

### IN 操作符

```go
// IN (WHERE id IN (1, 2, 3))
selector := orm.RegisterSelector[User](db).Where(orm.Col("ID").In(1, 2, 3))

// NOT IN (WHERE status NOT IN ('deleted', 'banned'))
selector := orm.RegisterSelector[User](db).Where(orm.Col("Status").NotIn("deleted", "banned"))
```

### BETWEEN 操作符

```go
// BETWEEN (WHERE age BETWEEN 18 AND 65)
selector := orm.RegisterSelector[User](db).Where(orm.Col("Age").Between(18, 65))

// NOT BETWEEN (WHERE created_at NOT BETWEEN '2020-01-01' AND '2020-12-31')
startDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC)
selector := orm.RegisterSelector[User](db).Where(orm.Col("CreatedAt").NotBetween(startDate, endDate))
```

### 逻辑操作符

多个条件会自动使用 AND 连接：

```go
// WHERE age > 18 AND status = 'active'
selector := orm.RegisterSelector[User](db).Where(
    orm.Col("Age").Gt(18),
    orm.Col("Status").Eq("active"),
)
```

使用 NOT 操作符：

```go
// WHERE NOT (role = 'guest')
selector := orm.RegisterSelector[User](db).Where(
    orm.NOT(orm.Col("Role").Eq("guest")),
)
```

### 原始 SQL 条件

可以使用原始 SQL 构建更复杂的条件：

```go
// WHERE created_at > DATE_SUB(NOW(), INTERVAL 7 DAY)
selector := orm.RegisterSelector[User](db).Where(
    orm.Raw("created_at > DATE_SUB(NOW(), INTERVAL 7 DAY)"),
)
```

## 排序和分页

### 排序

使用 `OrderBy` 方法对结果进行排序：

```go
// 单列升序排序 (ORDER BY created_at ASC)
selector := orm.RegisterSelector[User](db).
    Select().
    OrderBy(orm.Asc(orm.Col("CreatedAt")))

// 单列降序排序 (ORDER BY created_at DESC)
selector := orm.RegisterSelector[User](db).
    Select().
    OrderBy(orm.Desc(orm.Col("CreatedAt")))

// 多列排序 (ORDER BY status ASC, created_at DESC)
selector := orm.RegisterSelector[User](db).
    Select().
    OrderBy(
        orm.Asc(orm.Col("Status")),
        orm.Desc(orm.Col("CreatedAt")),
    )
```

### 分页

通过 `Limit` 和 `Offset` 方法实现分页：

```go
// 限制返回的行数 (LIMIT 10)
selector := orm.RegisterSelector[User](db).
    Select().
    Limit(10)

// 设置偏移量 (OFFSET 20)
selector := orm.RegisterSelector[User](db).
    Select().
    Offset(20)

// 组合使用实现分页 (LIMIT 10 OFFSET 20)
selector := orm.RegisterSelector[User](db).
    Select().
    Limit(10).
    Offset(20)
```

### 分页助手函数

实现分页查询的辅助函数：

```go
func paginate[T any](ctx context.Context, db *orm.DB, page, pageSize int, where ...orm.Condition) ([]*T, int64, error) {
    // 创建选择器
    selector := orm.RegisterSelector[T](db)
    
    // 计算总记录数
    countSelector := orm.RegisterSelector[T](db)
    countResult, err := countSelector.
        Select(orm.Count("*").As("count")).
        Where(where...).
        Get(ctx)
    if err != nil {
        return nil, 0, err
    }
    
    // 获取分页数据
    offset := (page - 1) * pageSize
    items, err := selector.
        Select().
        Where(where...).
        OrderBy(orm.Desc(orm.Col("CreatedAt"))).
        Limit(pageSize).
        Offset(offset).
        GetMulti(ctx)
    
    if err != nil {
        return nil, 0, err
    }
    
    // 从计数查询中提取总数
    total := countResult.Count
    
    return items, total, nil
}
```

使用分页函数：

```go
// 获取第2页，每页10条数据
users, total, err := paginate[User](ctx, db, 2, 10, orm.Col("Status").Eq("active"))
```

## 聚合函数

WebFrame ORM 支持各种聚合函数，如 COUNT、SUM、AVG、MAX 和 MIN。

### 基本聚合查询

```go
// COUNT - 计算用户总数
countResult, err := orm.RegisterSelector[User](db).
    Select(orm.Count("*").As("total")).
    Get(ctx)

// SUM - 计算总年龄
sumResult, err := orm.RegisterSelector[User](db).
    Select(orm.Sum("Age").As("total_age")).
    Get(ctx)

// AVG - 计算平均年龄
avgResult, err := orm.RegisterSelector[User](db).
    Select(orm.Avg("Age").As("avg_age")).
    Get(ctx)

// MAX - 查找最大年龄
maxResult, err := orm.RegisterSelector[User](db).
    Select(orm.Max("Age").As("max_age")).
    Get(ctx)

// MIN - 查找最小年龄
minResult, err := orm.RegisterSelector[User](db).
    Select(orm.Min("Age").As("min_age")).
    Get(ctx)
```

### DISTINCT 聚合

使用 DISTINCT 关键字进行聚合：

```go
// COUNT DISTINCT - 计算不重复的部门数量
countDistinctResult, err := orm.RegisterSelector[User](db).
    Select(orm.CountDistinct("DepartmentID").As("dept_count")).
    Get(ctx)
```

### 多聚合查询

在同一查询中使用多个聚合函数：

```go
// 同时获取多个统计数据
statsResult, err := orm.RegisterSelector[User](db).
    Select(
        orm.Count("*").As("total_users"),
        orm.Avg("Age").As("avg_age"),
        orm.Min("Age").As("min_age"),
        orm.Max("Age").As("max_age"),
    ).
    Get(ctx)

// 访问结果
totalUsers := statsResult.TotalUsers
avgAge := statsResult.AvgAge
minAge := statsResult.MinAge
maxAge := statsResult.MaxAge
```

### 分组聚合

使用 `GroupBy` 进行分组聚合查询：

```go
// 按部门统计用户数量
deptStats, err := orm.RegisterSelector[User](db).
    Select(
        orm.Col("DepartmentID"),
        orm.Count("*").As("user_count"),
        orm.Avg("Age").As("avg_age"),
    ).
    GroupBy(orm.Col("DepartmentID")).
    GetMulti(ctx)
```

### 聚合过滤

使用 `Having` 过滤聚合结果：

```go
// 查找用户数量大于5的部门
largeDepts, err := orm.RegisterSelector[User](db).
    Select(
        orm.Col("DepartmentID"),
        orm.Count("*").As("user_count"),
    ).
    GroupBy(orm.Col("DepartmentID")).
    Having(orm.Col("user_count").Gt(5)).
    GetMulti(ctx)
```

### 子查询中的聚合

在子查询中使用聚合函数：

```go
// 查找用户数量高于平均值的部门
avgUserCountQuery := orm.RegisterSelector[User](db).
    Select(orm.Avg(orm.Count("*")).As("avg_count")).
    GroupBy(orm.Col("DepartmentID")).
    AsSubQuery("avg_stats")

depts, err := orm.RegisterSelector[Department](db).
    Select().
    Join(orm.InnerJoin, avgUserCountQuery).
    Where(orm.Col("user_count").Gt(orm.Col("avg_stats.avg_count"))).
    GetMulti(ctx)
```

## 综合示例

下面是一个综合示例，展示了如何结合使用选择器、条件构建、排序分页和聚合函数：

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/fyerfyer/fyer-webframe/orm"
)

// User 用户模型
type User struct {
    ID           int64
    Name         string
    Age          int
    Email        string
    DepartmentID int64
    Status       string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

func main() {
    // 连接数据库
    db, err := orm.OpenDB("mysql", "user:password@tcp(localhost:3306)/testdb", "mysql")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    ctx := context.Background()
    
    // 复杂查询示例：
    // 查找活跃部门中年龄超过平均年龄的用户
    // 按创建时间降序排列，并分页显示
    
    // 1. 首先查询平均年龄
    avgAgeResult, err := orm.RegisterSelector[User](db).
        Select(orm.Avg("Age").As("avg_age")).
        Get(ctx)
    if err != nil {
        panic(err)
    }
    avgAge := avgAgeResult.AvgAge
    
    // 2. 查询活跃部门ID (假设有一个活跃部门的阈值)
    activeDeptQuery := orm.RegisterSelector[User](db).
        Select(
            orm.Col("DepartmentID"),
            orm.Count("*").As("user_count"),
        ).
        GroupBy(orm.Col("DepartmentID")).
        Having(orm.Col("user_count").Gt(5)).
        AsSubQuery("active_depts")
    
    // 3. 主查询：查找活跃部门中年龄超过平均年龄的用户
    page := 1
    pageSize := 10
    offset := (page - 1) * pageSize
    
    users, err := orm.RegisterSelector[User](db).
        Select().
        Where(
            orm.Col("Age").Gt(avgAge),
            orm.Col("Status").Eq("active"),
            orm.Col("DepartmentID").In(activeDeptQuery),
        ).
        OrderBy(orm.Desc(orm.Col("CreatedAt"))).
        Limit(pageSize).
        Offset(offset).
        GetMulti(ctx)
    
    if err != nil {
        panic(err)
    }
    
    // 处理结果
    for _, user := range users {
        fmt.Printf("ID: %d, Name: %s, Age: %d\n", user.ID, user.Name, user.Age)
    }
}
```
