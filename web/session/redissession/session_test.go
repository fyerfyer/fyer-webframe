package redissession

import (
	"context"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-webframe/web/session"

	"net/http"
	"net/http/httptest"

	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/session/cookiepropagator"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RedisSessionTestSuite struct {
	suite.Suite
	client  redis.Cmdable
	storage *RedisStorage
}

func (s *RedisSessionTestSuite) SetupSuite() {
	s.client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	_, err := s.client.Ping(ctx).Result()
	require.NoError(s.T(), err, "Redis server must be available")
}

func (s *RedisSessionTestSuite) SetupTest() {
	s.storage = NewRedisStorage(
		s.client,
		WithExpireTime(5*time.Second),
		WithPrefix("test_sess_"),
		WithCleanupInterval(1*time.Second),
	)
}

func (s *RedisSessionTestSuite) TearDownTest() {
	ctx := context.Background()
	iter := s.client.Scan(ctx, 0, "test_sess_*", 100).Iterator()
	for iter.Next(ctx) {
		s.client.Del(ctx, iter.Val())
	}
	require.NoError(s.T(), iter.Err())

	require.NoError(s.T(), s.storage.Close())
}

func (s *RedisSessionTestSuite) TestSessionCreate() {
	ctx := context.Background()
	id := uuid.New().String()

	sess, err := s.storage.Create(ctx, id)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), sess)
	assert.Equal(s.T(), id, sess.ID())

	exists, err := s.client.Exists(ctx, "test_sess_"+id).Result()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), exists)
}

func (s *RedisSessionTestSuite) TestSessionSetGet() {
	ctx := context.Background()
	id := uuid.New().String()

	sess, err := s.storage.Create(ctx, id)
	require.NoError(s.T(), err)
	require.NoError(s.T(), sess.Set(ctx, "test_key", "test_value"))

	// Get value
	val, err := sess.Get(ctx, "test_key")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "test_value", val)
}

func (s *RedisSessionTestSuite) TestSessionFind() {
	ctx := context.Background()
	id := uuid.New().String()

	// Create session
	sess1, err := s.storage.Create(ctx, id)
	require.NoError(s.T(), err)
	require.NoError(s.T(), sess1.Set(ctx, "test_key", "test_value"))

	// Find session
	sess2, err := s.storage.Find(ctx, id)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), sess2)

	// Verify data
	val, err := sess2.Get(ctx, "test_key")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "test_value", val)
}

func (s *RedisSessionTestSuite) TestSessionExpiration() {
	ctx := context.Background()
	id := uuid.New().String()

	// Create session with shorter expiration
	shortStorage := NewRedisStorage(
		s.client,
		WithExpireTime(1*time.Second),
		WithPrefix("test_sess_"),
	)
	defer shortStorage.Close()

	sess, err := shortStorage.Create(ctx, id)
	require.NoError(s.T(), err)
	require.NoError(s.T(), sess.Set(ctx, "test_key", "test_value"))

	// Verify key exists in Redis
	exists, err := s.client.Exists(ctx, "test_sess_"+id).Result()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1), exists)

	// Wait for expiration - give a bit more time to ensure Redis clears it
	time.Sleep(3 * time.Second)

	// Verify key doesn't exist in Redis anymore
	exists, err = s.client.Exists(ctx, "test_sess_"+id).Result()
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(0), exists, "Session key should have been removed from Redis")

	// Now try to find the session - should fail with redis.Nil
	_, err = shortStorage.Find(ctx, id)
	assert.Equal(s.T(), redis.Nil, err, "Session should have expired")
}

func (s *RedisSessionTestSuite) TestSessionTouch() {
	ctx := context.Background()
	id := uuid.New().String()

	// Create session with short expiration
	shortStorage := NewRedisStorage(
		s.client,
		WithExpireTime(2*time.Second),
		WithPrefix("test_sess_"),
	)
	defer shortStorage.Close()

	sess, err := shortStorage.Create(ctx, id)
	require.NoError(s.T(), err)
	require.NoError(s.T(), sess.Set(ctx, "test_key", "test_value"))

	// Wait partial expiration time
	time.Sleep(1 * time.Second)

	// Touch to renew
	require.NoError(s.T(), sess.Touch(ctx))

	// Wait another partial time - session should still be alive
	time.Sleep(1 * time.Second)

	// Session should still be available
	foundSess, err := shortStorage.Find(ctx, id)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), foundSess)

	val, err := foundSess.Get(ctx, "test_key")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "test_value", val)
}

func (s *RedisSessionTestSuite) TestGarbageCollection() {
	ctx := context.Background()

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		id := uuid.New().String()
		sess, err := s.storage.Create(ctx, id)
		require.NoError(s.T(), err)
		require.NoError(s.T(), sess.Set(ctx, "test_key", i))
	}

	// Create sessions that should expire
	expiredIds := make([]string, 3)
	shortStorage := NewRedisStorage(
		s.client,
		WithExpireTime(1*time.Second),
		WithPrefix("test_sess_"),
	)
	defer shortStorage.Close()

	for i := 0; i < 3; i++ {
		id := uuid.New().String()
		expiredIds[i] = id
		sess, err := shortStorage.Create(ctx, id)
		require.NoError(s.T(), err)
		require.NoError(s.T(), sess.Set(ctx, "test_key", i))
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Force garbage collection
	require.NoError(s.T(), s.storage.GC(ctx))

	// Verify expired sessions are removed from local cache
	for _, id := range expiredIds {
		// Directly check cache
		_, ok := s.storage.sessions.Load(id)
		assert.False(s.T(), ok, "Expired session should be removed from cache")
	}
}

func (s *RedisSessionTestSuite) TestSessionManager() {
	// Create a session manager
	propagator := cookiepropagator.NewCookiePropagator()
	manager := session.NewMagager(s.storage, propagator, "sess")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := &web.Context{
		Req:        req,  // 添加请求
		Resp:       resp, // 添加响应
		Context:    context.Background(),
		UserValues: make(map[string]any), // 初始化 UserValues
	}

	// Initialize a session
	id := uuid.New().String()
	sess, err := manager.InitSession(ctx, id)
	require.NoError(s.T(), err)

	// Set some data
	require.NoError(s.T(), sess.Set(ctx.Context, "test_key", "test_value"))

	// Verify cookie was set
	cookies := resp.Result().Cookies()
	assert.GreaterOrEqual(s.T(), len(cookies), 1)
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session_id" {
			sessionCookie = cookie
			break
		}
	}
	require.NotNil(s.T(), sessionCookie)
	assert.Equal(s.T(), id, sessionCookie.Value)

	// Test session retrieval
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(sessionCookie)
	resp = httptest.NewRecorder()
	ctx = &web.Context{ // 创建新的上下文
		Req:        req,  // 使用新的请求
		Resp:       resp, // 使用新的响应
		Context:    context.Background(),
		UserValues: make(map[string]any), // 初始化 UserValues
	}

	// Get the session
	foundSess, err := manager.GetSession(ctx)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), id, foundSess.ID())

	// Test TouchSession
	require.NoError(s.T(), manager.TouchSession(ctx))

	// Clean up
	require.NoError(s.T(), manager.Close())
}

func TestRedisSessionSuite(t *testing.T) {
	suite.Run(t, new(RedisSessionTestSuite))
}
