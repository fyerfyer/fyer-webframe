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
