package redissession

import (
	"context"
	"testing"
	"time"

	"github.com/fyerfyer/fyer-webframe/web/session"

	"net/http"
	"net/http/httptest"

	"github.com/fyerfyer/fyer-kit/pool"
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/session/cookiepropagator"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MockRedisConnection implements pool.Connection interface for testing
type MockRedisConnection struct {
	client *redis.Client
	closed bool
}

func (m *MockRedisConnection) Close() error {
	m.closed = true
	return nil
}

func (m *MockRedisConnection) Raw() interface{} {
	return m.client
}

func (m *MockRedisConnection) IsAlive() bool {
	return !m.closed
}

func (m *MockRedisConnection) ResetState() error {
	m.closed = false
	return nil
}

// MockRedisPool implements pool.Pool interface for testing
type MockRedisPool struct {
	client *redis.Client
}

func (p *MockRedisPool) Get(ctx context.Context) (pool.Connection, error) {
	return &MockRedisConnection{client: p.client}, nil
}

func (p *MockRedisPool) Put(conn pool.Connection, err error) error {
	// We don't need to do anything special here for the test
	return nil
}

func (p *MockRedisPool) Shutdown(ctx context.Context) error {
	return nil
}

func (p *MockRedisPool) Stats() pool.Stats {
	return pool.Stats{}
}

func NewMockRedisPool(client *redis.Client) pool.Pool {
	return &MockRedisPool{client: client}
}

type RedisSessionTestSuite struct {
	suite.Suite
	client  *redis.Client
	pool    pool.Pool
	storage *RedisStorage
}

func (s *RedisSessionTestSuite) SetupSuite() {
	s.client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	_, err := s.client.Ping(ctx).Result()
	require.NoError(s.T(), err, "Redis server must be available")

	// Create a mock pool using our Redis client
	s.pool = NewMockRedisPool(s.client)
}

func (s *RedisSessionTestSuite) SetupTest() {
	s.storage = NewRedisStorage(
		s.pool,
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
		s.pool,
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
		s.pool,
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

	// Create a storage with short expiration and cleanup
	gcStorage := NewRedisStorage(
		s.pool,
		WithExpireTime(1*time.Second),
		WithPrefix("test_sess_"),
		WithCleanupInterval(2*time.Second),
	)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		id := uuid.New().String()
		sess, err := gcStorage.Create(ctx, id)
		require.NoError(s.T(), err)
		require.NoError(s.T(), sess.Set(ctx, "key", i))
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Force garbage collection
	require.NoError(s.T(), gcStorage.GC(ctx))

	// Cleanup
	gcStorage.Close()
}

func (s *RedisSessionTestSuite) TestSessionManager() {
	// Create a session manager
	cookieProp := cookiepropagator.NewCookiePropagator()
	manager := session.NewMagager(s.storage, cookieProp, "test_session")

	// Create a test request and response
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := &web.Context{
		Req:        req,
		Resp:       w,
		UserValues: make(map[string]any),
		Context:    req.Context(), // Add this line to initialize Context
	}

	// Initialize a session
	id := uuid.New().String()
	sess, err := manager.InitSession(ctx, id)
	require.NoError(s.T(), err)

	// Set some data
	err = sess.Set(ctx.Context, "test_key", "test_value")
	require.NoError(s.T(), err)

	// Verify cookie was set
	response := w.Result()
	cookies := response.Cookies()
	require.Len(s.T(), cookies, 1)
	assert.Equal(s.T(), "session_id", cookies[0].Name)
	assert.Equal(s.T(), id, cookies[0].Value)

	// Test session retrieval
	req.AddCookie(cookies[0])
	ctx2 := &web.Context{
		Req:        req,
		Resp:       httptest.NewRecorder(),
		UserValues: make(map[string]any),
		Context:    req.Context(), // Add this line to initialize Context
	}

	// Get the session
	sess2, err := manager.GetSession(ctx2)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), id, sess2.ID())

	// Test TouchSession
	require.NoError(s.T(), manager.TouchSession(ctx2))

	// Clean up
	require.NoError(s.T(), manager.DeleteSession(ctx2))
}

func TestRedisSessionSuite(t *testing.T) {
	suite.Run(t, new(RedisSessionTestSuite))
}