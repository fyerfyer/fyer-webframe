package web

import (
	"github.com/patrickmn/go-cache"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	smallFileSize  = 10 * 1024       // 10KB
	mediumFileSize = 100 * 1024      // 100KB
	largeFileSize  = 1024 * 1024     // 1MB
	hugeFileSize   = 10 * 1024 * 1024 // 10MB
)

func BenchmarkStaticResource(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "static_resource_benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	testFiles := createTestFiles(b, tempDir)

	contentTypes := map[string]string{
		".txt": "text/plain",
		".html": "text/html",
		".css": "text/css",
		".js": "application/javascript",
	}

	benchmarks := []struct {
		name      string
		setupFunc func() (*StaticResource, *httptest.ResponseRecorder, *Context)
		fileSize  int
	}{
		{
			name: "SmallFile_NoCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(-1, -1) // Disable cache
				return setupBenchmarkRequest(sr, testFiles["small.txt"])
			},
			fileSize: smallFileSize,
		},
		{
			name: "SmallFile_WithCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(5*time.Minute, time.Minute)
				sr.maxSize = smallFileSize * 2
				return setupBenchmarkRequest(sr, testFiles["small.txt"])
			},
			fileSize: smallFileSize,
		},
		{
			name: "MediumFile_NoCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(-1, -1)
				return setupBenchmarkRequest(sr, testFiles["medium.txt"])
			},
			fileSize: mediumFileSize,
		},
		{
			name: "MediumFile_WithCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(5*time.Minute, time.Minute)
				sr.maxSize = mediumFileSize * 2
				return setupBenchmarkRequest(sr, testFiles["medium.txt"])
			},
			fileSize: mediumFileSize,
		},
		{
			name: "LargeFile_NoCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(-1, -1)
				return setupBenchmarkRequest(sr, testFiles["large.txt"])
			},
			fileSize: largeFileSize,
		},
		{
			name: "LargeFile_WithCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(5*time.Minute, time.Minute)
				sr.maxSize = largeFileSize * 2
				return setupBenchmarkRequest(sr, testFiles["large.txt"])
			},
			fileSize: largeFileSize,
		},
		{
			name: "HugeFile_NoCache",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				sr.extContentType[".txt"] = "text/plain"
				sr.cache = cache.New(-1, -1)
				return setupBenchmarkRequest(sr, testFiles["huge.txt"])
			},
			fileSize: hugeFileSize,
		},
		{
			name: "MultipleContentTypes",
			setupFunc: func() (*StaticResource, *httptest.ResponseRecorder, *Context) {
				sr := NewStaticResource(tempDir)
				for ext, cType := range contentTypes {
					sr.extContentType[ext] = cType
				}
				return setupBenchmarkRequest(sr, testFiles["small.html"])
			},
			fileSize: smallFileSize,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(bm.fileSize))

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sr, w, ctx := bm.setupFunc()
				handler := sr.Handle()
				handler(ctx)

				if w.Code != http.StatusOK && i == 0 {
					b.Fatalf("Expected status 200, got %d", w.Code)
				}
			}
		})
	}

	b.Run("CachePerformance", func(b *testing.B) {
		sr := NewStaticResource(tempDir)
		sr.extContentType[".txt"] = "text/plain"
		sr.cache = cache.New(5*time.Minute, time.Minute)
		sr.maxSize = mediumFileSize * 2

		handler := sr.Handle()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, w, ctx := setupBenchmarkRequest(sr, testFiles["medium.txt"])
			handler(ctx)

			if w.Code != http.StatusOK && i == 0 {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

func createTestFiles(b *testing.B, dir string) map[string]string {
	files := map[string]string{
		"small.txt":  filepath.Join(dir, "small.txt"),
		"medium.txt": filepath.Join(dir, "medium.txt"),
		"large.txt":  filepath.Join(dir, "large.txt"),
		"huge.txt":   filepath.Join(dir, "huge.txt"),
		"small.html": filepath.Join(dir, "small.html"),
	}

	createFile(b, files["small.txt"], smallFileSize)
	createFile(b, files["medium.txt"], mediumFileSize)
	createFile(b, files["large.txt"], largeFileSize)
	createFile(b, files["huge.txt"], hugeFileSize)

	htmlContent := []byte("<!DOCTYPE html><html><body><h1>Test</h1></body></html>")
	if err := os.WriteFile(files["small.html"], htmlContent, 0644); err != nil {
		b.Fatal(err)
	}

	return files
}

func createFile(b *testing.B, path string, size int) {
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(i % 256)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		b.Fatal(err)
	}
}

func setupBenchmarkRequest(sr *StaticResource, filePath string) (*StaticResource, *httptest.ResponseRecorder, *Context) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/"+filepath.Base(filePath), nil)

	ctx := &Context{
		Req:  req,
		Resp: w,
		Param: map[string]string{
			"file": filepath.Base(filePath),
		},
	}

	return sr, w, ctx
}