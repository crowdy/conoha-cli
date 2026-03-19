package output

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// FilterRows filters a slice of structs by the given key=value filters.
// Filters are ANDed. Field names are matched against json tags (case-insensitive).
// Non-slice data is returned as-is.
func FilterRows(data any, filters []string) (any, error) {
	if len(filters) == 0 {
		return data, nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		return data, nil
	}
	if val.Len() == 0 {
		return data, nil
	}

	// Parse filters
	type filterPair struct {
		field string
		value string
	}
	var parsed []filterPair
	for _, f := range filters {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid filter %q: expected key=value", f)
		}
		parsed = append(parsed, filterPair{field: strings.ToLower(parts[0]), value: strings.ToLower(parts[1])})
	}

	// Build field index map from json tags
	elem := val.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	elemType := elem.Type()
	fieldIndex := make(map[string]int, elemType.NumField())
	var fieldNames []string
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			name = field.Name
		}
		lower := strings.ToLower(name)
		fieldIndex[lower] = i
		fieldNames = append(fieldNames, name)
	}

	// Validate filter fields
	for _, fp := range parsed {
		if _, ok := fieldIndex[fp.field]; !ok {
			return nil, fmt.Errorf("unknown field %q; available fields: %s", fp.field, strings.Join(fieldNames, ", "))
		}
	}

	// Filter
	result := reflect.MakeSlice(val.Type(), 0, val.Len())
	for i := 0; i < val.Len(); i++ {
		row := val.Index(i)
		if row.Kind() == reflect.Ptr {
			row = row.Elem()
		}
		match := true
		for _, fp := range parsed {
			idx := fieldIndex[fp.field]
			fieldVal := strings.ToLower(fmt.Sprintf("%v", row.Field(idx).Interface()))
			if fieldVal != fp.value {
				match = false
				break
			}
		}
		if match {
			result = reflect.Append(result, val.Index(i))
		}
	}
	return result.Interface(), nil
}

// SortRows sorts a slice of structs by the given field name.
// Field name is matched against json tags (case-insensitive).
// Non-slice data is returned as-is.
func SortRows(data any, sortBy string) (any, error) {
	if sortBy == "" {
		return data, nil
	}

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		return data, nil
	}
	if val.Len() <= 1 {
		return data, nil
	}

	// Find field index
	elem := val.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	elemType := elem.Type()
	sortByLower := strings.ToLower(sortBy)
	fieldIdx := -1
	var fieldNames []string
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			name = field.Name
		}
		fieldNames = append(fieldNames, name)
		if strings.ToLower(name) == sortByLower {
			fieldIdx = i
		}
	}
	if fieldIdx < 0 {
		return nil, fmt.Errorf("unknown field %q; available fields: %s", sortBy, strings.Join(fieldNames, ", "))
	}

	// Make a sortable copy
	sorted := reflect.MakeSlice(val.Type(), val.Len(), val.Len())
	reflect.Copy(sorted, val)

	sort.SliceStable(sorted.Interface(), func(i, j int) bool {
		a := sorted.Index(i)
		b := sorted.Index(j)
		if a.Kind() == reflect.Ptr {
			a = a.Elem()
		}
		if b.Kind() == reflect.Ptr {
			b = b.Elem()
		}
		fa := a.Field(fieldIdx)
		fb := b.Field(fieldIdx)

		switch fa.Kind() {
		case reflect.String:
			return strings.ToLower(fa.String()) < strings.ToLower(fb.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return fa.Int() < fb.Int()
		case reflect.Float32, reflect.Float64:
			return fa.Float() < fb.Float()
		case reflect.Bool:
			return !fa.Bool() && fb.Bool()
		default:
			return fmt.Sprintf("%v", fa.Interface()) < fmt.Sprintf("%v", fb.Interface())
		}
	})

	return sorted.Interface(), nil
}
