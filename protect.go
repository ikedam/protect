package protect

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

// Protector is the struct to customize the behavior of protect.
type Protector struct {
	// tagName is the tag name to specify fields to be protected.
	tagName string
	// optTagName is the tag name to specify options for protection.
	optTagName string

	// For testing: maps to override options
	sliceOptions sync.Map
	mapOptions   sync.Map

	// primitiveStructs is a map to store types that should be treated as primitive values
	primitiveStructs sync.Map
}

// DefaultProtector is the default Protector instance used by package level functions.
var DefaultProtector = NewProtector("protectfor", "protectopt")

// NewProtector creates a new instance of Protector with the specified tag names.
func NewProtector(tagName, optTagName string) *Protector {
	p := &Protector{
		tagName:    tagName,
		optTagName: optTagName,
	}

	// Register time.Time as a primitive struct by default
	p.AddPrimitiveStruct(&time.Time{})

	return p
}

// AddPrimitiveStruct registers a struct type to be treated as a primitive value when copying.
// This means the struct will be copied by direct assignment rather than field-by-field.
func (p *Protector) AddPrimitiveStruct(v interface{}) {
	t := reflect.TypeOf(v)

	// If it's a pointer, get the underlying type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Only register struct types
	if t.Kind() == reflect.Struct {
		p.primitiveStructs.Store(t, true)
	}
}

// IsPrimitiveStruct checks if a type should be treated as a primitive value.
func (p *Protector) IsPrimitiveStruct(t reflect.Type) bool {
	// If it's a pointer, get the underlying type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if it's registered as a primitive struct
	_, ok := p.primitiveStructs.Load(t)
	return ok
}

// Copy copies the values from src to dst excluding fields marked with the tag.
// The tag value should be a comma-separated list of values.
// If the tag contains the value specified by "tag", the field will be skipped.
func Copy(tag string, src, dst interface{}) error {
	return DefaultProtector.Copy(tag, src, dst)
}

// Copy copies the values from src to dst excluding fields marked with the tag.
// The tag value should be a comma-separated list of values.
// If the tag contains the value specified by "tag", the field will be skipped.
func (p *Protector) Copy(tag string, src, dst interface{}) error {
	if src == nil || dst == nil {
		return fmt.Errorf("src and dst must not be nil")
	}

	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)

	// Dereference pointers to get the actual value
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return fmt.Errorf("src must not be nil pointer")
		}
		srcVal = srcVal.Elem()
	}

	if dstVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	if dstVal.IsNil() {
		return fmt.Errorf("dst must not be nil pointer")
	}

	dstVal = dstVal.Elem()

	if srcVal.Type() != dstVal.Type() {
		return fmt.Errorf("src and dst must be the same type, got %s and %s", srcVal.Type(), dstVal.Type())
	}

	return p.copyValue(tag, srcVal, dstVal)
}

// Clone creates a deep copy of src.
func Clone(src interface{}) interface{} {
	return DefaultProtector.Clone(src)
}

// Clone creates a deep copy of src.
func (p *Protector) Clone(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	srcVal := reflect.ValueOf(src)

	// Handle pointer indirection
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return nil
		}

		// Create a new pointer of the same type
		dstVal := reflect.New(srcVal.Elem().Type())
		// Deep copy the pointed value
		p.copyValue("", srcVal.Elem(), dstVal.Elem())
		return dstVal.Interface()
	}

	// For non-pointer values
	dstVal := reflect.New(srcVal.Type())
	p.copyValue("", srcVal, dstVal.Elem())
	return dstVal.Elem().Interface()
}

// copyValue copies a value from src to dst, respecting protection tags.
func (p *Protector) copyValue(tag string, src, dst reflect.Value) error {
	if !src.IsValid() || !dst.IsValid() {
		return nil
	}

	// Check if it's a registered primitive struct type
	if src.Kind() == reflect.Struct && p.IsPrimitiveStruct(src.Type()) {
		// For primitive structs, treat them like basic types and copy directly
		if dst.CanSet() {
			dst.Set(src)
		}
		return nil
	}

	switch src.Kind() {
	case reflect.Struct:
		return p.copyStruct(tag, src, dst)
	case reflect.Ptr:
		return p.copyPtr(tag, src, dst)
	case reflect.Slice:
		return p.copySlice(tag, src, dst)
	case reflect.Map:
		return p.copyMap(tag, src, dst)
	case reflect.Interface:
		return p.copyInterface(tag, src, dst)
	default:
		// For basic types (int, string, bool, etc.), just set the value
		if src.CanInterface() && dst.CanSet() {
			dst.Set(src)
		}
		return nil
	}
}

// copyStruct copies a struct from src to dst, respecting protection tags.
func (p *Protector) copyStruct(tag string, src, dst reflect.Value) error {
	srcType := src.Type()

	for i := 0; i < srcType.NumField(); i++ {
		field := srcType.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check if the field should be protected
		if tag != "" {
			tagValue := field.Tag.Get(p.tagName)
			if isProtected(tagValue, tag) {
				continue
			}
		}

		srcField := src.Field(i)
		dstField := dst.Field(i)

		if !dstField.CanSet() {
			continue
		}

		if err := p.copyValue(tag, srcField, dstField); err != nil {
			return fmt.Errorf("error copying field %s: %w", field.Name, err)
		}
	}

	return nil
}

// copyPtr copies a pointer from src to dst.
func (p *Protector) copyPtr(tag string, src, dst reflect.Value) error {
	if src.IsNil() {
		// If source is nil, set destination to nil as well
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	// Create a new pointer if destination is nil
	if dst.IsNil() {
		dst.Set(reflect.New(dst.Type().Elem()))
	}

	// Copy the underlying value
	return p.copyValue(tag, src.Elem(), dst.Elem())
}

// copyInterface copies an interface from src to dst.
func (p *Protector) copyInterface(tag string, src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	// Get the concrete value from the interface
	srcElem := src.Elem()

	// Create a new instance of the concrete type
	dstElem := reflect.New(srcElem.Type()).Elem()

	// Copy the value
	if err := p.copyValue(tag, srcElem, dstElem); err != nil {
		return err
	}

	// Set the interface value
	dst.Set(dstElem)
	return nil
}

// For testing purposes: set slice option
func (p *Protector) setSliceOption(slice interface{}, option string) {
	p.sliceOptions.Store(fmt.Sprintf("%p", slice), option)
}

// For testing purposes: set map option
func (p *Protector) setMapOption(m interface{}, option string) {
	mapVal := reflect.ValueOf(m)
	if mapVal.Kind() == reflect.Map {
		// マップの一意なIDとしてメモリアドレスを使用
		p.mapOptions.Store(mapVal.UnsafePointer(), option)
	}
}

// getSliceOption gets the slice operation option from the options map or field tag
func (p *Protector) getSliceOption(sliceVal reflect.Value) string {
	// For testing: use override if available
	key := fmt.Sprintf("%p", sliceVal.Interface())
	if option, ok := p.sliceOptions.Load(key); ok {
		return option.(string)
	}

	// Default option
	return "overwrite"
}

// getMapOption gets the map operation option from the options map or field tag
func (p *Protector) getMapOption(mapVal reflect.Value) string {
	// For testing: use override if available
	if mapVal.Kind() == reflect.Map {
		if option, ok := p.mapOptions.Load(mapVal.UnsafePointer()); ok {
			return option.(string)
		}
	}

	// Default option
	return "overwrite"
}

// simpleCloneElement creates a simple clone of a value ignoring tags
func (p *Protector) simpleCloneElement(src reflect.Value) reflect.Value {
	// Simply clone the value without considering tags
	if !src.IsValid() {
		return reflect.Value{}
	}

	dst := reflect.New(src.Type()).Elem()

	// Check if it's a registered primitive struct type
	if src.Kind() == reflect.Struct && p.IsPrimitiveStruct(src.Type()) {
		// For primitive structs, treat them like basic types and copy directly
		dst.Set(src)
		return dst
	}

	switch src.Kind() {
	case reflect.Struct:
		srcType := src.Type()
		for i := 0; i < srcType.NumField(); i++ {
			field := srcType.Field(i)
			if !field.IsExported() {
				continue
			}

			srcField := src.Field(i)
			dstField := dst.Field(i)

			if dstField.CanSet() {
				clonedVal := p.simpleCloneElement(srcField)
				if clonedVal.IsValid() {
					dstField.Set(clonedVal)
				}
			}
		}
	case reflect.Ptr:
		if src.IsNil() {
			return dst // Zero value (nil pointer)
		}
		newPtr := reflect.New(src.Elem().Type())
		clonedVal := p.simpleCloneElement(src.Elem())
		if clonedVal.IsValid() {
			newPtr.Elem().Set(clonedVal)
		}
		dst.Set(newPtr)
	case reflect.Slice:
		if src.IsNil() {
			return dst // Zero value (nil slice)
		}
		newSlice := reflect.MakeSlice(src.Type(), src.Len(), src.Cap())
		for i := 0; i < src.Len(); i++ {
			clonedVal := p.simpleCloneElement(src.Index(i))
			if clonedVal.IsValid() {
				newSlice.Index(i).Set(clonedVal)
			}
		}
		dst.Set(newSlice)
	case reflect.Map:
		if src.IsNil() {
			return dst // Zero value (nil map)
		}
		newMap := reflect.MakeMap(src.Type())
		iter := src.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			clonedVal := p.simpleCloneElement(v)
			if clonedVal.IsValid() {
				newMap.SetMapIndex(k, clonedVal)
			}
		}
		dst.Set(newMap)
	case reflect.Interface:
		if src.IsNil() {
			return dst // Zero value (nil interface)
		}
		srcElem := src.Elem()
		clonedVal := p.simpleCloneElement(srcElem)
		if clonedVal.IsValid() {
			dst.Set(clonedVal)
		}
	default:
		// For basic types, copy as is
		dst.Set(src)
	}

	return dst
}

// copySlice copies a slice from src to dst.
func (p *Protector) copySlice(tag string, src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	// Get the slice option
	option := p.getSliceOption(dst)

	srcLen := src.Len()
	dstLen := dst.Len()

	switch option {
	case "overwrite":
		// Create a new slice with the same length as src
		newSlice := reflect.MakeSlice(dst.Type(), srcLen, srcLen)

		// For overwrite option, simply clone each element ignoring tags
		for i := 0; i < srcLen; i++ {
			srcElem := src.Index(i)
			dstElem := newSlice.Index(i)
			clonedElem := p.simpleCloneElement(srcElem)
			if clonedElem.IsValid() {
				dstElem.Set(clonedElem)
			}
		}

		dst.Set(newSlice)
	case "match":
		// Adjust destination length to match source length
		if dstLen != srcLen {
			newSlice := reflect.MakeSlice(dst.Type(), srcLen, srcLen)
			// Copy existing elements if available
			copyLen := dstLen
			if copyLen > srcLen {
				copyLen = srcLen
			}
			for i := 0; i < copyLen; i++ {
				newSlice.Index(i).Set(dst.Index(i))
			}
			dst.Set(newSlice)
		}

		// Copy each element with tag protection
		for i := 0; i < srcLen; i++ {
			srcElem := src.Index(i)
			dstElem := dst.Index(i)

			// Use copyValue recursively to handle different element types properly
			if i < dstLen {
				// For existing elements in destination, apply normal protection rules
				if err := p.copyValue(tag, srcElem, dstElem); err != nil {
					return err
				}
			} else {
				// For new elements, create with protection
				if srcElem.Kind() == reflect.Struct {
					// For struct types, we need special handling to respect protection tags
					structType := srcElem.Type()

					// Create a new struct
					newStructVal := reflect.New(structType).Elem()

					// Apply copyStruct to copy fields with protection
					if err := p.copyStruct(tag, srcElem, newStructVal); err != nil {
						return err
					}

					// Set the new struct to the destination element
					dstElem.Set(newStructVal)
				} else {
					// For non-struct types, use simple copy
					if err := p.copyValue(tag, srcElem, dstElem); err != nil {
						return err
					}
				}
			}
		}
	case "longer":
		// Keep destination length if it's longer than source
		if dstLen < srcLen {
			// Need to extend destination slice
			newSlice := reflect.MakeSlice(dst.Type(), srcLen, srcLen)
			// Copy existing elements
			for i := 0; i < dstLen; i++ {
				newSlice.Index(i).Set(dst.Index(i))
			}
			dst.Set(newSlice)
		}

		// Copy elements up to srcLen (with tag protection)
		copyLen := srcLen
		if copyLen > dstLen {
			copyLen = dstLen
		}

		// Apply same logic as match option for existing elements
		for i := 0; i < copyLen; i++ {
			srcElem := src.Index(i)
			dstElem := dst.Index(i)

			// Use copyValue to properly handle different types with protection rules
			if err := p.copyValue(tag, srcElem, dstElem); err != nil {
				return err
			}
		}
	case "shorter":
		// Copy elements up to the shorter length
		copyLen := srcLen
		if dstLen < copyLen {
			copyLen = dstLen
		}

		// Copy elements with tag protection
		for i := 0; i < copyLen; i++ {
			srcElem := src.Index(i)
			dstElem := dst.Index(i)

			// Use copyValue to properly handle different types with protection rules
			if err := p.copyValue(tag, srcElem, dstElem); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unknown slice option: %s", option)
	}

	return nil
}

// copyMap copies a map from src to dst.
func (p *Protector) copyMap(tag string, src, dst reflect.Value) error {
	if src.IsNil() {
		dst.Set(reflect.Zero(dst.Type()))
		return nil
	}

	// Get the map option
	option := p.getMapOption(dst)

	switch option {
	case "overwrite":
		// Create a new map
		newMap := reflect.MakeMap(dst.Type())

		// For overwrite option, simply clone each element ignoring tags
		iter := src.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()

			// Simple clone without considering tags
			clonedVal := p.simpleCloneElement(v)
			if clonedVal.IsValid() {
				newMap.SetMapIndex(k, clonedVal)
			}
		}

		dst.Set(newMap)
	case "match":
		// Create a new map
		newMap := reflect.MakeMap(dst.Type())

		// Copy values from source
		iter := src.MapRange()
		for iter.Next() {
			k := iter.Key()
			srcV := iter.Value()

			// Check if key exists in destination
			dstV := dst.MapIndex(k)

			if srcV.Kind() == reflect.Struct && dstV.IsValid() {
				structType := srcV.Type()

				// Create a temporary copy of srcV
				tempVal := reflect.New(structType).Elem()

				// First copy all fields from srcV to tempVal
				for i := 0; i < structType.NumField(); i++ {
					field := structType.Field(i)
					if !field.IsExported() {
						continue
					}

					srcField := srcV.Field(i)
					tempField := tempVal.Field(i)

					if tempField.CanSet() {
						tempField.Set(srcField)
					}
				}

				// Then copy all fields from dstV to tempVal for fields that should be protected
				if tag != "" {
					for i := 0; i < structType.NumField(); i++ {
						field := structType.Field(i)
						if !field.IsExported() {
							continue
						}

						tagValue := field.Tag.Get(p.tagName)
						if isProtected(tagValue, tag) {
							dstField := dstV.Field(i)
							tempField := tempVal.Field(i)

							if tempField.CanSet() && dstField.IsValid() {
								tempField.Set(dstField)
							}
						}
					}
				}

				// Set the value in the new map
				newMap.SetMapIndex(k, tempVal)
			} else {
				// For new keys or non-struct values, create a new value
				newV := reflect.New(srcV.Type()).Elem()

				// Copy with tag protection for existing keys
				if dstV.IsValid() {
					// First set to existing value
					newV.Set(dstV)
					// Then copy non-protected fields
					p.copyValue(tag, srcV, newV)
				} else {
					// For new keys, simple clone
					newV = p.simpleCloneElement(srcV)
				}

				if newV.IsValid() {
					newMap.SetMapIndex(k, newV)
				}
			}
		}

		dst.Set(newMap)
	case "patch":
		// Keep the existing map and add/update values from source
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(dst.Type()))
		}

		// Copy values from source, updating or adding as needed
		iter := src.MapRange()
		for iter.Next() {
			k := iter.Key()
			srcV := iter.Value()

			// If key exists in destination
			dstV := dst.MapIndex(k)

			if dstV.IsValid() {
				// Key exists
				if srcV.Kind() == reflect.Struct {
					structType := srcV.Type()

					// Create a temporary copy of dstV
					tempVal := reflect.New(structType).Elem()
					tempVal.Set(dstV)

					// Copy fields from srcV to tempVal except protected fields
					for i := 0; i < structType.NumField(); i++ {
						field := structType.Field(i)
						if !field.IsExported() {
							continue
						}

						tagValue := field.Tag.Get(p.tagName)
						if !isProtected(tagValue, tag) || tag == "" {
							// Copy this field from src to dst
							srcField := srcV.Field(i)
							tempField := tempVal.Field(i)

							if tempField.CanSet() {
								tempField.Set(srcField)
							}
						}
					}

					// Update the map with the modified value
					dst.SetMapIndex(k, tempVal)
				} else {
					// For non-struct type, use copyValue with tag protection
					tempV := reflect.New(srcV.Type()).Elem()
					tempV.Set(dstV)
					p.copyValue(tag, srcV, tempV)
					dst.SetMapIndex(k, tempV)
				}
			} else {
				// Key doesn't exist - simple clone
				clonedVal := p.simpleCloneElement(srcV)
				if clonedVal.IsValid() {
					dst.SetMapIndex(k, clonedVal)
				}
			}
		}
	default:
		return fmt.Errorf("unknown map option: %s", option)
	}

	return nil
}

// isProtected checks if the field with the given tag value should be protected for the specified tag.
func isProtected(tagValue, tag string) bool {
	if tagValue == "" || tag == "" {
		return false
	}

	// Split comma-separated values
	tags := strings.Split(tagValue, ",")

	for _, t := range tags {
		if strings.TrimSpace(t) == tag {
			return true
		}
	}

	return false
}

// CopySlice copies values from src to dst slice with the specified option.
// It specifically handles slice copying with more control than the regular Copy function.
// The tag value is used to protect fields in slice elements.
// The option parameter controls how slices are copied and can be one of:
// - "overwrite": Creates a new slice and copies all elements (default)
// - "match": Adjusts destination length to match source length
// - "longer": Keeps destination if longer than source, otherwise extends it
// - "shorter": Truncates to the shorter of the two slices
func CopySlice(tag string, src, dst interface{}, option string) error {
	return DefaultProtector.CopySlice(tag, src, dst, option)
}

// CopySlice copies values from src to dst slice with the specified option.
// It specifically handles slice copying with more control than the regular Copy function.
// The tag value is used to protect fields in slice elements.
// The option parameter controls how slices are copied and can be one of:
// - "overwrite": Creates a new slice and copies all elements (default)
// - "match": Adjusts destination length to match source length
// - "longer": Keeps destination if longer than source, otherwise extends it
// - "shorter": Truncates to the shorter of the two slices
func (p *Protector) CopySlice(tag string, src, dst interface{}, option string) error {
	if src == nil || dst == nil {
		return fmt.Errorf("src and dst must not be nil")
	}

	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)

	// Dereference pointers to get the actual value
	if srcVal.Kind() == reflect.Ptr {
		if srcVal.IsNil() {
			return fmt.Errorf("src must not be nil pointer")
		}
		srcVal = srcVal.Elem()
	}

	if dstVal.Kind() != reflect.Ptr {
		return fmt.Errorf("dst must be a pointer")
	}

	if dstVal.IsNil() {
		return fmt.Errorf("dst must not be nil pointer")
	}

	dstVal = dstVal.Elem()

	if srcVal.Type() != dstVal.Type() {
		return fmt.Errorf("src and dst must be the same type, got %s and %s", srcVal.Type(), dstVal.Type())
	}

	// Ensure both src and dst are slices
	if srcVal.Kind() != reflect.Slice || dstVal.Kind() != reflect.Slice {
		return fmt.Errorf("src and dst must be slices, got %s and %s", srcVal.Kind(), dstVal.Kind())
	}

	// Override slice option for this operation
	// Save the original option in a temporary variable
	p.sliceOptions.Store(fmt.Sprintf("%p", dstVal.Interface()), option)
	defer p.sliceOptions.Delete(fmt.Sprintf("%p", dstVal.Interface()))

	// Use the existing copySlice function with the specified option
	return p.copySlice(tag, srcVal, dstVal)
}
