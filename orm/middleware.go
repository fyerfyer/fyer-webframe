package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Handler 处理器接口定义
type Handler interface {
	QueryHandler(ctx context.Context, qc *QueryContext) (*QueryResult, error)
}

// Middleware 中间件定义
type Middleware func(Handler) Handler

// QueryContext 查询上下文定义
type QueryContext struct {
	QueryType  string
	Query      *Query
	Model      *model
	Builder    QueryBuilder
	TableName  string      // 表名，支持分片时可能会被替换
	ShardKey   string      // 用于分片的键
	ShardValue interface{} // 分片键的值
}

// QueryResult 查询结果定义
type QueryResult struct {
	Result        Result
	Rows          *sql.Rows
	Err           error
	CachedData    []map[string]interface{} // 缓存的数据
	CachedColumns []string                 // 缓存的列名
}

// BuildChain 构建处理器调用链
func BuildChain(core Handler, ms []Middleware) Handler {
	h := core
	// 从后往前构建,保证最先添加的中间件最先执行
	for i := len(ms) - 1; i >= 0; i-- {
		h = ms[i](h)
	}
	return h
}

// CoreHandler 核心处理器
// CoreHandler 是整个中间件链的最后一环，它负责实际执行数据库操作。
type CoreHandler struct {
	db *DB
}

func (c *CoreHandler) QueryHandler(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
	switch qc.QueryType {
	case "query":
		rows, err := c.db.queryContext(ctx, qc.Query.SQL, qc.Query.Args...)
		return &QueryResult{
			Rows: rows,
			Err:  err,
		}, err
	case "exec":
		res, err := c.db.execContext(ctx, qc.Query.SQL, qc.Query.Args...)
		return &QueryResult{
			Result: Result{
				res: res,
				err: err,
			},
			Err: err,
		}, err
	default:
		return nil, fmt.Errorf("unknown query type: %s", qc.QueryType)
	}
}

// HandlerFunc 用于将函数转换为 Handler 接口
type HandlerFunc func(ctx context.Context, qc *QueryContext) (*QueryResult, error)

func (h HandlerFunc) QueryHandler(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
	return h(ctx, qc)
}

// ShardingMiddleware 创建分片中间件
func ShardingMiddleware(manager *ShardingManager) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
			// 如果分片未启用，直接传给下一个处理器
			if manager == nil || !manager.IsEnabled() {
				return next.QueryHandler(ctx, qc)
			}

			// 获取模型名称作为表名
			modelName := ""
			if qc.Model != nil {
				modelName = qc.Model.GetTableName()
			} else if qc.TableName != "" {
				modelName = qc.TableName
			} else {
				// 无法确定表名，直接传给下一个处理器
				return next.QueryHandler(ctx, qc)
			}

			// 尝试从查询上下文中获取分片键值
			values := make(map[string]interface{})

			// 如果查询上下文中已有明确的分片键和值
			if qc.ShardKey != "" && qc.ShardValue != nil {
				values[qc.ShardKey] = qc.ShardValue
			} else {
				// 尝试从条件中提取分片键
				var err error
				values, err = extractShardKeyFromQuery(qc, manager)
				if err != nil {
					// 提取失败但不是致命错误，可以继续使用默认数据库
					if !errors.Is(err, ErrNoShardKeyFound) {
						// 记录日志或处理其他类型错误
					}
				}
			}

			// 如果没有找到分片键，使用默认路由
			if len(values) == 0 {
				return next.QueryHandler(ctx, qc)
			}

			// 计算路由
			shardDB, tableName, err := manager.Route(ctx, modelName, values)
			if err != nil {
				// 路由失败，使用默认数据库
				return next.QueryHandler(ctx, qc)
			}

			// 如果需要替换表名
			if tableName != "" && tableName != modelName {
				// 替换SQL中的表名
				// 这里的实现比较简化，实际可能需要更复杂的SQL解析和替换
				qc.Query.SQL = replaceTableName(qc.Query.SQL, modelName, tableName)
				qc.TableName = tableName
			}

			// 创建一个新的查询上下文，使用分片DB处理
			shardHandler := &CoreHandler{db: shardDB}
			return shardHandler.QueryHandler(ctx, qc)
		})
	}
}

// 提取查询中的分片键值
func extractShardKeyFromQuery(qc *QueryContext, manager *ShardingManager) (map[string]interface{}, error) {
	if qc.Query == nil || qc.Model == nil {
		return nil, ErrNoShardKeyFound
	}

	// 获取模型的分片信息
	modelName := qc.Model.GetTableName()
	info, ok := manager.GetModelInfo(modelName)
	if !ok {
		return nil, ErrModelNotRegistered
	}

	// 获取分片键
	shardKey := info.strategy.GetShardKey()
	if shardKey == "" {
		return nil, ErrNoShardKeyFound
	}

	// 如果查询上下文中已经有分片键值，直接返回
	if qc.ShardKey == shardKey && qc.ShardValue != nil {
		values := make(map[string]interface{})
		values[shardKey] = qc.ShardValue
		return values, nil
	}

	// 根据查询类型提取分片键值
	switch builder := qc.Builder.(type) {
	case *Selector[any]:
		// 尝试从Where条件中查找分片键
		if len(builder.args) > 0 {
			// 简单实现：检查SQL中是否包含分片键列名
			colName := ""
			if field, ok := qc.Model.fieldsMap[shardKey]; ok {
				colName = field.colName
			} else {
				colName = shardKey // 默认使用键名作为列名
			}

			// 简化处理：在SQL中查找分片键列名
			if strings.Contains(qc.Query.SQL, colName) {
				// 尝试在参数中找到对应值
				for _, arg := range qc.Query.Args {
					// 这里只是简单示例，实际需要更精确的匹配
					values := make(map[string]interface{})
					values[shardKey] = arg
					return values, nil
				}
			}
		}
	case *Inserter[any]:
		// 尝试从插入数据中提取分片键
		if len(qc.Query.Args) > 0 {
			// 简化处理：假设第一个参数可能是分片键值
			values := make(map[string]interface{})
			values[shardKey] = qc.Query.Args[0]
			return values, nil
		}
	}

	// 无法提取
	return nil, ErrNoShardKeyFound
}

// replaceTableName 替换SQL语句中的表名
func replaceTableName(sql string, oldName string, newName string) string {
	// 一个简单的实现，仅供示例
	// 实际项目中应该使用SQL解析器来更准确地替换表名
	return sql
}
