package kubeconfig

import (
	"fmt"
	"regexp"
	"strings"
)

type ContextValidator struct {
	AllowedContexts      []string
	ForceContext         bool
	CurrentContext       string
	AllAvailableContexts []string
}

func NewContextValidator(forceContext bool) (*ContextValidator, error) {
	current, err := LoadCurrentContext()
	if err != nil {
		return nil, err
	}
	available, _ := ListAvailableContexts()
	return &ContextValidator{
		AllowedContexts:      defaultAllowedContexts(),
		ForceContext:         forceContext,
		CurrentContext:       current.Name,
		AllAvailableContexts: available,
	}, nil
}

func (cv *ContextValidator) Validate() error {

	if cv.isAllowed(cv.CurrentContext) {
		return nil
	}
	if cv.ForceContext {
		return nil
	}
	return cv.createBlockedError()
}

func (cv *ContextValidator) ValidateContext(context string) error {
	if cv.isAllowed(context) {
		return nil
	}
	if cv.ForceContext {
		return nil
	}
	return fmt.Errorf(
		"context %q not in whitelist\n\n"+
			"Allowed contexts: %v\n"+
			"Use --force-context to override",
		context,
		cv.AllowedContexts,
	)
}

func (cv *ContextValidator) isAllowed(context string) bool {
	for _, pattern := range cv.AllowedContexts {
		if matches(context, pattern) {
			return true
		}
	}
	return false
}

func matches(name, pattern string) bool {
	if name == pattern {
		return true
	}
	if strings.Contains(pattern, "*") {
		regexPattern := "^" + regexp.QuoteMeta(pattern)
		regexPattern = strings.ReplaceAll(regexPattern, `\*`, ".*")
		regexPattern += "$"

		if regex, err := regexp.Compile(regexPattern); err == nil {
			return regex.MatchString(name)
		}
	}
	return false
}

func (cv *ContextValidator) createBlockedError() error {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("current context %q is not in whitelist\n\n", cv.CurrentContext))

	msg.WriteString("Allowed contexts (for safety):\n")
	for _, context := range cv.AllowedContexts {
		msg.WriteString(fmt.Sprintf("- %s\n", context))
	}
	if len(cv.AllAvailableContexts) > 0 {
		msg.WriteString(fmt.Sprintf("\nAvailable contexts in kubeconfig:\n"))
		for _, context := range cv.AllAvailableContexts {
			marker := " "
			if context == cv.CurrentContext {
				marker = "*"
			}
			msg.WriteString(fmt.Sprintf("- %s %s\n", marker, context))
		}
	}
	msg.WriteString(fmt.Sprintf(
		"\nTo override and proceed at your own risk: \n" +
			" kudev --force-context <command>\n\n" +
			"To change context: \n" +
			" kubectl config use-context <context-name>\n"))
	return fmt.Errorf("%s", msg.String())
}

func defaultAllowedContexts() []string {
	return []string{
		"docker-desktop",
		"docker-for-desktop",
		"minikube",
		"kind-*",
		"k3d-*",
		"*-local*",
		"localhost",
		"127.0.0.1",
	}
}

// WithAllowedContexts sets custom allowed contexts (for testing).
func (cv *ContextValidator) WithAllowedContexts(contexts []string) *ContextValidator {
	cv.AllowedContexts = contexts
	return cv
}

// WithCurrentContext sets current context (for testing).
func (cv *ContextValidator) WithCurrentContext(name string) *ContextValidator {
	cv.CurrentContext = name
	return cv
}
