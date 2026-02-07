package config

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Details  []string
	Examples []string
}

//todo refactor

//	type ValidationError struct {
//		ErrorObj []ErrorObj
//	}
//
//	type ErrorObj struct{
//		Detail string
//		Example string
//	}
func (ve *ValidationError) Add(msg string) {
	ve.Details = append(ve.Details, msg)
	if len(ve.Examples) < len(ve.Details) {
		ve.Examples = append(ve.Examples, "")
	}
}

func (ve *ValidationError) AddExample(msg string) {
	if len(ve.Details) == 0 {
		return
	}

	if len(ve.Examples) < len(ve.Details) {
		ve.Examples = append(ve.Examples, "")
	}
	ve.Examples[len(ve.Examples)-1] = msg
}
func (ve *ValidationError) Merge(other ValidationError) {
	ve.Details = append(ve.Details, other.Details...)
	ve.Examples = append(ve.Examples, other.Examples...)
}

func (ve *ValidationError) HasErrors() bool {
	return len(ve.Details) > 0
}

func (ve *ValidationError) Error() string {
	if len(ve.Details) == 0 {
		return "no validation errors"
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(
		"Configuration validation failed (%d error%s):\n",
		len(ve.Details),
		pluralize(len(ve.Details)),
	))

	for i, _ := range ve.Details {
		sb.WriteString(fmt.Sprintf(" %d. %s\n", i+1, ve.Details[i]))
		if len(ve.Examples) > i && ve.Examples[i] != "" {
			example := ve.Examples[i]
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
