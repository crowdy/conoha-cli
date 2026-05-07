package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
)

type CSVFormatter struct {
	NoHeaders bool
}

func (f *CSVFormatter) Format(w io.Writer, data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	// Handle single struct by wrapping as one-element slice
	if val.Kind() != reflect.Slice {
		if val.Kind() == reflect.Struct {
			val = reflect.Append(reflect.MakeSlice(reflect.SliceOf(val.Type()), 0, 1), val)
		} else {
			return fmt.Errorf("csv formatter requires a struct or slice, got %T", data)
		}
	}
	if val.Len() == 0 {
		return nil
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Get headers from struct field names or json tags
	elem := val.Index(0)
	if elem.Kind() == reflect.Pointer {
		elem = elem.Elem()
	}
	elemType := elem.Type()

	if !f.NoHeaders {
		headers := make([]string, elemType.NumField())
		for i := 0; i < elemType.NumField(); i++ {
			field := elemType.Field(i)
			name := field.Tag.Get("json")
			if name == "" || name == "-" {
				name = field.Name
			}
			headers[i] = name
		}
		if err := writer.Write(headers); err != nil {
			return err
		}
	}

	// Write rows
	for i := 0; i < val.Len(); i++ {
		row := val.Index(i)
		if row.Kind() == reflect.Pointer {
			row = row.Elem()
		}
		record := make([]string, row.NumField())
		for j := 0; j < row.NumField(); j++ {
			record[j] = fmt.Sprintf("%v", row.Field(j).Interface())
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}
