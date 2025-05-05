package protectecho

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

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

// BindSlice binds the request data to the provided destination slice
// and applies the protection rules specified by the tag,
// with specific slice copying behavior controlled by the option parameter.
// This is similar to Bind but with more control over how slices are handled.
//
// The option parameter controls how slices are copied and can be one of:
// - "overwrite": Creates a new slice and copies all elements (default)
// - "match": Adjusts destination length to match source length
// - "longer": Keeps destination if longer than source, otherwise extends it
// - "shorter": Truncates to the shorter of the two slices
func BindSlice(tag string, c echo.Context, dst interface{}, option string) error {
	// Validate dst is a pointer to a slice
	dstVal := reflect.ValueOf(dst)
	if dstVal.Kind() != reflect.Ptr || dstVal.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dst must be a pointer to a slice, got %T", dst)
	}

	// Create a clone of dst to work with
	dstType := dstVal.Type()
	elemType := dstVal.Elem().Type().Elem()
	cloneDst := reflect.New(dstType.Elem()).Interface()

	// Read the request body
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	defer c.Request().Body.Close()

	// Reset the request body for further use
	c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

	// Try to decode the JSON data
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return err
	}

	// Determine if the JSON is an array or an object
	var jsonArray []interface{}
	switch data := jsonData.(type) {
	case []interface{}:
		// Direct array in JSON root
		jsonArray = data
	case map[string]interface{}:
		// Look for array in the object
		for _, v := range data {
			if arr, ok := v.([]interface{}); ok {
				jsonArray = arr
				break
			}
		}
	}

	if jsonArray == nil {
		return fmt.Errorf("no array found in the JSON data")
	}

	// Create a new slice with the data from JSON
	sliceVal := reflect.MakeSlice(reflect.SliceOf(elemType), len(jsonArray), len(jsonArray))

	// Convert JSON array items to struct elements
	for i, item := range jsonArray {
		if mapItem, ok := item.(map[string]interface{}); ok {
			elemPtr := reflect.New(elemType)

			// Use json.Marshal/Unmarshal to properly convert the map to a struct
			itemBytes, err := json.Marshal(mapItem)
			if err != nil {
				return err
			}

			if err := json.Unmarshal(itemBytes, elemPtr.Interface()); err != nil {
				return err
			}

			// Set the new element in the slice
			sliceVal.Index(i).Set(elemPtr.Elem())
		}
	}

	// Set the created slice to the clone
	reflect.ValueOf(cloneDst).Elem().Set(sliceVal)

	// Apply protection rules with specified slice option
	return protect.CopySlice(tag, cloneDst, dst, option)
}

// updateDataField updates a slice field with data from a JSON array
func updateDataField(field reflect.Value, jsonArray []interface{}, elemType reflect.Type) error {
	// Create a new slice with the appropriate length
	newSlice := reflect.MakeSlice(reflect.SliceOf(elemType), len(jsonArray), len(jsonArray))

	// Create each element and set it in the slice
	for i, item := range jsonArray {
		if mapItem, ok := item.(map[string]interface{}); ok {
			// Create a new element
			elemPtr := reflect.New(elemType)
			elem := elemPtr.Elem()

			// Fill the fields from the map
			for j := 0; j < elem.NumField(); j++ {
				fieldInfo := elemType.Field(j)
				jsonName := strings.Split(fieldInfo.Tag.Get("json"), ",")[0]
				if jsonName == "" {
					jsonName = strings.ToLower(fieldInfo.Name)
				}

				if fieldValue, exists := mapItem[jsonName]; exists {
					field := elem.Field(j)
					if field.CanSet() {
						switch field.Kind() {
						case reflect.String:
							if strVal, ok := fieldValue.(string); ok {
								field.SetString(strVal)
							}
						case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
							if numVal, ok := fieldValue.(float64); ok {
								field.SetInt(int64(numVal))
							}
						case reflect.Float32, reflect.Float64:
							if numVal, ok := fieldValue.(float64); ok {
								field.SetFloat(numVal)
							}
						case reflect.Bool:
							if boolVal, ok := fieldValue.(bool); ok {
								field.SetBool(boolVal)
							}
						}
					}
				}
			}

			// Set the element in the slice
			newSlice.Index(i).Set(elem)
		}
	}

	// Set the field to the new slice
	field.Set(newSlice)
	return nil
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
