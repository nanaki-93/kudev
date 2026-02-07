package config

import (
	"strings"
	"testing"
)

func TestValidationError_Add(t *testing.T) {

	obj := ValidationError{}
	obj.Add("error")
	if len(obj.Errors) != 1 {
		t.Errorf("Error not added")
	}

	if obj.Errors[0].Detail != "error" {
		t.Errorf("Error message not added")
	}
}

func TestValidationError_AddWithExample(t *testing.T) {

	obj := ValidationError{}
	obj.AddWithExample("error", "example")
	if len(obj.Errors) != 1 {
		t.Errorf("Error not added")
	}

	if obj.Errors[0].Detail != "error" {
		t.Errorf("Error message not added")
	}
	if obj.Errors[0].Example != "example" {
		t.Errorf("Example not added")
	}
}

func TestValidationError_HasErrors(t *testing.T) {

	obj := ValidationError{}
	if obj.HasErrors() {
		t.Errorf("HasErrors() returned true when there were no errors")
	}
	obj.Add("error")
	if !obj.HasErrors() {
		t.Errorf("HasErrors() returned false when there were errors")
	}
}

func TestValidationError_Merge(t *testing.T) {
	obj1 := ValidationError{}
	obj1.Add("error1")
	obj2 := ValidationError{}
	obj2.Add("error2")

	obj1.Merge(obj2)
	if len(obj1.Errors) != 2 {
		t.Errorf("Errors not merged")
	}
	if obj1.Errors[0].Detail != "error1" {
		t.Errorf("Error message not merged correctly, error1 expected, got %q", obj1.Errors[0].Detail)
	}
	if obj1.Errors[1].Detail != "error2" {
		t.Errorf("Error message not merged correctly, error2 expected, got %q", obj1.Errors[1].Detail)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name         string
		ve           ValidationError
		expectedErrs []string
	}{{
		name:         "no errors",
		ve:           ValidationError{},
		expectedErrs: []string{ErrNoValidationErrors},
	},
		{name: "one error without example",
			ve:           ValidationError{[]ErrorObj{{Detail: "error"}}},
			expectedErrs: []string{"Configuration validation failed (1 error):", "1. error"},
		},
		{name: "one error with example",
			ve:           ValidationError{[]ErrorObj{{Detail: "error", Example: "example"}}},
			expectedErrs: []string{"Configuration validation failed (1 error):", "1. error", "example"},
		},
		{name: "two errors without examples",
			ve:           ValidationError{[]ErrorObj{{Detail: "error1"}, {Detail: "error2"}}},
			expectedErrs: []string{"Configuration validation failed (2 errors):", " 1. error1", "2. error2"},
		},
		{name: "multiple errors with mixed examples",
			ve: ValidationError{[]ErrorObj{
				{Detail: "error1"},
				{Detail: "error2", Example: "example2"},
				{Detail: "error3"},
				{Detail: "error4", Example: "example4"},
				{Detail: "error5"},
			},
			},
			expectedErrs: []string{"Configuration validation failed (5 errors):",
				" 1. error1",
				"2. error2",
				"3. error3",
				"4. error4",
				"5. error5",
				"example2",
				"example4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ve.Error()
			for _, expectedErr := range tt.expectedErrs {
				if !stringContains(err, expectedErr) {
					t.Errorf("Error message %q does not contain %q", err, expectedErr)
				}
			}
		})
	}
}

func TestValidationError_ErrorInterface(t *testing.T) {
	ve := ValidationError{}
	ve.Add("test error")

	var err error = &ve
	if err.Error() == "" {
		t.Errorf("ValidationError does not properly implement error interface")
	}
}

func TestFieldErrors_Error(t *testing.T) {
	tests := []struct {
		name        string
		fe          FieldError
		expectedErr []string
	}{
		{name: "no error", fe: FieldError{}, expectedErr: []string{}},
		{
			name:        "complete error",
			fe:          FieldError{"field1", "error", "example"},
			expectedErr: []string{"field1", "error", "example"},
		},
		{
			name:        "error no example",
			fe:          FieldError{"field1", "error", ""},
			expectedErr: []string{"field1: error"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fe.Error()
			if len(tt.expectedErr) > len(strings.Split(err, " ")) {
				t.Errorf("Expected %d words, got %d", len(tt.expectedErr), len(strings.Split(err, " ")))
			}
			for _, expectedErr := range tt.expectedErr {
				if !stringContains(err, expectedErr) {
					t.Errorf("Error message %q does not contain %q", err, expectedErr)
				}
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"zero", 0, "s"},
		{"one", 1, ""},
		{"two", 2, "s"},
		{"many", 10, "s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pluralize(tt.n); got != tt.want {
				t.Errorf("pluralize(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestIndentLines(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		indent string
		want   string
	}{
		{"single line", "line1", "  ", "  line1"},
		{"multiple lines", "line1\nline2", "--", "--line1\n--line2"},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indentLines(tt.text, tt.indent); got != tt.want {
				t.Errorf("indentLines(%q, %q) = %q, want %q", tt.text, tt.indent, got, tt.want)
			}
		})
	}
}
