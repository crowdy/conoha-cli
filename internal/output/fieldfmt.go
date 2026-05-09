package output

import (
	"fmt"
	"reflect"
)

// formatFieldValue renders a struct field value for tabular formatters
// (table/csv). It dereferences pointer values so that *int / *string fields
// display the underlying value rather than the pointer address (#193).
// nil pointers render as the empty string.
func formatFieldValue(v reflect.Value) string {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	return fmt.Sprintf("%v", v.Interface())
}
