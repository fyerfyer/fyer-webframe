package web

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fyerfyer/fyer-kit/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	t.Run("bind JSON", func(t *testing.T) {
		bodyReader := strings.NewReader(`{"name": "test"}`)
		req, err := http.NewRequest(http.MethodPost, "/test", bodyReader)
		req.Header.Set("Content-Type", "application/json")
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		type User struct {
			Name string `json:"name"`
		}
		var user User

		err = ctx.BindJSON(&user)
		assert.NoError(t, err)
		assert.Equal(t, "test", user.Name)
	})

	t.Run("bind XML", func(t *testing.T) {
		bodyReader := strings.NewReader(`<User><Name>test</Name></User>`)
		req, err := http.NewRequest(http.MethodPost, "/test", bodyReader)
		req.Header.Set("Content-Type", "application/xml")
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		type User struct {
			Name string `xml:"Name"`
		}
		var user User

		err = ctx.BindXML(&user)
		assert.NoError(t, err)
		assert.Equal(t, "test", user.Name)
	})

	t.Run("form values", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("name", "test")
		formData.Set("age", "25")
		formData.Set("active", "true")
		formData.Set("height", "1.85")

		req, err := http.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		ctx := &Context{
			Req: req,
		}

		nameVal := ctx.FormValue("name")
		assert.Equal(t, "test", nameVal.Value)
		assert.Nil(t, nameVal.Error)

		ageVal := ctx.FormInt("age")
		assert.Equal(t, 25, ageVal.Value)
		assert.Nil(t, ageVal.Error)

		activeVal := ctx.FormBool("active")
		assert.Equal(t, true, activeVal.Value)
		assert.Nil(t, activeVal.Error)

		heightVal := ctx.FormFloat("height")
		assert.Equal(t, 1.85, heightVal.Value)
		assert.Nil(t, heightVal.Error)

		missingVal := ctx.FormValue("notexist")
		assert.NotNil(t, missingVal.Error)

		invalidInt := ctx.FormInt("name")
		assert.NotNil(t, invalidInt.Error)
	})

	t.Run("query parameters", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test?name=test&age=25&active=true&height=1.85", nil)
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		nameVal := ctx.QueryParam("name")
		assert.Equal(t, "test", nameVal.Value)
		assert.Nil(t, nameVal.Error)

		ageVal := ctx.QueryInt("age")
		assert.Equal(t, 25, ageVal.Value)
		assert.Nil(t, ageVal.Error)

		activeVal := ctx.QueryBool("active")
		assert.Equal(t, true, activeVal.Value)
		assert.Nil(t, activeVal.Error)

		heightVal := ctx.QueryFloat("height")
		assert.Equal(t, 1.85, heightVal.Value)
		assert.Nil(t, heightVal.Error)

		missingVal := ctx.QueryParam("notexist")
		assert.NotNil(t, missingVal.Error)

		invalidInt := ctx.QueryInt("name")
		assert.NotNil(t, invalidInt.Error)

		allQuery := ctx.QueryAll()
		assert.Equal(t, 4, len(allQuery))
		assert.Equal(t, "test", allQuery.Get("name"))
	})

	t.Run("path parameters", func(t *testing.T) {
		ctx := &Context{
			Param: map[string]string{
				"id":     "123",
				"name":   "test",
				"active": "true",
				"height": "1.85",
			},
		}

		idVal := ctx.PathParam("id")
		assert.Equal(t, "123", idVal.Value)
		assert.Nil(t, idVal.Error)

		idIntVal := ctx.PathInt("id")
		assert.Equal(t, 123, idIntVal.Value)
		assert.Nil(t, idIntVal.Error)

		activeVal := ctx.PathBool("active")
		assert.Equal(t, true, activeVal.Value)
		assert.Nil(t, activeVal.Error)

		heightVal := ctx.PathFloat("height")
		assert.InDelta(t, 1.85, heightVal.Value, 0.001)
		assert.Nil(t, heightVal.Error)

		missingVal := ctx.PathParam("notexist")
		assert.NotNil(t, missingVal.Error)

		invalidInt := ctx.PathInt("name")
		assert.NotNil(t, invalidInt.Error)
	})

	t.Run("headers", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("X-Custom", "custom-value")
		req.Header.Add("X-Multi", "value1")
		req.Header.Add("X-Multi", "value2")

		ctx := &Context{
			Req:  req,
			Resp: httptest.NewRecorder(),
		}

		assert.Equal(t, "test-agent", ctx.GetHeader("User-Agent"))
		assert.Equal(t, "custom-value", ctx.GetHeader("X-Custom"))

		multiValues := ctx.GetHeaders("X-Multi")
		assert.Equal(t, 2, len(multiValues))
		assert.Equal(t, "value1", multiValues[0])
		assert.Equal(t, "value2", multiValues[1])

		ctx.SetHeader("Content-Type", "application/json").
			AddHeader("X-Response", "test")

		respHeader := ctx.Resp.Header()
		assert.Equal(t, "application/json", respHeader.Get("Content-Type"))
		assert.Equal(t, "test", respHeader.Get("X-Response"))
	})

	t.Run("content type helpers", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		ctx := &Context{
			Req: req,
		}

		assert.True(t, ctx.IsJSON())
		assert.False(t, ctx.IsXML())
		assert.Equal(t, "application/json; charset=utf-8", ctx.ContentType())
	})

	t.Run("client information", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("Referer", "http://example.com")
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		req.RemoteAddr = "192.168.1.1:12345"

		ctx := &Context{
			Req: req,
		}

		assert.Equal(t, "203.0.113.195", ctx.ClientIP())

		assert.Equal(t, "test-agent", ctx.UserAgent())
		assert.Equal(t, "http://example.com", ctx.Referer())

		req.Header.Del("X-Forwarded-For")
		assert.Equal(t, "192.168.1.1", ctx.ClientIP())
	})
}

func TestContextWithValues(t *testing.T) {
	ctx := &Context{
		Context: context.WithValue(context.Background(), "key", "value"),
	}

	assert.Equal(t, "value", ctx.Context.Value("key"))
}

func TestReadBody(t *testing.T) {
	bodyContent := []byte("test body content")
	req, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyContent))
	require.NoError(t, err)

	ctx := &Context{
		Req: req,
	}

	body, err := ctx.ReadBody()
	assert.NoError(t, err)
	assert.Equal(t, bodyContent, body)
}

func TestFileUploads(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("file", "test.txt")
	require.NoError(t, err)

	_, err = fw.Write([]byte("test file content"))
	require.NoError(t, err)

	err = w.WriteField("name", "test")
	require.NoError(t, err)

	w.Close()

	req, err := http.NewRequest(http.MethodPost, "/upload", &b)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())

	ctx := &Context{
		Req: req,
	}

	_, err = ctx.FormFile("file")
	assert.NoError(t, err)

	files, err := ctx.FormFiles("file")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "test.txt", files[0].Filename)

	nameVal := ctx.FormValue("name")
	assert.Equal(t, "test", nameVal.Value)
	assert.Nil(t, nameVal.Error)
}

func TestCookies(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "test-cookie",
		Value: "cookie-value",
	})

	w := httptest.NewRecorder()
	ctx := &Context{
		Req:  req,
		Resp: w,
	}

	cookie, err := ctx.GetCookie("test-cookie")
	assert.NoError(t, err)
	assert.Equal(t, "test-cookie", cookie.Name)
	assert.Equal(t, "cookie-value", cookie.Value)

	ctx.SetCookie(&http.Cookie{
		Name:  "response-cookie",
		Value: "response-value",
	})

	cookies := w.Result().Cookies()
	assert.Equal(t, 1, len(cookies))
	assert.Equal(t, "response-cookie", cookies[0].Name)
	assert.Equal(t, "response-value", cookies[0].Value)
}

// MockPoolManager 实现 pool.PoolManager 接口用于测试
// MockPoolManager 实现 pool.PoolManager 接口用于测试
type MockPoolManager struct {
	pools         map[string]pool.Pool
	failGetPool   bool
	failGetConn   bool
	connLifecycle []string
}

func NewMockPoolManager() *MockPoolManager {
	return &MockPoolManager{
		pools:         make(map[string]pool.Pool),
		connLifecycle: make([]string, 0),
	}
}

func (m *MockPoolManager) Get(name string) (pool.Pool, error) {
	if m.failGetPool {
		return nil, fmt.Errorf("failed to get pool")
	}
	p, ok := m.pools[name]
	if !ok {
		return nil, fmt.Errorf("pool not found: %s", name)
	}
	return p, nil
}

func (m *MockPoolManager) Register(name string, p pool.Pool) error {
	m.pools[name] = p
	return nil
}

func (m *MockPoolManager) Remove(name string) error {
	delete(m.pools, name)
	return nil
}

func (m *MockPoolManager) Shutdown(ctx context.Context) error {
	for _, p := range m.pools {
		if err := p.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockPoolManager) Stats() map[string]pool.Stats {
	stats := make(map[string]pool.Stats)
	for name, p := range m.pools {
		stats[name] = p.Stats()
	}
	return stats
}

// MockPool 实现 pool.Pool 接口用于测试
type MockPool struct {
	manager       *MockPoolManager
	connections   []pool.Connection
	failGetConn   bool
	failPutConn   bool
	failShutdown  bool
	currentConnID int
}

func NewMockPool(manager *MockPoolManager) *MockPool {
	return &MockPool{
		manager:     manager,
		connections: make([]pool.Connection, 0),
	}
}

func (p *MockPool) Get(ctx context.Context) (pool.Connection, error) {
	if p.failGetConn {
		return nil, fmt.Errorf("failed to get connection")
	}
	p.manager.connLifecycle = append(p.manager.connLifecycle, "get connection")
	p.currentConnID++
	conn := &MockConnection{
		id:      p.currentConnID,
		pool:    p,
		closed:  false,
		manager: p.manager,
	}
	p.connections = append(p.connections, conn)
	return conn, nil
}

func (p *MockPool) Put(conn pool.Connection, err error) error {
	if p.failPutConn {
		return fmt.Errorf("failed to return connection")
	}
	p.manager.connLifecycle = append(p.manager.connLifecycle, "return connection")
	return nil
}

func (p *MockPool) Shutdown(ctx context.Context) error {
	if p.failShutdown {
		return fmt.Errorf("failed to close pool")
	}
	p.manager.connLifecycle = append(p.manager.connLifecycle, "shutdown pool")
	return nil
}

func (p *MockPool) Stats() pool.Stats {
	return pool.Stats{
		Active: len(p.connections),
	}
}

// MockConnection 实现 pool.Connection 接口用于测试
type MockConnection struct {
	id      int
	pool    *MockPool
	closed  bool
	manager *MockPoolManager
}

func (c *MockConnection) Close() error {
	c.closed = true
	c.manager.connLifecycle = append(c.manager.connLifecycle, "close connection")
	return nil
}

func (c *MockConnection) Raw() interface{} {
	c.manager.connLifecycle = append(c.manager.connLifecycle, "access connection")
	return c.id
}

func (c *MockConnection) IsAlive() bool {
	return !c.closed
}

func (c *MockConnection) ResetState() error {
	return nil
}

func TestContextPoolAccess(t *testing.T) {
	t.Run("test getting conn", func(t *testing.T) {
		// 创建mock连接池管理器
		mockManager := NewMockPoolManager()
		mockPool := NewMockPool(mockManager)

		// 注册连接池
		err := mockManager.Register("testdb", mockPool)
		require.NoError(t, err)

		// 创建context并设置连接池管理器
		ctx := &Context{
			Context:     context.Background(),
			poolManager: mockManager,
		}

		// 从连接池获取连接
		conn, err := ctx.GetConnection("testdb")
		require.NoError(t, err)
		require.NotNil(t, conn)

		// 检查连接的ID
		connID := conn.Raw().(int)
		assert.Equal(t, 1, connID)

		// 关闭连接
		err = conn.Close()
		require.NoError(t, err)
	})

	t.Run("test conn lifecycle", func(t *testing.T) {
		mockManager := NewMockPoolManager()
		mockPool := NewMockPool(mockManager)

		// 注册连接池
		err := mockManager.Register("db", mockPool)
		require.NoError(t, err)

		ctx := &Context{
			Context:     context.Background(),
			poolManager: mockManager,
		}

		// 1. 获取连接
		conn, err := ctx.GetConnection("db")
		require.NoError(t, err)

		// 2. 使用连接
		id := conn.Raw()
		assert.NotNil(t, id)

		// 3. 关闭连接
		err = conn.Close()
		require.NoError(t, err)

		// 验证生命周期事件
		expectedLifecycle := []string{"get connection", "access connection", "close connection"}
		assert.Equal(t, expectedLifecycle, mockManager.connLifecycle)

		// 验证连接状态
		assert.False(t, conn.IsAlive())
	})

	t.Run("test pool error handling", func(t *testing.T) {
		mockManager := NewMockPoolManager()
		mockPool := NewMockPool(mockManager)

		// 设置获取池失败标志
		mockManager.failGetPool = true

		ctx := &Context{
			Context:     context.Background(),
			poolManager: mockManager,
		}

		// 测试获取不存在的连接池
		conn, err := ctx.GetConnection("non-existent-pool")
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "failed to get pool")

		// 重置标志并注册池
		mockManager.failGetPool = false
		err = mockManager.Register("faildb", mockPool)
		require.NoError(t, err)

		// 设置获取连接失败标志
		mockPool.failGetConn = true

		// 测试获取连接失败
		conn, err = ctx.GetConnection("faildb")
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "failed to get connection")

		// 测试连接池管理器为nil的情况
		ctx.poolManager = nil
		conn, err = ctx.GetConnection("any-pool")
		assert.Error(t, err)
		assert.Nil(t, conn)
		assert.Contains(t, err.Error(), "pool manager not initialized")
	})

	t.Run("test pool access helper methods", func(t *testing.T) {
		mockManager := NewMockPoolManager()
		mockPool := NewMockPool(mockManager)

		err := mockManager.Register("maindb", mockPool)
		require.NoError(t, err)

		ctx := &Context{
			Context: context.Background(),
		}

		// 测试 SetPoolManager
		ctx.SetPoolManager(mockManager)
		assert.Equal(t, mockManager, ctx.poolManager)

		// 测试 Pool 方法
		pool, err := ctx.Pool("maindb")
		require.NoError(t, err)
		assert.Equal(t, mockPool, pool)

		// 测试获取不存在的池
		pool, err = ctx.Pool("non-existent-pool")
		assert.Error(t, err)
		assert.Nil(t, pool)
	})
}