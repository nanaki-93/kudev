package config

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Errors []ErrorObj
}

type ErrorObj struct {
	Detail  string
	Example string
}

func (ve *ValidationError) Add(msg string) {
	ve.Errors = append(ve.Errors, ErrorObj{Detail: msg})
}
func (ve *ValidationError) AddWithExample(msg string, example string) {
	ve.Errors = append(ve.Errors, ErrorObj{Detail: msg, Example: example})
}

func (ve *ValidationError) Merge(other ValidationError) {
	ve.Errors = append(ve.Errors, other.Errors...)
}

func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
}

func (ve *ValidationError) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"Configuration validation failed (%d error%s):\n",
		len(ve.Errors),
		pluralize(len(ve.Errors)),
	))

	for i := range ve.Errors {
		sb.WriteString(fmt.Sprintf(" %d. %s\n", i+1, ve.Errors[i].Detail))
		if ve.Errors[i].Example != "" {
			example := ve.Errors[i].Example
			indentedExample := indentLines(example, "    ")
			sb.WriteString(fmt.Sprintf("    Example:\n%s\n", indentedExample))
		}
		sb.WriteString("\n")

	}

	return sb.String()
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func indentLines(text string, indent string) string {
	lines := strings.Split(text, "\n")
	for i := range lines {
		lines[i] = indent + lines[i]
	}
	return strings.Join(lines, "\n")
}

type FieldError struct {
	Field   string
	Message string
	Example string
}

func (fe *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", fe.Field, fe.Message)
}
