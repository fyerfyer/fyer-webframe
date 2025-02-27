package web

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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