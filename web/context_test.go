package web

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestContext(t *testing.T) {
	t.Run("bind JSON", func(t *testing.T) {
		bodyReader := strings.NewReader(`{"name": "test"}`)
		req, err := http.NewRequest(http.MethodPost, "/test", bodyReader)
		req.Header.Set("Content-Type", "application/json")
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

	t.Run("bind XML", func(t *testing.T) {
		bodyReader := strings.NewReader(`<User><Name>test</Name></User>`)
		req, err := http.NewRequest(http.MethodPost, "/test", bodyReader)
		req.Header.Set("Content-Type", "application/xml")
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		type User struct {
			Name string `xml:"Name"`
		}
		var user User

		err = ctx.BindXML(&user)
		assert.NoError(t, err)
		assert.Equal(t, "test", user.Name)
	})

	t.Run("form values", func(t *testing.T) {
		formData := url.Values{}
		formData.Set("name", "test")
		formData.Set("age", "25")
		formData.Set("active", "true")
		formData.Set("height", "1.85")

		req, err := http.NewRequest(http.MethodPost, "/test", strings.NewReader(formData.Encode()))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		ctx := &Context{
			Req: req,
		}

		nameVal := ctx.FormValue("name")
		assert.Equal(t, "test", nameVal.Value)
		assert.Nil(t, nameVal.Error)

		ageVal := ctx.FormInt("age")
		assert.Equal(t, 25, ageVal.Value)
		assert.Nil(t, ageVal.Error)

		activeVal := ctx.FormBool("active")
		assert.Equal(t, true, activeVal.Value)
		assert.Nil(t, activeVal.Error)

		heightVal := ctx.FormFloat("height")
		assert.Equal(t, 1.85, heightVal.Value)
		assert.Nil(t, heightVal.Error)

		missingVal := ctx.FormValue("notexist")
		assert.NotNil(t, missingVal.Error)

		invalidInt := ctx.FormInt("name")
		assert.NotNil(t, invalidInt.Error)
	})

	t.Run("query parameters", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test?name=test&age=25&active=true&height=1.85", nil)
		require.NoError(t, err)

		ctx := &Context{
			Req: req,
		}

		nameVal := ctx.QueryParam("name")
		assert.Equal(t, "test", nameVal.Value)
		assert.Nil(t, nameVal.Error)

		ageVal := ctx.QueryInt("age")
		assert.Equal(t, 25, ageVal.Value)
		assert.Nil(t, ageVal.Error)

		activeVal := ctx.QueryBool("active")
		assert.Equal(t, true, activeVal.Value)
		assert.Nil(t, activeVal.Error)

		heightVal := ctx.QueryFloat("height")
		assert.Equal(t, 1.85, heightVal.Value)
		assert.Nil(t, heightVal.Error)

		missingVal := ctx.QueryParam("notexist")
		assert.NotNil(t, missingVal.Error)

		invalidInt := ctx.QueryInt("name")
		assert.NotNil(t, invalidInt.Error)

		allQuery := ctx.QueryAll()
		assert.Equal(t, 4, len(allQuery))
		assert.Equal(t, "test", allQuery.Get("name"))
	})

	t.Run("path parameters", func(t *testing.T) {
		ctx := &Context{
			Param: map[string]string{
				"id":     "123",
				"name":   "test",
				"active": "true",
				"height": "1.85",
			},
		}

		idVal := ctx.PathParam("id")
		assert.Equal(t, "123", idVal.Value)
		assert.Nil(t, idVal.Error)

		idIntVal := ctx.PathInt("id")
		assert.Equal(t, 123, idIntVal.Value)
		assert.Nil(t, idIntVal.Error)

		activeVal := ctx.PathBool("active")
		assert.Equal(t, true, activeVal.Value)
		assert.Nil(t, activeVal.Error)

		heightVal := ctx.PathFloat("height")
		assert.InDelta(t, 1.85, heightVal.Value, 0.001)
		assert.Nil(t, heightVal.Error)

		missingVal := ctx.PathParam("notexist")
		assert.NotNil(t, missingVal.Error)

		invalidInt := ctx.PathInt("name")
		assert.NotNil(t, invalidInt.Error)
	})

	t.Run("headers", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("X-Custom", "custom-value")
		req.Header.Add("X-Multi", "value1")
		req.Header.Add("X-Multi", "value2")

		ctx := &Context{
			Req:  req,
			Resp: httptest.NewRecorder(),
		}

		assert.Equal(t, "test-agent", ctx.GetHeader("User-Agent"))
		assert.Equal(t, "custom-value", ctx.GetHeader("X-Custom"))

		multiValues := ctx.GetHeaders("X-Multi")
		assert.Equal(t, 2, len(multiValues))
		assert.Equal(t, "value1", multiValues[0])
		assert.Equal(t, "value2", multiValues[1])

		ctx.SetHeader("Content-Type", "application/json").
			AddHeader("X-Response", "test")

		respHeader := ctx.Resp.Header()
		assert.Equal(t, "application/json", respHeader.Get("Content-Type"))
		assert.Equal(t, "test", respHeader.Get("X-Response"))
	})

	t.Run("content type helpers", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")

		ctx := &Context{
			Req: req,
		}

		assert.True(t, ctx.IsJSON())
		assert.False(t, ctx.IsXML())
		assert.Equal(t, "application/json; charset=utf-8", ctx.ContentType())
	})

	t.Run("client information", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/test", nil)
		require.NoError(t, err)
		req.Header.Set("User-Agent", "test-agent")
		req.Header.Set("Referer", "http://example.com")
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		req.RemoteAddr = "192.168.1.1:12345"

		ctx := &Context{
			Req: req,
		}

		assert.Equal(t, "203.0.113.195", ctx.ClientIP())

		assert.Equal(t, "test-agent", ctx.UserAgent())
		assert.Equal(t, "http://example.com", ctx.Referer())

		req.Header.Del("X-Forwarded-For")
		assert.Equal(t, "192.168.1.1", ctx.ClientIP())
	})
}

func TestContextWithValues(t *testing.T) {
	ctx := &Context{
		Context: context.WithValue(context.Background(), "key", "value"),
	}

	assert.Equal(t, "value", ctx.Context.Value("key"))
}

func TestReadBody(t *testing.T) {
	bodyContent := []byte("test body content")
	req, err := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(bodyContent))
	require.NoError(t, err)

	ctx := &Context{
		Req: req,
	}

	body, err := ctx.ReadBody()
	assert.NoError(t, err)
	assert.Equal(t, bodyContent, body)
}

func TestFileUploads(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("file", "test.txt")
	require.NoError(t, err)

	_, err = fw.Write([]byte("test file content"))
	require.NoError(t, err)

	err = w.WriteField("name", "test")
	require.NoError(t, err)

	w.Close()

	req, err := http.NewRequest(http.MethodPost, "/upload", &b)
	require.NoError(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())

	ctx := &Context{
		Req: req,
	}

	_, err = ctx.FormFile("file")
	assert.NoError(t, err)

	files, err := ctx.FormFiles("file")
	assert.NoError(t, err)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, "test.txt", files[0].Filename)

	nameVal := ctx.FormValue("name")
	assert.Equal(t, "test", nameVal.Value)
	assert.Nil(t, nameVal.Error)
}

func TestCookies(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{
		Name:  "test-cookie",
		Value: "cookie-value",
	})

	w := httptest.NewRecorder()
	ctx := &Context{
		Req:  req,
		Resp: w,
	}

	cookie, err := ctx.GetCookie("test-cookie")
	assert.NoError(t, err)
	assert.Equal(t, "test-cookie", cookie.Name)
	assert.Equal(t, "cookie-value", cookie.Value)

	ctx.SetCookie(&http.Cookie{
		Name:  "response-cookie",
		Value: "response-value",
	})

	cookies := w.Result().Cookies()
	assert.Equal(t, 1, len(cookies))
	assert.Equal(t, "response-cookie", cookies[0].Name)
	assert.Equal(t, "response-value", cookies[0].Value)
}