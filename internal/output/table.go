package output

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"text/tabwriter"
)

type TableFormatter struct {
	NoHeaders bool
}

func (f *TableFormatter) Format(w io.Writer, data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	// If not a slice, wrap single struct as one-row table
	if val.Kind() != reflect.Slice {
		if val.Kind() == reflect.Struct {
			val = reflect.Append(reflect.MakeSlice(reflect.SliceOf(val.Type()), 0, 1), val)
		} else {
			_, err := fmt.Fprintf(w, "%v\n", data)
			return err
		}
	}

	if val.Len() == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	// Get headers
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
			headers[i] = strings.ToUpper(name)
		}
		if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
			return err
		}
	}

	// Write rows
	for i := 0; i < val.Len(); i++ {
		row := val.Index(i)
		if row.Kind() == reflect.Pointer {
			row = row.Elem()
		}
		fields := make([]string, row.NumField())
		for j := 0; j < row.NumField(); j++ {
			fields[j] = fmt.Sprintf("%v", row.Field(j).Interface())
		}
		if _, err := fmt.Fprintln(tw, strings.Join(fields, "\t")); err != nil {
			return err
		}
	}

	return tw.Flush()
}
