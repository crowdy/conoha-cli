package output

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type filterOp int

const (
	opEq filterOp = iota
	opContains
	opRegex
)

// FilterRows filters a slice of structs by the given filters. Supported
// operators (checked in order):
//
//	key~=regex  — field matches regex (case-insensitive)
//	key~value   — field contains value (case-insensitive substring)
//	key=value   — field exactly equals value (case-insensitive)
//
// Filters are ANDed. Field names are matched against json tags
// (case-insensitive). Non-slice data is returned as-is.
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
		op    filterOp
		value string
		re    *regexp.Regexp
	}
	var parsed []filterPair
	for _, f := range filters {
		key, op, value, err := splitFilter(f)
		if err != nil {
			return nil, err
		}
		fp := filterPair{field: strings.ToLower(key), op: op, value: strings.ToLower(value)}
		if op == opRegex {
			re, err := regexp.Compile("(?i)" + value)
			if err != nil {
				return nil, fmt.Errorf("invalid filter %q: bad regex: %w", f, err)
			}
			fp.re = re
		}
		parsed = append(parsed, fp)
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
			var ok bool
			switch fp.op {
			case opContains:
				ok = strings.Contains(fieldVal, fp.value)
			case opRegex:
				ok = fp.re.MatchString(fieldVal)
			default:
				ok = fieldVal == fp.value
			}
			if !ok {
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

// splitFilter parses a filter expression into (key, op, value). Operators are
// checked longest-first so "~=" is recognised before "~" or "=".
func splitFilter(f string) (string, filterOp, string, error) {
	if i := strings.Index(f, "~="); i >= 0 {
		if i == 0 {
			return "", 0, "", fmt.Errorf("invalid filter %q: empty key", f)
		}
		return f[:i], opRegex, f[i+2:], nil
	}
	if i := strings.Index(f, "~"); i >= 0 {
		if i == 0 {
			return "", 0, "", fmt.Errorf("invalid filter %q: empty key", f)
		}
		return f[:i], opContains, f[i+1:], nil
	}
	if i := strings.Index(f, "="); i >= 0 {
		if i == 0 {
			return "", 0, "", fmt.Errorf("invalid filter %q: empty key", f)
		}
		return f[:i], opEq, f[i+1:], nil
	}
	return "", 0, "", fmt.Errorf("invalid filter %q: expected key=value, key~value, or key~=regex", f)
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
