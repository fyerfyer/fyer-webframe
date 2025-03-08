package sharding

import (
	"errors"
	"fmt"
	"hash/crc32"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidShardKey   = errors.New("invalid shard key value")
	ErrInvalidShardRange = errors.New("invalid shard range value")
	ErrInvalidDateFormat = errors.New("invalid date format")
)

// Strategy 分片策略接口，所有具体分片策略都实现此接口
type Strategy interface {
	// Route 计算给定键值应该路由到哪个分片
	Route(key interface{}) (dbIndex, tableIndex int, err error)

	// GetShardName 获取分片的数据库和表名
	GetShardName(dbIndex, tableIndex int) (dbName, tableName string, err error)
}

// BaseStrategy 提供分片策略的基本功能
type BaseStrategy struct {
	// 数据库配置
	DBCount    int    // 数据库数量
	DBPrefix   string // 数据库名称前缀

	// 表配置
	TableCount  int    // 每个数据库中的表数量
	TablePrefix string // 表名前缀

	// 分片键配置
	ShardKey   string // 用于分片的键名
	ModelName  string // 模型名称
}

// NewBaseStrategy 创建基础分片策略
func NewBaseStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string) *BaseStrategy {
	return &BaseStrategy{
		DBPrefix:    dbPrefix,
		DBCount:     dbCount,
		TablePrefix: tablePrefix,
		TableCount:  tableCount,
		ShardKey:    shardKey,
	}
}

// GetShardName 获取分片的数据库和表名
func (s *BaseStrategy) GetShardName(dbIndex, tableIndex int) (string, string, error) {
	if dbIndex < 0 || dbIndex >= s.DBCount {
		return "", "", fmt.Errorf("db index out of range: %d", dbIndex)
	}

	if tableIndex < 0 || tableIndex >= s.TableCount {
		return "", "", fmt.Errorf("table index out of range: %d", tableIndex)
	}

	dbName := fmt.Sprintf("%s%d", s.DBPrefix, dbIndex)
	tableName := fmt.Sprintf("%s%d", s.TablePrefix, tableIndex)

	return dbName, tableName, nil
}

// HashStrategy 基于哈希的分片策略
type HashStrategy struct {
	*BaseStrategy
}

// NewHashStrategy 创建基于哈希的分片策略
func NewHashStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string) *HashStrategy {
	return &HashStrategy{
		BaseStrategy: NewBaseStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey),
	}
}

// Route 基于哈希的路由算法
func (s *HashStrategy) Route(key interface{}) (int, int, error) {
	if key == nil {
		return 0, 0, ErrInvalidShardKey
	}

	// 将key转换为字符串
	var strKey string

	switch v := key.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		strKey = fmt.Sprintf("%d", v)
	case string:
		strKey = v
	case []byte:
		strKey = string(v)
	default:
		return 0, 0, fmt.Errorf("unsupported key type: %T", key)
	}

	// 计算哈希值
	hashCode := crc32.ChecksumIEEE([]byte(strKey))

	// 计算分片索引
	dbIndex := int(hashCode % uint32(s.DBCount))
	tableIndex := int(hashCode / uint32(s.DBCount) % uint32(s.TableCount))

	return dbIndex, tableIndex, nil
}

// RangeStrategy 基于范围的分片策略
type RangeStrategy struct {
	*BaseStrategy
	Ranges []int64 // 范围分界值
}

// NewRangeStrategy 创建基于范围的分片策略
func NewRangeStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string, ranges []int64) *RangeStrategy {
	if len(ranges) == 0 {
		panic("ranges cannot be empty for range strategy")
	}

	return &RangeStrategy{
		BaseStrategy: NewBaseStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey),
		Ranges:       ranges,
	}
}

// Route 基于范围的路由算法
func (s *RangeStrategy) Route(key interface{}) (int, int, error) {
	if key == nil {
		return 0, 0, ErrInvalidShardKey
	}

	// 将key转换为int64
	var intKey int64

	switch v := key.(type) {
	case int:
		intKey = int64(v)
	case int8:
		intKey = int64(v)
	case int16:
		intKey = int64(v)
	case int32:
		intKey = int64(v)
	case int64:
		intKey = v
	case uint:
		intKey = int64(v)
	case uint8:
		intKey = int64(v)
	case uint16:
		intKey = int64(v)
	case uint32:
		intKey = int64(v)
	case uint64:
		intKey = int64(v)
	case string:
		var err error
		intKey, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, 0, ErrInvalidShardKey
		}
	default:
		return 0, 0, fmt.Errorf("unsupported key type: %T", key)
	}

	// 查找所在范围
	idx := 0
	for i, r := range s.Ranges {
		if intKey < r {
			idx = i
			break
		}
		idx = i + 1
	}

	// 计算分片索引
	totalShards := s.DBCount * s.TableCount
	shardIndex := idx % totalShards

	dbIndex := shardIndex % s.DBCount
	tableIndex := shardIndex / s.DBCount

	return dbIndex, tableIndex, nil
}

// DateStrategy 基于日期的分片策略
type DateStrategy struct {
	*BaseStrategy
	DateFormat  string // 日期格式，支持"daily", "weekly", "monthly", "yearly"或自定义格式
	StartTime   time.Time
}

// NewDateStrategy 创建基于日期的分片策略
func NewDateStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string, dateFormat string) *DateStrategy {
	return &DateStrategy{
		BaseStrategy: NewBaseStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey),
		DateFormat:   dateFormat,
		StartTime:    time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), // 默认起始日期
	}
}

// WithStartTime 设置起始时间
func (s *DateStrategy) WithStartTime(startTime time.Time) *DateStrategy {
	s.StartTime = startTime
	return s
}

// Route 基于日期的路由算法
func (s *DateStrategy) Route(key interface{}) (int, int, error) {
	if key == nil {
		return 0, 0, ErrInvalidShardKey
	}

	// 将key转换为time.Time
	var dateKey time.Time
	var err error

	switch v := key.(type) {
	case time.Time:
		dateKey = v
	case string:
		// 尝试多种常见格式解析
		formats := []string{
			"2006-01-02 15:04:05",
			"2006-01-02",
			time.RFC3339,
		}
		parsed := false
		for _, format := range formats {
			dateKey, err = time.Parse(format, v)
			if err == nil {
				parsed = true
				break
			}
		}
		if !parsed {
			return 0, 0, ErrInvalidDateFormat
		}
	case []byte:
		return s.Route(string(v))
	case int64:
		// 假设是Unix时间戳
		dateKey = time.Unix(v, 0)
	default:
		return 0, 0, fmt.Errorf("unsupported key type: %T", key)
	}

	// 计算时间差
	var period int
	switch strings.ToLower(s.DateFormat) {
	case "daily":
		period = int(dateKey.Sub(s.StartTime).Hours() / 24)
	case "weekly":
		period = int(dateKey.Sub(s.StartTime).Hours() / 24 / 7)
	case "monthly":
		yearDiff := dateKey.Year() - s.StartTime.Year()
		monthDiff := int(dateKey.Month()) - int(s.StartTime.Month())
		period = yearDiff*12 + monthDiff
	case "yearly":
		period = dateKey.Year() - s.StartTime.Year()
	default:
		// 使用自定义格式，基于日期字符串进行哈希
		dateStr := dateKey.Format(s.DateFormat)
		hashCode := crc32.ChecksumIEEE([]byte(dateStr))
		totalShards := s.DBCount * s.TableCount
		shardIndex := int(hashCode % uint32(totalShards))

		return shardIndex % s.DBCount, shardIndex / s.DBCount, nil
	}

	// 对于负周期(早于起始日期)，使用mod的绝对值
	if period < 0 {
		period = -period
	}

	// 计算分片索引
	totalShards := s.DBCount * s.TableCount
	shardIndex := period % totalShards

	dbIndex := shardIndex % s.DBCount
	tableIndex := shardIndex / s.DBCount

	return dbIndex, tableIndex, nil
}

// GetShardName 重写获取分片名称，支持按日期格式化
func (s *DateStrategy) GetShardName(dbIndex, tableIndex int) (string, string, error) {
	if dbIndex < 0 || dbIndex >= s.DBCount {
		return "", "", fmt.Errorf("db index out of range: %d", dbIndex)
	}

	if tableIndex < 0 || tableIndex >= s.TableCount {
		return "", "", fmt.Errorf("table index out of range: %d", tableIndex)
	}

	dbName := fmt.Sprintf("%s%d", s.DBPrefix, dbIndex)

	// 如果表前缀包含日期格式化占位符，则进行格式化
	tableName := s.TablePrefix
	if strings.Contains(tableName, "%") {
		now := time.Now()
		tableName = now.Format(tableName)
	}

	tableName = fmt.Sprintf("%s%d", tableName, tableIndex)

	return dbName, tableName, nil
}

// ModStrategy 取模分片策略 (简单但有效的分片方法)
type ModStrategy struct {
	*BaseStrategy
}

// NewModStrategy 创建基于取模的分片策略
func NewModStrategy(dbPrefix string, dbCount int, tablePrefix string, tableCount int, shardKey string) *ModStrategy {
	return &ModStrategy{
		BaseStrategy: NewBaseStrategy(dbPrefix, dbCount, tablePrefix, tableCount, shardKey),
	}
}

// Route 基于取模的路由算法
func (s *ModStrategy) Route(key interface{}) (int, int, error) {
	if key == nil {
		return 0, 0, ErrInvalidShardKey
	}

	// 将key转换为int64
	var intKey int64

	switch v := key.(type) {
	case int:
		intKey = int64(v)
	case int8:
		intKey = int64(v)
	case int16:
		intKey = int64(v)
	case int32:
		intKey = int64(v)
	case int64:
		intKey = v
	case uint:
		intKey = int64(v)
	case uint8:
		intKey = int64(v)
	case uint16:
		intKey = int64(v)
	case uint32:
		intKey = int64(v)
	case uint64:
		intKey = int64(v)
	case string:
		var err error
		intKey, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			// 对于非数字字符串，使用哈希值
			hashCode := crc32.ChecksumIEEE([]byte(v))
			intKey = int64(hashCode)
		}
	default:
		// 对于其他类型，尝试通过反射获取数值
		rv := reflect.ValueOf(key)
		if rv.Kind() >= reflect.Int && rv.Kind() <= reflect.Int64 {
			intKey = rv.Int()
		} else if rv.Kind() >= reflect.Uint && rv.Kind() <= reflect.Uint64 {
			intKey = int64(rv.Uint())
		} else {
			return 0, 0, fmt.Errorf("unsupported key type: %T", key)
		}
	}

	// 确保是非负数
	if intKey < 0 {
		intKey = -intKey
	}

	// 计算分片索引
	dbIndex := int(intKey % int64(s.DBCount))
	tableIndex := int((intKey / int64(s.DBCount)) % int64(s.TableCount))

	return dbIndex, tableIndex, nil
}