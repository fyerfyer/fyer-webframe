package web

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/fyerfyer/fyer-kit/pool"
	objPool "github.com/fyerfyer/fyer-webframe/web/pool"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Context 表示HTTP请求和响应的上下文信息
type Context struct {
	Req            *http.Request       // HTTP请求对象
	Resp           http.ResponseWriter // HTTP响应写入器
	Param          map[string]string   // 路由参数映射
	RouteURL       string              // 当前路由的URL
	RespStatusCode int                 // 响应状态码
	RespData       []byte              // 响应数据
	unhandled      bool                // 标记是否已处理请求
	tplEngine      Template            // 模板引擎
	UserValues     map[string]any      // 用户自定义值存储
	Context        context.Context     // 标准上下文对象
	aborted        bool                // 标记是否终止处理
	poolManager    pool.PoolManager    // 连接池管理器 (注意：这不是对象池)
}

// Reset 重置Context对象以便重用
// 实现objPool.Poolable接口
func (c *Context) Reset() {
	// 清空核心字段
	c.Req = nil
	c.Resp = nil
	c.Context = nil
	c.RespStatusCode = 0
	c.RespData = nil
	c.RouteURL = ""
	c.unhandled = true
	c.aborted = false

	// 清空路由参数映射但不重新分配
	for k := range c.Param {
		delete(c.Param, k)
	}

	// 清空用户值但不重新分配
	for k := range c.UserValues {
		delete(c.UserValues, k)
	}

	// 保留模板引擎和连接池管理器引用，这些不需要重置
}

// SetRequest 设置请求对象，用于对象池重用时
func (c *Context) SetRequest(req *http.Request) {
	c.Req = req
	if req != nil {
		c.Context = req.Context()
	}
}

// SetResponse 设置响应写入器，用于对象池重用时
func (c *Context) SetResponse(resp http.ResponseWriter) {
	c.Resp = resp
}

// newContextForPool 创建一个新的Context，用于对象池
func newContextForPool(opts objPool.CtxOptions) interface{} {
	paramCap := 8 // 默认参数容量
	if opts.ParamCapacity > 0 {
		paramCap = opts.ParamCapacity
	}

	ctx := &Context{
		Param:      make(map[string]string, paramCap),
		UserValues: make(map[string]any, paramCap),
		unhandled:  true,
	}

	// 只在tplEngine非空时进行类型断言
	if opts.TplEngine != nil {
		ctx.tplEngine = opts.TplEngine.(Template)
	}

	if opts.PoolManager != nil {
		ctx.poolManager = opts.PoolManager.(pool.PoolManager)
	}

	return ctx
}

// InitContextPool 初始化Context对象池
func InitContextPool(tplEngine Template, connPoolManager pool.PoolManager, paramCap int) {
	opts := objPool.CtxOptions{
		TplEngine:     tplEngine,
		PoolManager:   connPoolManager,
		ParamCapacity: paramCap,
	}

	objPool.InitDefaultContextPool(newContextForPool, opts)
}

// AcquireContext 从池中获取一个Context对象
func AcquireContext(req *http.Request, resp http.ResponseWriter) *Context {
	ctx := objPool.AcquireContext(req, resp).(*Context)
	return ctx
}

// ReleaseContext 将Context对象返回到池中
func ReleaseContext(ctx *Context) {
	if ctx != nil {
		objPool.ReleaseContext(ctx)
	}
}

// Abort 终止当前请求的处理流程
func (c *Context) Abort() {
	c.aborted = true
}

// IsAborted 判断当前请求是否已被终止
func (c *Context) IsAborted() bool {
	return c.aborted
}

// Next 继续执行下一个处理函数，如果请求未被终止
func (c *Context) Next(next HandlerFunc) {
	if !c.aborted {
		next(c)
	}
}

// 请求绑定相关方法

// BindJSON 将请求体绑定到JSON结构体
func (c *Context) BindJSON(v any) error {
	if c.Req.Body == nil {
		return errors.New("request body is empty")
	}
	return json.NewDecoder(c.Req.Body).Decode(v)
}

// BindXML 将请求体绑定到XML结构体
func (c *Context) BindXML(v any) error {
	if c.Req.Body == nil {
		return errors.New("request body is empty")
	}
	return xml.NewDecoder(c.Req.Body).Decode(v)
}

// ReadBody 读取请求体的字节内容
func (c *Context) ReadBody() ([]byte, error) {
	if c.Req.Body == nil {
		return nil, errors.New("request body is empty")
	}
	defer c.Req.Body.Close()
	return io.ReadAll(c.Req.Body)
}

// 值类型定义

// StringValue 表示带有可选错误的字符串值
type StringValue struct {
	Value string
	Error error
}

// IntValue 表示带有可选错误的整数值
type IntValue struct {
	Value int
	Error error
}

// Int64Value 表示带有可选错误的64位整数值
type Int64Value struct {
	Value int64
	Error error
}

// FloatValue 表示带有可选错误的浮点数值
type FloatValue struct {
	Value float64
	Error error
}

// BoolValue 表示带有可选错误的布尔值
type BoolValue struct {
	Value bool
	Error error
}

// 表单相关方法

// FormValue 获取指定键的表单值
func (c *Context) FormValue(key string) StringValue {
	err := c.Req.ParseForm()
	if err != nil {
		return StringValue{Error: fmt.Errorf("failed to parse form value: %w", err)}
	}
	val := c.Req.FormValue(key)
	if val == "" {
		return StringValue{Error: errors.New("key not found")}
	}
	return StringValue{Value: val}
}

// FormInt 获取表单中的整数值
func (c *Context) FormInt(key string) IntValue {
	sv := c.FormValue(key)
	if sv.Error != nil {
		return IntValue{Error: sv.Error}
	}
	val, err := strconv.Atoi(sv.Value)
	if err != nil {
		return IntValue{Error: fmt.Errorf("invalid int value: %w", err)}
	}
	return IntValue{Value: val}
}

// FormInt64 获取表单中的64位整数值
func (c *Context) FormInt64(key string) Int64Value {
	sv := c.FormValue(key)
	if sv.Error != nil {
		return Int64Value{Error: sv.Error}
	}
	val, err := strconv.ParseInt(sv.Value, 10, 64)
	if err != nil {
		return Int64Value{Error: fmt.Errorf("invalid int value: %w", err)}
	}
	return Int64Value{Value: val}
}

// FormFloat 获取表单中的浮点数值
func (c *Context) FormFloat(key string) FloatValue {
	sv := c.FormValue(key)
	if sv.Error != nil {
		return FloatValue{Error: sv.Error}
	}
	val, err := strconv.ParseFloat(sv.Value, 64)
	if err != nil {
		return FloatValue{Error: fmt.Errorf("invalid float value: %w", err)}
	}
	return FloatValue{Value: val}
}

// FormBool 获取表单中的布尔值
func (c *Context) FormBool(key string) BoolValue {
	sv := c.FormValue(key)
	if sv.Error != nil {
		return BoolValue{Error: sv.Error}
	}
	val, err := strconv.ParseBool(sv.Value)
	if err != nil {
		return BoolValue{Error: fmt.Errorf("invalid bool value: %w", err)}
	}
	return BoolValue{Value: val}
}

// FormAll 获取所有表单值
func (c *Context) FormAll() (url.Values, error) {
	err := c.Req.ParseForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse form value: %w", err)
	}
	return c.Req.Form, nil
}

// 文件处理相关方法

// FormFile 获取上传的单个文件
func (c *Context) FormFile(key string) (*multipart.FileHeader, error) {
	err := c.Req.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart form file: %w", err)
	}
	file, header, err := c.Req.FormFile(key)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return header, nil
}

// FormFiles 获取上传的多个文件
func (c *Context) FormFiles(key string) ([]*multipart.FileHeader, error) {
	err := c.Req.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart form file: %w", err)
	}
	form := c.Req.MultipartForm
	if form == nil || form.File == nil {
		return nil, errors.New("file not found")
	}
	files := form.File[key]
	if len(files) == 0 {
		return nil, fmt.Errorf("cannot found file with key: %s", key)
	}
	return files, nil
}

// 查询参数相关方法

// QueryParam 获取查询参数值
func (c *Context) QueryParam(key string) StringValue {
	val := c.Req.URL.Query().Get(key)
	if val == "" {
		return StringValue{Error: errors.New("key not found")}
	}
	return StringValue{Value: val}
}

// QueryInt 获取整数类型的查询参数
func (c *Context) QueryInt(key string) IntValue {
	sv := c.QueryParam(key)
	if sv.Error != nil {
		return IntValue{Error: sv.Error}
	}
	val, err := strconv.Atoi(sv.Value)
	if err != nil {
		return IntValue{Error: fmt.Errorf("invalid int value: %w", err)}
	}
	return IntValue{Value: val}
}

// QueryInt64 获取64位整数类型的查询参数
func (c *Context) QueryInt64(key string) Int64Value {
	sv := c.QueryParam(key)
	if sv.Error != nil {
		return Int64Value{Error: sv.Error}
	}
	val, err := strconv.ParseInt(sv.Value, 10, 64)
	if err != nil {
		return Int64Value{Error: fmt.Errorf("invalid int value: %w", err)}
	}
	return Int64Value{Value: val}
}

// QueryFloat 获取浮点数类型的查询参数
func (c *Context) QueryFloat(key string) FloatValue {
	sv := c.QueryParam(key)
	if sv.Error != nil {
		return FloatValue{Error: sv.Error}
	}
	val, err := strconv.ParseFloat(sv.Value, 64)
	if err != nil {
		return FloatValue{Error: fmt.Errorf("invalid float value: %w", err)}
	}
	return FloatValue{Value: val}
}

// QueryBool 获取布尔类型的查询参数
func (c *Context) QueryBool(key string) BoolValue {
	sv := c.QueryParam(key)
	if sv.Error != nil {
		return BoolValue{Error: sv.Error}
	}
	val, err := strconv.ParseBool(sv.Value)
	if err != nil {
		return BoolValue{Error: fmt.Errorf("invalid bool value: %w", err)}
	}
	return BoolValue{Value: val}
}

// QueryAll 获取所有查询参数
func (c *Context) QueryAll() url.Values {
	return c.Req.URL.Query()
}

// 路径参数相关方法

// PathParam 获取路径参数值
func (c *Context) PathParam(key string) StringValue {
	val, ok := c.Param[key]
	if !ok {
		return StringValue{Error: errors.New("key not found")}
	}
	return StringValue{Value: val}
}

// PathInt 获取整数类型的路径参数
func (c *Context) PathInt(key string) IntValue {
	sv := c.PathParam(key)
	if sv.Error != nil {
		return IntValue{Error: sv.Error}
	}
	val, err := strconv.Atoi(sv.Value)
	if err != nil {
		return IntValue{Error: fmt.Errorf("invalid int value: %w", err)}
	}
	return IntValue{Value: val}
}

// PathInt64 获取64位整数类型的路径参数
func (c *Context) PathInt64(key string) Int64Value {
	sv := c.PathParam(key)
	if sv.Error != nil {
		return Int64Value{Error: sv.Error}
	}
	val, err := strconv.ParseInt(sv.Value, 10, 64)
	if err != nil {
		return Int64Value{Error: fmt.Errorf("invalid int value: %w", err)}
	}
	return Int64Value{Value: val}
}

// PathFloat 获取浮点数类型的路径参数
func (c *Context) PathFloat(key string) FloatValue {
	sv := c.PathParam(key)
	if sv.Error != nil {
		return FloatValue{Error: sv.Error}
	}
	val, err := strconv.ParseFloat(sv.Value, 64)
	if err != nil {
		return FloatValue{Error: fmt.Errorf("invalid float value: %w", err)}
	}
	return FloatValue{Value: val}
}

// PathBool 获取布尔类型的路径参数
func (c *Context) PathBool(key string) BoolValue {
	sv := c.PathParam(key)
	if sv.Error != nil {
		return BoolValue{Error: sv.Error}
	}
	val, err := strconv.ParseBool(sv.Value)
	if err != nil {
		return BoolValue{Error: fmt.Errorf("invalid float value: %w", err)}
	}
	return BoolValue{Value: val}
}

// HTTP头部处理

// GetHeader 获取请求头的值
func (c *Context) GetHeader(key string) string {
	return c.Req.Header.Get(key)
}

// GetHeaders 获取请求头的所有值
func (c *Context) GetHeaders(key string) []string {
	return c.Req.Header.Values(key)
}

// SetHeader 设置响应头
func (c *Context) SetHeader(key, value string) *Context {
	c.Resp.Header().Set(key, value)
	return c
}

// AddHeader 添加响应头
func (c *Context) AddHeader(key, value string) *Context {
	c.Resp.Header().Add(key, value)
	return c
}

// Status 设置HTTP状态码
func (c *Context) Status(code int) *Context {
	c.RespStatusCode = code
	return c
}

// Cookie处理

// GetCookie 根据名称获取Cookie
func (c *Context) GetCookie(name string) (*http.Cookie, error) {
	return c.Req.Cookie(name)
}

// SetCookie 设置Cookie
func (c *Context) SetCookie(cookie *http.Cookie) *Context {
	http.SetCookie(c.Resp, cookie)
	return c
}

// 内容类型处理

// ContentType 获取Content-Type头部
func (c *Context) ContentType() string {
	return c.GetHeader("Content-Type")
}

// IsContentType 检查Content-Type是否包含指定类型
func (c *Context) IsContentType(kind string) bool {
	ct := c.ContentType()
	if ct == "" {
		return false
	}
	return strings.Contains(ct, kind)
}

// IsJSON 检查请求Content-Type是否为JSON
func (c *Context) IsJSON() bool {
	return c.IsContentType(ContentTypeJSON)
}

// IsXML 检查请求Content-Type是否为XML
func (c *Context) IsXML() bool {
	return c.IsContentType(ContentTypeXML)
}

// IsForm 检查请求Content-Type是否为表单数据
func (c *Context) IsForm() bool {
	return c.IsContentType(ContentTypeForm)
}

// IsMultipartForm 检查请求Content-Type是否为多部分表单数据
func (c *Context) IsMultipartForm() bool {
	return c.IsContentType(ContentTypeMultipartForm)
}

// 客户端信息

// ClientIP 获取客户端IP地址
func (c *Context) ClientIP() string {
	// 检查X-Forwarded-For和X-Real-IP头部（用于代理）
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For头部可能包含多个IP
		ips := strings.Split(ip, ",")
		// 返回第一个非空地址
		for _, ipAddress := range ips {
			ipAddress = strings.TrimSpace(ipAddress)
			if ipAddress != "" {
				return ipAddress
			}
		}
	}

	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	// 否则使用RemoteAddr
	ip, _, _ := strings.Cut(c.Req.RemoteAddr, ":")
	return ip
}

// UserAgent 获取User-Agent头部信息
func (c *Context) UserAgent() string {
	return c.GetHeader("User-Agent")
}

// Referer 获取Referer头部信息
func (c *Context) Referer() string {
	return c.GetHeader("Referer")
}

// Pool 从连接池管理器中获取指定名称的连接池
func (c *Context) Pool(name string) (pool.Pool, error) {
	if c.poolManager == nil {
		return nil, errors.New("pool manager not initialized")
	}
	return c.poolManager.Get(name)
}

// SetPoolManager 设置连接池管理器
func (c *Context) SetPoolManager(manager pool.PoolManager) {
	c.poolManager = manager
}

// GetConnection 从指定池中获取连接
func (c *Context) GetConnection(poolName string) (pool.Connection, error) {
	p, err := c.Pool(poolName)
	if err != nil {
		return nil, err
	}
	return p.Get(c.Context)
}
