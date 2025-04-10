package utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strings"
)

// StructToCsvHeader takes a struct type and returns a slice of strings representing the CSV header.
// It uses the `csv` tag on struct fields to determine the header name.
// If a field doesn't have a `csv` tag, the field name is used.
func StructToCsvHeader(t reflect.Type) []string {
	var headers []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		csvTag := field.Tag.Get("csv")

		// If the csv tag is present, use it as the header name, otherwise use the field name.
		headerName := field.Name
		if csvTag != "" {
			headerName = csvTag
		}
		headers = append(headers, headerName)
	}
	return headers
}

// WriteToCsvFile writes the given headers and data to a CSV file at the specified filePath.
// For slices, it joins the elements using a semicolon (;) to handle multi-value fields.
func WriteToCsvFile[T any](filePath string, headers []string, data []T) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the headers
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Write the data rows
	for _, item := range data {
		row := make([]string, len(headers))
		v := reflect.ValueOf(item)

		// If item is a pointer, get the value it points to
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() != reflect.Struct {
			return fmt.Errorf("data must be a slice of structs")
		}

		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			fieldValue := v.Field(i)

			// Get the index in the headers slice
			csvTag := field.Tag.Get("csv")
			headerName := field.Name
			if csvTag != "" {
				headerName = csvTag
			}

			idx := indexOf(headers, headerName)
			if idx < 0 {
				continue // Skip fields not in the headers
			}

			// Convert field value to string based on its kind
			var strValue string
			if fieldValue.Kind() == reflect.Slice {
				// Join slice elements with semicolon
				var sliceValues []string
				for j := 0; j < fieldValue.Len(); j++ {
					sliceValues = append(sliceValues, fmt.Sprintf("%v", fieldValue.Index(j).Interface()))
				}
				strValue = strings.Join(sliceValues, ";")
			} else {
				strValue = fmt.Sprintf("%v", fieldValue.Interface())
			}

			row[idx] = strValue
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// indexOf returns the index of a string in a slice or -1 if not found
func indexOf(slice []string, item string) int {
	for i, v := range slice {
		if v == item {
			return i
		}
	}
	return -1
}
