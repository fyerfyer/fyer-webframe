package session

import (
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/fyerfyer/fyer-webframe/web/session"
	"github.com/google/uuid"
)

// Middleware 用于初始化和处理会话的中间件
type Middleware struct {
	SessionManager *session.Manager
	// 是否自动创建会话
	AutoCreate     bool
	// 会话初始化器，用于初始化新会话
	Initializer    SessionInitializer
}

// SessionInitializer 初始化最初的会话值
type SessionInitializer func(s session.Session) error

// Build 中间件构建器
func (m *Middleware) Build() web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			// 获取会话
			sess, err := m.SessionManager.GetSession(ctx)

			// 如果会话不存在且自动创建为真，则创建一个新会话
			if err != nil && m.AutoCreate {
				id := uuid.New().String()
				sess, err = m.SessionManager.InitSession(ctx, id)
				if err == nil && m.Initializer != nil {
					// Initialize the session with defaults
					err = m.Initializer(sess)
				}
			}

			// 执行下一个HandleFunc
			next(ctx)

			// 刷新会话
			if sess != nil {
				m.SessionManager.RefreshSession(ctx)
			}
		}
	}
}

// NewSessionMiddleware creates a new session middleware
func NewSessionMiddleware(manager *session.Manager, autoCreate bool) *Middleware {
	return &Middleware{
		SessionManager: manager,
		AutoCreate:     autoCreate,
	}
}

// WithInitializer adds a session initializer to the middleware
func (m *Middleware) WithInitializer(init SessionInitializer) *Middleware {
	m.Initializer = init
	return m
}