package web

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFileUploader(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		setup      func() (*http.Request, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name: "successful upload",
			setup: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "test.txt")
				if err != nil {
					return nil, err
				}
				_, err = part.Write([]byte("test content"))
				if err != nil {
					return nil, err
				}
				err = writer.Close()
				if err != nil {
					return nil, err
				}

				req := httptest.NewRequest(http.MethodPost, "/upload", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				return req, nil
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "no file provided",
			setup: func() (*http.Request, error) {
				return httptest.NewRequest(http.MethodPost, "/upload", nil), nil
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := tt.setup()
			require.NoError(t, err)

			recorder := httptest.NewRecorder()
			ctx := &Context{
				Req:  req,
				Resp: recorder,
			}

			uploader := FileUploder{
				FileField: "file",
				DestPath:  tmpDir,
			}

			uploader.HandleUpload()(ctx)
			assert.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}

func TestFileDownloader(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	testContent := []byte("test content")
	testFilePath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFilePath, testContent, 0666)
	require.NoError(t, err)

	tests := []struct {
		name       string
		filePath   string
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "successful download",
			filePath:   "test.txt",
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "file not found",
			filePath:   "nonexistent.txt",
			wantStatus: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "invalid path",
			filePath:   "../test.txt",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name:       "empty filename",
			filePath:   "",
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/download/"+tt.filePath, nil)
			recorder := httptest.NewRecorder()

			ctx := &Context{
				Req:   req,
				Resp:  recorder,
				Param: map[string]string{"file": tt.filePath},
			}

			downloader := FileDownloader{
				DestPath: tmpDir,
			}

			downloader.HandleDownload()(ctx)
			assert.Equal(t, tt.wantStatus, recorder.Code)

			if !tt.wantErr {
				resp := recorder.Result()
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, testContent, body)
				assert.Contains(t, resp.Header.Get("Content-Disposition"), "attachment")
			}
		})
	}
}