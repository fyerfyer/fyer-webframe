package session

import (
	"github.com/fyerfyer/fyer-webframe/web"
)

type Manager struct {
	Storage
	Propagator
	sessionKey string
}

func NewMagager(storage Storage, propagator Propagator, sessionKey string) *Manager {
	return &Manager{
		Storage:    storage,
		Propagator: propagator,
		sessionKey: sessionKey,
	}
}

// InitSession 初始化一个session，并把它注入到context中
func (m *Manager) InitSession(ctx *web.Context, id string) (Session, error) {
	// 确保 UserValues 已初始化
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any)
	}

	sess, err := m.Create(ctx.Context, id)
	if err != nil {
		return nil, err
	}

	if err := m.Insert(id, ctx.Resp); err != nil {
		return nil, err
	}

	// 将 session 存储到 context 中
	ctx.UserValues[m.sessionKey] = sess
	return sess, nil
}

func (m *Manager) GetSession(ctx *web.Context) (Session, error) {
	// 先尝试从 UserValues 中获取
	if sess, ok := ctx.UserValues[m.sessionKey]; ok {
		return sess.(Session), nil
	}

	// 如果不存在，则从请求中提取并存储
	id, err := m.Extract(ctx.Req)
	if err != nil {
		return nil, err
	}

	sess, err := m.Find(ctx.Context, id)
	if err != nil {
		return nil, err
	}

	// 存储到 context 中
	if ctx.UserValues == nil {
		ctx.UserValues = make(map[string]any)
	}
	ctx.UserValues[m.sessionKey] = sess

	return sess, nil
}

func (m *Manager) RefreshSession(ctx *web.Context) error {
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err
	}

	return m.Refresh(ctx.Context, sess.ID())
}

// TouchSession renews the session expiration without changing its data
func (m *Manager) TouchSession(ctx *web.Context) error {
	sess, err := m.GetSession(ctx)
	if err != nil {
		return err
	}

	return sess.Touch(ctx.Context)
}

func (m *Manager) DeleteSession(ctx *web.Context) error {
	id, err := m.Extract(ctx.Req)
	if err != nil {
		return err
	}

	if err := m.Delete(ctx.Context, id); err != nil {
		return err
	}

	return m.Remove(ctx.Resp)
}

// Close cleans up resources used by the session manager
func (m *Manager) Close() error {
	return m.Storage.Close()
}
