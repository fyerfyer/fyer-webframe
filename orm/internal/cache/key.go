package cache

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// KeyGenerator 定义缓存键生成器接口
type KeyGenerator interface {
	// Generate 为查询生成唯一缓存键
	Generate(modelName, operation string, query string, args []interface{}) string

	// GenerateWithTags 生成带标签的缓存键
	GenerateWithTags(modelName, operation string, query string, args []interface{}, tags ...string) string

	// BuildTagKey 为标签生成键
	BuildTagKey(tag string) string
}

// DefaultKeyGenerator 是默认的缓存键生成器实现
type DefaultKeyGenerator struct {
	prefix      string        // 缓存键前缀
	includeArgs bool          // 是否在键中包含参数值
	tagPrefix   string        // 标签键前缀
	maxKeySize  int           // 键的最大长度
}

// NewDefaultKeyGenerator 创建一个新的默认键生成器
func NewDefaultKeyGenerator(prefix string) *DefaultKeyGenerator {
	return &DefaultKeyGenerator{
		prefix:      prefix,
		includeArgs: true,
		tagPrefix:   "tag:",
		maxKeySize:  200, // 默认最大键长度为200字符
	}
}

// WithIncludeArgs 设置是否在缓存键中包含参数值
func (g *DefaultKeyGenerator) WithIncludeArgs(include bool) *DefaultKeyGenerator {
	g.includeArgs = include
	return g
}

// WithTagPrefix 设置标签键前缀
func (g *DefaultKeyGenerator) WithTagPrefix(prefix string) *DefaultKeyGenerator {
	g.tagPrefix = prefix
	return g
}

// WithMaxKeySize 设置键的最大长度
func (g *DefaultKeyGenerator) WithMaxKeySize(size int) *DefaultKeyGenerator {
	g.maxKeySize = size
	return g
}

// Generate 为查询生成唯一缓存键
func (g *DefaultKeyGenerator) Generate(modelName, operation string, query string, args []interface{}) string {
	var key strings.Builder

	// 添加前缀
	key.WriteString(g.prefix)
	if len(g.prefix) > 0 && !strings.HasSuffix(g.prefix, ":") {
		key.WriteString(":")
	}

	// 添加模型名和操作类型
	key.WriteString(modelName)
	key.WriteString(":")
	key.WriteString(operation)
	key.WriteString(":")

	// 添加查询的哈希值
	queryHash := md5.Sum([]byte(query))
	key.WriteString(hex.EncodeToString(queryHash[:]))

	// 如果需要添加参数
	if g.includeArgs && len(args) > 0 {
		key.WriteString(":")
		argsStr := argsToString(args)
		argsHash := md5.Sum([]byte(argsStr))
		key.WriteString(hex.EncodeToString(argsHash[:]))
	}

	// 如果键太长，截断它
	result := key.String()
	if len(result) > g.maxKeySize {
		resultHash := md5.Sum([]byte(result))
		result = result[:g.maxKeySize-32] + hex.EncodeToString(resultHash[:])
	}

	return result
}

// GenerateWithTags 生成带标签的缓存键
func (g *DefaultKeyGenerator) GenerateWithTags(modelName, operation string, query string, args []interface{}, tags ...string) string {
	baseKey := g.Generate(modelName, operation, query, args)

	// 如果没有标签，直接返回基础键
	if len(tags) == 0 {
		return baseKey
	}

	// 排序标签以确保生成一致的键
	sortedTags := make([]string, len(tags))
	copy(sortedTags, tags)
	sort.Strings(sortedTags)

	var tagsStr strings.Builder
	for i, tag := range sortedTags {
		if i > 0 {
			tagsStr.WriteString(",")
		}
		tagsStr.WriteString(tag)
	}

	return baseKey + ":tags:" + tagsStr.String()
}

// BuildTagKey 为标签生成键
func (g *DefaultKeyGenerator) BuildTagKey(tag string) string {
	return g.tagPrefix + tag
}

// argsToString 将查询参数转换为字符串
func argsToString(args []interface{}) string {
	if len(args) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, arg := range args {
		if i > 0 {
			builder.WriteString(",")
		}

		switch v := arg.(type) {
		case nil:
			builder.WriteString("nil")
		case string:
			builder.WriteString(v)
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			builder.WriteString(fmt.Sprintf("%d", v))
		case float32, float64:
			builder.WriteString(strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64))
		case bool:
			builder.WriteString(strconv.FormatBool(v))
		default:
			// 对于复杂类型，使用类型名称和哈希值
			typeName := reflect.TypeOf(v).String()
			builder.WriteString(typeName)
			builder.WriteString(":")
			builder.WriteString(fmt.Sprintf("%p", v))
		}
	}

	return builder.String()
}

// QueryHash 通过组合查询语句和参数生成哈希值
func QueryHash(query string, args []interface{}) string {
	h := md5.New()
	h.Write([]byte(query))

	if len(args) > 0 {
		h.Write([]byte(argsToString(args)))
	}

	return hex.EncodeToString(h.Sum(nil))
}