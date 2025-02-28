package web

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContext(t *testing.T) {
	t.Run("test bind json", func(t *testing.T) {
		bodyReader := strings.NewReader(`{"name": "test"}`)
		req, err := http.NewRequest(http.MethodPost, "/test", bodyReader)
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		type User struct {
			Name string `json:"name"`
		}
		var user User

		err = ctx.BindJSON(&user)
		assert.NoError(t, err)
		assert.Equal(t, "test", user.Name)
	})

	t.Run("test form value", func(t *testing.T) {
		// 创建一个带查询参数的请求
		req, err := http.NewRequest(http.MethodPost, "/test?test_key=test_value", nil)
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		val := ctx.FormValue("test_key")
		assert.Equal(t, "test_value", val.Value)
		assert.Nil(t, val.Error)

		val = ctx.FormValue("not_exist")
		assert.NotNil(t, val.Error)
	})

	t.Run("test path param", func(t *testing.T) {
		ctx := &Context{
			Param: map[string]string{
				"id": "123",
			},
		}

		val := ctx.PathParam("id")
		assert.Equal(t, "123", val.Value)
		assert.Nil(t, val.Error)

		val = ctx.PathParam("not_exist")
		assert.Equal(t, "", val.Value)
		assert.NotNil(t, val.Error)
	})

	t.Run("test json response", func(t *testing.T) {
		w := httptest.NewRecorder()
		ctx := &Context{
			Resp: w,
		}

		type User struct {
			Name string `json:"name"`
		}
		user := &User{Name: "test"}

		err := ctx.JSON(http.StatusOK, user)
		assert.NoError(t, err)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name":"test"}`, string(body))
	})
}

func TestBindJSONError(t *testing.T) {
	ctx := &Context{
		Req: &http.Request{},
	}

	type User struct {
		Name string `json:"name"`
	}
	var user User

	err := ctx.BindJSON(&user)
	assert.Error(t, err)
	assert.Equal(t, "missing request body", err.Error())
}

func TestContextResponseMethods(t *testing.T) {
	testCases := []struct {
		name           string
		method         func(ctx *Context) error
		expectedStatus int
		expectedHeader string
		expectedBody   string
	}{
		{
			name: "RespJSON",
			method: func(ctx *Context) error {
				return ctx.RespJSON(http.StatusCreated, map[string]string{"foo": "bar"})
			},
			expectedStatus: http.StatusCreated,
			expectedHeader: "application/json; charset=utf-8",
			expectedBody:   `{"foo":"bar"}`,
		},
		{
			name: "RespString",
			method: func(ctx *Context) error {
				return ctx.RespString(http.StatusAccepted, "hello world")
			},
			expectedStatus: http.StatusAccepted,
			expectedHeader: "text/plain; charset=utf-8",
			expectedBody:   "hello world",
		},
		{
			name: "RespBytes",
			method: func(ctx *Context) error {
				return ctx.RespBytes(http.StatusOK, []byte("binary data"))
			},
			expectedStatus: http.StatusOK,
			expectedHeader: "application/octet-stream",
			expectedBody:   "binary data",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ctx := &Context{
				Resp: w,
				Req:  httptest.NewRequest(http.MethodGet, "/", nil),
			}

			err := tc.method(ctx)
			require.NoError(t, err)

			// Process the response as the server would
			s := &HTTPServer{}
			s.handleResponse(ctx)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			assert.Equal(t, tc.expectedHeader, resp.Header.Get("Content-Type"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBody, string(body))
		})
	}
}

func TestContextRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := &Context{
		Resp: w,
		Req:  httptest.NewRequest(http.MethodGet, "/", nil),
	}

	err := ctx.Redirect(http.StatusFound, "/redirect-target")
	require.NoError(t, err)

	assert.Equal(t, http.StatusFound, ctx.RespStatusCode)
	assert.Equal(t, "/redirect-target", w.Header().Get("Location"))
	assert.False(t, ctx.unhandled)
}

func TestContextChaining(t *testing.T) {
	w := httptest.NewRecorder()
	ctx := &Context{
		Resp: w,
		Req:  httptest.NewRequest(http.MethodGet, "/", nil),
	}

	// Test method chaining
	ctx.Status(http.StatusCreated).
		SetHeader("X-Custom", "value").
		SetHeader("X-Another", "another-value")

	assert.Equal(t, http.StatusCreated, ctx.RespStatusCode)
	assert.Equal(t, "value", w.Header().Get("X-Custom"))
	assert.Equal(t, "another-value", w.Header().Get("X-Another"))
}