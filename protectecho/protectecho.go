package protectecho

import (
	"bytes"
	"io"

	"github.com/ikedam/protect"
	"github.com/labstack/echo/v4"
)

// Bind binds the request data to the provided destination struct
// and applies the protection rules specified by the tag.
// This is a wrapper around echo.Context.Bind() that adds protection.
func Bind(tag string, c echo.Context, dst interface{}) error {
	// Create a clone of the destination
	clone := protect.Clone(dst)

	// Bind the request data to the clone
	if err := c.Bind(clone); err != nil {
		return err
	}

	// Apply protection rules
	return protect.Copy(tag, clone, dst)
}

// rebindableContext is a wrapper around echo.Context that allows rebinding.
type rebindableContext struct {
	echo.Context
	body []byte
}

// ReBindable wraps the echo.Context to allow for multiple calls to Bind().
// By default, Echo's Context.Bind() can only be called once because it consumes the request body.
// This function creates a wrapper that saves the request body so it can be re-used.
func ReBindable(c echo.Context) echo.Context {
	// If it's already a rebindableContext, just return it
	if _, ok := c.(*rebindableContext); ok {
		return c
	}

	// Read the request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c
	}

	// Close the original body
	c.Request().Body.Close()

	// Create a new buffered reader with the body content
	c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

	// Return a new rebindableContext
	return &rebindableContext{
		Context: c,
		body:    body,
	}
}

// Bind overrides the echo.Context.Bind() method to allow rebinding.
func (c *rebindableContext) Bind(i interface{}) error {
	// Reset the request body
	c.Request().Body = io.NopCloser(bytes.NewBuffer(c.body))

	// Call the original Bind method
	return c.Context.Bind(i)
}
