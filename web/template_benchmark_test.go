package web

import (
	"html/template"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkTemplateRender(b *testing.B) {
	tmpDir := b.TempDir()

	layoutContent := `
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <header>{{.Header}}</header>
    {{template "content" .}}
    <footer>{{.Footer}}</footer>
</body>
</html>`

	// Create a simple content template
	contentContent := `{{define "content"}}
<div class="content">
    <h1>{{.Title}}</h1>
    <p>{{.Message}}</p>
</div>
{{end}}`

	// Complex template with loops and conditionals
	complexContent := `{{define "complex"}}
<div class="container">
    <h1>{{.Title}}</h1>
    <ul>
        {{range .Items}}
            <li>
                {{if .Important}}
                    <strong>{{.Name}}</strong>
                {{else}}
                    {{.Name}}
                {{end}}
            </li>
        {{end}}
    </ul>
</div>
{{end}}`

	layoutPath := filepath.Join(tmpDir, "layout.html")
	contentPath := filepath.Join(tmpDir, "content.html")
	complexPath := filepath.Join(tmpDir, "complex.html")

	err := os.WriteFile(layoutPath, []byte(layoutContent), 0644)
	if err != nil {
		b.Fatal(err)
	}
	err = os.WriteFile(contentPath, []byte(contentContent), 0644)
	if err != nil {
		b.Fatal(err)
	}
	err = os.WriteFile(complexPath, []byte(complexContent), 0644)
	if err != nil {
		b.Fatal(err)
	}

	ctx := &Context{}

	simpleData := map[string]interface{}{
		"Title":   "Benchmark Test",
		"Header":  "Header Content",
		"Footer":  "Footer Content",
		"Message": "This is a benchmark test.",
	}

	complexData := map[string]interface{}{
		"Title": "Complex Benchmark",
		"Items": []map[string]interface{}{
			{"Name": "Item 1", "Important": true},
			{"Name": "Item 2", "Important": false},
			{"Name": "Item 3", "Important": true},
			{"Name": "Item 4", "Important": false},
			{"Name": "Item 5", "Important": true},
		},
	}

	// Create function map
	funcMap := template.FuncMap{
		"upper": func(s string) string {
			return s
		},
		"formatDate": func(t string) string {
			return t
		},
	}

	b.Run("LoadFromFiles", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tpl := NewGoTemplate(WithFuncMap(funcMap))
			err := tpl.LoadFromFiles(layoutPath, contentPath, complexPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LoadFromGlob", func(b *testing.B) {
		pattern := filepath.Join(tmpDir, "*.html")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tpl := NewGoTemplate(WithFuncMap(funcMap))
			err := tpl.LoadFromGlob(pattern)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RenderSimpleTemplate", func(b *testing.B) {
		tpl := NewGoTemplate(WithFuncMap(funcMap))
		err := tpl.LoadFromFiles(layoutPath, contentPath)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tpl.Render(ctx, "layout.html", simpleData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RenderComplexTemplate", func(b *testing.B) {
		tpl := NewGoTemplate(WithFuncMap(funcMap))
		err := tpl.LoadFromFiles(complexPath)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tpl.Render(ctx, "complex.html", complexData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RenderWithReload", func(b *testing.B) {
		tpl := NewGoTemplate(WithFuncMap(funcMap), WithAutoReload(true))
		err := tpl.LoadFromFiles(layoutPath, contentPath)
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tpl.Render(ctx, "layout.html", simpleData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("IntegrationWithServer", func(b *testing.B) {
		tpl := NewGoTemplate(WithFuncMap(funcMap))
		err := tpl.LoadFromFiles(layoutPath, contentPath)
		if err != nil {
			b.Fatal(err)
		}

		server := NewHTTPServer(WithTemplate(tpl))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := &Context{
				tplEngine: server.GetTemplateEngine(),
				Resp: httptest.NewRecorder(),
			}
			err := ctx.Template("layout.html", simpleData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}