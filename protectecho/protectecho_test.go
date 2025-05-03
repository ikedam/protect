package protectecho

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	ID   string `protectfor:"create,update" json:"id"`
	Code string `protectfor:"update" json:"code"`
	Name string `json:"name"`
}

func TestBind(t *testing.T) {
	t.Run("Create mode", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `{"id":"123", "code":"ABC", "name":"Test"}`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Create destination struct and bind
		dst := TestStruct{}
		err := Bind("create", c, &dst)
		assert.NoError(t, err)

		// Verify protection - ID should be protected, Code and Name should be copied
		assert.Empty(t, dst.ID)
		assert.Equal(t, "ABC", dst.Code)
		assert.Equal(t, "Test", dst.Name)
	})

	t.Run("Update mode", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `{"id":"123", "code":"ABC", "name":"Test"}`
		req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Create destination struct and bind
		dst := TestStruct{}
		err := Bind("update", c, &dst)
		assert.NoError(t, err)

		// Verify protection - ID and Code should be protected, Name should be copied
		assert.Empty(t, dst.ID)
		assert.Empty(t, dst.Code)
		assert.Equal(t, "Test", dst.Name)
	})
}

func TestReBindable(t *testing.T) {
	// Set up Echo and the request
	e := echo.New()
	reqBody := `{"id":"123", "code":"ABC", "name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Make context rebindable
	c = ReBindable(c)

	// First bind - should work
	dst1 := TestStruct{}
	err := Bind("create", c, &dst1)
	assert.NoError(t, err)
	assert.Empty(t, dst1.ID)
	assert.Equal(t, "ABC", dst1.Code)
	assert.Equal(t, "Test", dst1.Name)

	// Second bind - would fail with normal Echo context, but should work with ReBindable
	dst2 := TestStruct{}
	err = Bind("update", c, &dst2)
	assert.NoError(t, err)
	assert.Empty(t, dst2.ID)
	assert.Empty(t, dst2.Code)
	assert.Equal(t, "Test", dst2.Name)
}

func TestReBindableIdempotent(t *testing.T) {
	// Set up Echo and the request
	e := echo.New()
	reqBody := `{"id":"123", "code":"ABC", "name":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Make context rebindable
	c1 := ReBindable(c)
	// Call ReBindable again - should return the same context
	c2 := ReBindable(c1)

	// Check that ReBindable is idempotent
	assert.Same(t, c1, c2)
}
