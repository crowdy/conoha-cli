package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
)

type CSVFormatter struct{}

func (f *CSVFormatter) Format(w io.Writer, data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Handle slice of structs
	if val.Kind() != reflect.Slice {
		return fmt.Errorf("csv formatter requires a slice, got %T", data)
	}
	if val.Len() == 0 {
		return nil
	}

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Get headers from struct field names or json tags
	elem := val.Index(0)
	if elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}
	elemType := elem.Type()

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

	// Write rows
	for i := 0; i < val.Len(); i++ {
		row := val.Index(i)
		if row.Kind() == reflect.Ptr {
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
