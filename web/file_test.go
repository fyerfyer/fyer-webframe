package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
)

func TestFileUploader(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func() (*http.Request, error)
		fileOptions []FileOption
		wantStatus  int
		wantErr     bool
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
			name: "file exceeds size limit",
			setup: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "large.txt")
				if err != nil {
					return nil, err
				}

				// Create content larger than size limit (50 bytes)
				largeContent := bytes.Repeat([]byte("a"), 100)
				_, err = part.Write(largeContent)
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
			fileOptions: []FileOption{
				WithFileMaxSize(50),
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "file type not allowed",
			setup: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile("file", "test.exe")
				if err != nil {
					return nil, err
				}
				_, err = part.Write([]byte("executable content"))
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
			fileOptions: []FileOption{
				WithAllowedTypes([]string{"text/plain", "image/jpeg"}),
			},
			wantStatus: http.StatusBadRequest,
			wantErr:    true,
		},
		{
			name: "allowed file type",
			setup: func() (*http.Request, error) {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)

				// 创建一个新的 form file part
				h := make(textproto.MIMEHeader)
				h.Set("Content-Type", "text/plain")
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", "test.txt"))
				part, err := writer.CreatePart(h)
				if err != nil {
					return nil, err
				}

				_, err = part.Write([]byte("text content"))
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
			fileOptions: []FileOption{
				WithAllowedTypes([]string{"text/plain", "image/jpeg"}),
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

			uploader := NewFileUploader("file", tmpDir, tt.fileOptions...)

			uploader.HandleUpload()(ctx)
			assert.Equal(t, tt.wantStatus, recorder.Code)

			if tt.wantErr {
				var resp map[string]string
				err := json.NewDecoder(recorder.Body).Decode(&resp)
				require.NoError(t, err)
				assert.Contains(t, resp, "error")

				switch tt.name {
				case "file exceeds size limit":
					assert.Contains(t, resp["error"], "file size exceeds")
				case "file type not allowed":
					assert.Contains(t, resp["error"], "not allowed")
				}
			}
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