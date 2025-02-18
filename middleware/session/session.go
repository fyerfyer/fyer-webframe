package session

import (
	"github.com/fyerfyer/fyer-webframe/session"
	"github.com/fyerfyer/fyer-webframe/web"
	"github.com/google/uuid"
)

// SessionMiddleware 创建一个处理 session 的中间件,自动化处理 HTTP 请求的会话管理。
func SessionMiddleware(m *session.Manager) web.Middleware {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(ctx *web.Context) {
			if ctx.UserValues == nil {
				ctx.UserValues = make(map[string]any)
			}

			// 尝试获取现有 session
			_, err := m.GetSession(ctx)
			if err != nil {
				// session 不存在，创建新的
				id := uuid.New().String()
				_, err = m.InitSession(ctx, id)
				if err != nil {
					ctx.RespStatusCode = 500
					ctx.RespData = []byte("failed to init session")
					return
				}
			}

			// 继续处理请求
			next(ctx)

			// 请求处理完成后刷新 session
			_ = m.RefreshSession(ctx)
		}
	}
}
