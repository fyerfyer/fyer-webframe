package cookiepropagator

import (
    "net/http"
    "time"
)

// CookiePropagator 使用cookie来实现session传播
type CookiePropagator struct {
    cookieName     string
    cookiePath     string
    cookieDomain   string
    cookieMaxAge   int
    cookieSecure   bool
    cookieHTTPOnly bool
    sameSite       http.SameSite
}

// CookiePropagatorOption 配置CookiePropagator
type CookiePropagatorOption func(*CookiePropagator)

// WithCookieName 设置cookie名称
func WithCookieName(name string) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.cookieName = name
    }
}

// WithCookiePath 设置cookie路径
func WithCookiePath(path string) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.cookiePath = path
    }
}

// WithCookieDomain 设置cookie域
func WithCookieDomain(domain string) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.cookieDomain = domain
    }
}

// WithCookieMaxAge 设置cookie最大存活时间（秒）
func WithCookieMaxAge(maxAge int) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.cookieMaxAge = maxAge
    }
}

// WithCookieSecure 设置cookie安全标志
func WithCookieSecure(secure bool) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.cookieSecure = secure
    }
}

// WithCookieHTTPOnly 设置cookie HTTP only标志
func WithCookieHTTPOnly(httpOnly bool) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.cookieHTTPOnly = httpOnly
    }
}

// WithSameSite 设置cookie SameSite属性
func WithSameSite(sameSite http.SameSite) CookiePropagatorOption {
    return func(p *CookiePropagator) {
        p.sameSite = sameSite
    }
}

// NewCookiePropagator 创建新的CookiePropagator
func NewCookiePropagator(opts ...CookiePropagatorOption) *CookiePropagator {
    p := &CookiePropagator{
        cookieName:     "session_id",
        cookiePath:     "/",
        cookieMaxAge:   3600, // 1 hour
        cookieSecure:   false,
        cookieHTTPOnly: true,
        sameSite:       http.SameSiteLaxMode,
    }

    for _, opt := range opts {
        opt(p)
    }

    return p
}

// Extract 从请求中提取session ID
func (p *CookiePropagator) Extract(req *http.Request) (string, error) {
    cookie, err := req.Cookie(p.cookieName)
    if err != nil {
        return "", err
    }
    return cookie.Value, nil
}

// Insert 在响应中设置带有session ID的cookie
func (p *CookiePropagator) Insert(id string, resp http.ResponseWriter) error {
    cookie := &http.Cookie{
        Name:     p.cookieName,
        Value:    id,
        Path:     p.cookiePath,
        Domain:   p.cookieDomain,
        MaxAge:   p.cookieMaxAge,
        Secure:   p.cookieSecure,
        HttpOnly: p.cookieHTTPOnly,
        SameSite: p.sameSite,
    }
    http.SetCookie(resp, cookie)
    return nil
}

// Remove 删除session cookie
func (p *CookiePropagator) Remove(resp http.ResponseWriter) error {
    cookie := &http.Cookie{
        Name:     p.cookieName,
        Value:    "",
        Path:     p.cookiePath,
        Domain:   p.cookieDomain,
        MaxAge:   -1,
        Secure:   p.cookieSecure,
        HttpOnly: p.cookieHTTPOnly,
        SameSite: p.sameSite,
        Expires:  time.Unix(0, 0),
    }
    http.SetCookie(resp, cookie)
    return nil
}