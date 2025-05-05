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

type TestSliceStruct struct {
	Items []TestStruct `json:"items"`
}

func TestBindSlice(t *testing.T) {
	t.Run("Overwrite option", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `[{"id":"123", "code":"ABC", "name":"Test1"},{"id":"456", "code":"DEF", "name":"Test2"}]`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Create destination with pre-existing content
		dst := []TestStruct{
			{ID: "existing", Code: "existing", Name: "existing"},
			{ID: "existing2", Code: "existing2", Name: "existing2"},
			{ID: "existing3", Code: "existing3", Name: "existing3"},
		}

		// Make context rebindable
		c = ReBindable(c)

		// Bind with "overwrite" option
		err := BindSlice("create", c, &dst, "overwrite")
		assert.NoError(t, err)

		// Verify results
		assert.Equal(t, 2, len(dst)) // Length should match source

		// In overwrite, tags should be ignored
		assert.Equal(t, "123", dst[0].ID) // ID is not protected in overwrite mode
		assert.Equal(t, "ABC", dst[0].Code)
		assert.Equal(t, "Test1", dst[0].Name)
	})

	t.Run("Match option", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `[{"id":"123", "code":"ABC", "name":"Test1"},{"id":"456", "code":"DEF", "name":"Test2"}]`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Create destination with pre-existing content
		dst := []TestStruct{
			{ID: "existing", Code: "existing", Name: "existing"},
			{ID: "existing2", Code: "existing2", Name: "existing2"},
			{ID: "existing3", Code: "existing3", Name: "existing3"},
		}

		// Make context rebindable
		c = ReBindable(c)

		// Bind with "match" option
		err := BindSlice("create", c, &dst, "match")
		assert.NoError(t, err)

		// Verify results
		assert.Equal(t, 2, len(dst)) // Length should match source

		// ID should be protected
		assert.Equal(t, "existing", dst[0].ID) // ID is preserved
		assert.Equal(t, "ABC", dst[0].Code)
		assert.Equal(t, "Test1", dst[0].Name)

		assert.Equal(t, "existing2", dst[1].ID) // ID is preserved
		assert.Equal(t, "DEF", dst[1].Code)
		assert.Equal(t, "Test2", dst[1].Name)
	})

	t.Run("Longer option", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `[{"id":"123", "code":"ABC", "name":"Test1"}]`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Create destination with pre-existing content (longer)
		dst := []TestStruct{
			{ID: "existing", Code: "existing", Name: "existing"},
			{ID: "existing2", Code: "existing2", Name: "existing2"},
			{ID: "existing3", Code: "existing3", Name: "existing3"},
		}

		originalLength := len(dst)

		// Make context rebindable
		c = ReBindable(c)

		// Bind with "longer" option
		err := BindSlice("create", c, &dst, "longer")
		assert.NoError(t, err)

		// Verify results
		assert.Equal(t, originalLength, len(dst)) // Length should remain the same (longer option)

		// First item should be updated with protected fields
		assert.Equal(t, "existing", dst[0].ID) // ID is preserved
		assert.Equal(t, "ABC", dst[0].Code)
		assert.Equal(t, "Test1", dst[0].Name)

		// Other items should remain unchanged
		assert.Equal(t, "existing2", dst[1].ID)
		assert.Equal(t, "existing2", dst[1].Code)
	})

	t.Run("Shorter option", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `[{"id":"123", "code":"ABC", "name":"Test1"},{"id":"456", "code":"DEF", "name":"Test2"},{"id":"789", "code":"GHI", "name":"Test3"}]`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Create destination with pre-existing content (shorter)
		dst := []TestStruct{
			{ID: "existing", Code: "existing", Name: "existing"},
			{ID: "existing2", Code: "existing2", Name: "existing2"},
		}

		originalLength := len(dst)

		// Make context rebindable
		c = ReBindable(c)

		// Bind with "shorter" option
		err := BindSlice("create", c, &dst, "shorter")
		assert.NoError(t, err)

		// Verify results
		assert.Equal(t, originalLength, len(dst)) // Length should remain the same (shorter option)

		// Items should be updated with protected fields
		assert.Equal(t, "existing", dst[0].ID) // ID is preserved
		assert.Equal(t, "ABC", dst[0].Code)
		assert.Equal(t, "Test1", dst[0].Name)

		assert.Equal(t, "existing2", dst[1].ID) // ID is preserved
		assert.Equal(t, "DEF", dst[1].Code)
		assert.Equal(t, "Test2", dst[1].Name)

		// Third item from source should not be copied
	})

	t.Run("With rebindable context", func(t *testing.T) {
		// Set up Echo and the request
		e := echo.New()
		reqBody := `[{"id":"123", "code":"ABC", "name":"Test1"},{"id":"456", "code":"DEF", "name":"Test2"}]`
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Make context rebindable
		c = ReBindable(c)

		// First bind - overwrite
		dst1 := []TestStruct{{ID: "existing", Code: "existing", Name: "existing"}}
		err := BindSlice("create", c, &dst1, "overwrite")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(dst1))
		assert.Equal(t, "123", dst1[0].ID) // In overwrite mode, tags are ignored

		// Second bind - match
		dst2 := []TestStruct{{ID: "existing", Code: "existing", Name: "existing"}}
		err = BindSlice("create", c, &dst2, "match")
		assert.NoError(t, err)
		assert.Equal(t, 2, len(dst2))
		assert.Equal(t, "existing", dst2[0].ID) // ID is preserved
	})
}
