package orm

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// CacheMiddleware 创建缓存中间件
func CacheMiddleware(cacheManager *CacheManager) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, qc *QueryContext) (*QueryResult, error) {
			// 如果缓存管理器未启用或查询类型不是查询操作，直接传递到下一个中间件
			if cacheManager == nil || !cacheManager.IsEnabled() || qc.QueryType != "query" {
				debugLog("Cache middleware: cache disabled or not a query operation")
				return next.QueryHandler(ctx, qc)
			}

			// 检查是否应该缓存此查询
			if !cacheManager.ShouldCache(ctx, qc) {
				debugLog("Cache middleware: should not cache this query")
				return next.QueryHandler(ctx, qc)
			}

			debugLog("Cache middleware: checking if we can cache this query")

			// 生成缓存键
			cacheKey := cacheManager.GenerateKey(qc)
			if cacheKey == "" {
				// 如果无法生成有效的缓存键，直接执行查询而不缓存
				debugLog("Cache middleware: cannot generate cache key")
				return next.QueryHandler(ctx, qc)
			}

			debugLog("Cache middleware: checking cache for key %s", cacheKey)

			// 尝试从缓存中获取结果
			var cachedResult QueryResult
			err := cacheManager.cache.Get(ctx, cacheKey, &cachedResult)
			if err == nil {
				// 缓存命中，直接返回缓存的结果
				debugLog("Cache middleware: cache hit")
				return &cachedResult, nil
			}

			if !errors.Is(err, ErrCacheMiss) {
				// 如果是其他错误而非缓存未命中，记录错误但继续执行查询
				debugLog("Cache middleware: cache error: %v", err)
			} else {
				debugLog("Cache middleware: cache miss for key %s", cacheKey)
			}

			// 缓存未命中，执行查询
			debugLog("Cache middleware: executing query")
			result, err := next.QueryHandler(ctx, qc)
			if err != nil {
				debugLog("Cache middleware: query error: %v", err)
				return nil, err
			}

			if result == nil {
				debugLog("Cache middleware: query result is nil")
				return result, err
			}

			// 查询成功，缓存结果前需要将结果数据读取到内存
			debugLog("Cache middleware: processing rows for caching")
			if result.Rows != nil {
				// 获取TTL
				var ttl time.Duration
				if qc.Model != nil {
					ttl = cacheManager.GetTTL(qc.Model.GetTableName())
					debugLog("Cache middleware: using model TTL: %v", ttl)
				} else {
					ttl = cacheManager.defaultTTL
					debugLog("Cache middleware: using default TTL: %v", ttl)
				}

				// 缓存结果
				debugLog("Cache middleware: setting cache with key %s, TTL %v", cacheKey, ttl)
				_ = cacheManager.cache.Set(ctx, cacheKey, *result, ttl)
			} else {
				debugLog("Cache middleware: no rows to cache")
			}

			return result, err
		})
	}
}

// readRowsData 从sql.Rows读取全部数据
// 返回行数据、列名和可能的错误
func readRowsData(rows *sql.Rows) ([]map[string]interface{}, []string, error) {
	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	// 准备列值缓冲区
	values := make([]interface{}, len(columns))
	scanArgs := make([]interface{}, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// 读取所有行
	var result []map[string]interface{}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, nil, err
		}

		// 创建行数据映射
		rowData := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			// 处理nil值
			if val == nil {
				rowData[col] = nil
				continue
			}

			// 尝试处理常见类型
			switch v := val.(type) {
			case []byte:
				// 对于[]byte类型，尝试转换为字符串
				rowData[col] = string(v)
			default:
				// 其他类型直接存储
				rowData[col] = v
			}
		}

		result = append(result, rowData)
	}

	// 检查遍历过程中是否有错误
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	return result, columns, nil
}

// WithDBMiddlewareCache 设置缓存实现
func WithDBMiddlewareCache(cache Cache) DBOption {
	return func(db *DB) error {
		if db.cacheManager == nil {
			db.cacheManager = NewCacheManager(cache)
		} else {
			db.cacheManager.cache = cache
		}
		db.cacheManager.Enable()

		// 同时注册缓存中间件
		db.Use(CacheMiddleware(db.cacheManager))
		return nil
	}
}