package web

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestGoTemplate(t *testing.T) {
	// 创建临时测试目录
	tmpDir := t.TempDir()

	// 创建测试模板文件
	tplContent := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <h1>{{.Message}}</h1>
</body>
</html>
`
	tplPath := filepath.Join(tmpDir, "test.html")
	err := os.WriteFile(tplPath, []byte(tplContent), 0666)
	require.NoError(t, err)

	tests := []struct {
		name    string
		tpl     *GoTemplate
		wantErr bool
	}{
		{
			name: "load from files",
			tpl: NewGoTemplate(WithFiles(tplPath)),
			wantErr: false,
		},
		{
			name: "load from glob",
			tpl: NewGoTemplate(WithPattern(filepath.Join(tmpDir, "*.html"))),
			wantErr: false,
		},
	}

	// 测试基本功能
	t.Run("basic functionality", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data := map[string]string{
					"Title":   "Test Page",
					"Message": "Hello, World!",
				}

				ctx := &Context{}
				result, err := tt.tpl.Render(ctx, "test.html", data)

				if tt.wantErr {
					assert.Error(t, err)
					return
				}

				assert.NoError(t, err)
				assert.Contains(t, string(result), "Test Page")
				assert.Contains(t, string(result), "Hello, World!")
			})
		}
	})

	// 测试边界情况
	t.Run("edge cases", func(t *testing.T) {
		t.Run("invalid template name", func(t *testing.T) {
			tpl := NewGoTemplate(WithFiles(tplPath))
			_, err := tpl.Render(&Context{}, "non_existent.html", nil)
			assert.Error(t, err)
		})

		t.Run("concurrent template reload", func(t *testing.T) {
			tpl := NewGoTemplate(WithFiles(tplPath))
			done := make(chan bool)

			// 并发重载和渲染
			go func() {
				err := tpl.Reload()
				assert.NoError(t, err)
				done <- true
			}()

			go func() {
				_, err := tpl.Render(&Context{}, "test.html", map[string]string{
					"Title":   "Test",
					"Message": "Test",
				})
				assert.NoError(t, err)
				done <- true
			}()

			<-done
			<-done
		})
	})
}

// TestTemplateWithHTTPServer 测试HTTP服务器与模板的集成
func TestTemplateWithHTTPServer(t *testing.T) {
	// 创建临时目录和模板
	tempDir := t.TempDir()

	// 创建视图模板
	viewDir := filepath.Join(tempDir, "views")
	err := os.MkdirAll(viewDir, 0755)
	require.NoError(t, err)

	// 创建模板文件
	layoutPath := filepath.Join(viewDir, "layout.html")
	homePath := filepath.Join(viewDir, "home.html")

	layoutContent := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <header>{{.ProjectName}}</header>
    {{template "content" .}}
    <footer>{{.CurrentYear}}</footer>
</body>
</html>`

	homeContent := `{{define "content"}}
<div class="content">
    <h1>{{.Title}}</h1>
    <p>{{.Message}}</p>
</div>
{{end}}`

	err = os.WriteFile(layoutPath, []byte(layoutContent), 0644)
	require.NoError(t, err)

	err = os.WriteFile(homePath, []byte(homeContent), 0644)
	require.NoError(t, err)

	// 创建HTTP服务器
	tpl := NewGoTemplate(WithPattern(filepath.Join(viewDir, "*.html")))
	s := NewHTTPServer(WithTemplate(tpl))

	// 注册路由
	s.Get("/", func(ctx *Context) {
		data := map[string]interface{}{
			"Title":       "首页",
			"Message":     "欢迎访问",
			"ProjectName": "测试项目",
			"CurrentYear": "2025",
		}

		err := ctx.Template("layout.html", data)
		assert.NoError(t, err)
	})

	// 测试请求
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	s.ServeHTTP(w, r)

	// 验证响应
	assert.Equal(t, http.StatusOK, w.Code)
	html := w.Body.String()
	assert.Contains(t, html, "<title>首页</title>")
	assert.Contains(t, html, "<h1>首页</h1>")
	assert.Contains(t, html, "<p>欢迎访问</p>")
	assert.Contains(t, html, "<header>测试项目</header>")
	assert.Contains(t, html, "<footer>2025</footer>")
}