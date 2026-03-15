package validation

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateIPAddress validates that the given string is a valid IP address.
func ValidateIPAddress(ip string) error {
	if ip == "" {
		return errors.New("IP address must not be empty")
	}
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("'%s' is not a valid IP address (expected IPv4 or IPv6 format)", ip)
	}
	return nil
}

// ValidateEmail validates that the given string is a valid email address.
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email address must not be empty")
	}
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("'%s' is not a valid email address", email)
	}
	return nil
}

// ValidateURL validates that the given string is a valid HTTP/HTTPS URL.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL must not be empty")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("'%s' is not a valid URL (must be a full http or https URL)", rawURL)
	}
	return nil
}

// ValidateRange validates that value is within [min, max] for the named field.
func ValidateRange(value float64, min, max float64, fieldName string) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("field '%s' must be a finite number", fieldName)
	}
	if value < min || value > max {
		return fmt.Errorf("field '%s' must be between %g and %g (got %g)", fieldName, min, max, value)
	}
	return nil
}

// FormatBindingErrors converts gin/validator binding errors into a human-readable message.
func FormatBindingErrors(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			msgs = append(msgs, formatFieldError(fe))
		}
		return strings.Join(msgs, "; ")
	}
	return err.Error()
}

func formatFieldError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("field '%s' is required", field)
	case "min":
		return fmt.Sprintf("field '%s' must be at least %s", field, fe.Param())
	case "max":
		return fmt.Sprintf("field '%s' must be at most %s", field, fe.Param())
	case "email":
		return fmt.Sprintf("field '%s' must be a valid email address", field)
	case "url":
		return fmt.Sprintf("field '%s' must be a valid URL", field)
	case "oneof":
		return fmt.Sprintf("field '%s' must be one of: %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("field '%s' must be greater than %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("field '%s' must be greater than or equal to %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("field '%s' must be less than %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("field '%s' must be less than or equal to %s", field, fe.Param())
	default:
		return fmt.Sprintf("field '%s' failed validation (%s)", field, fe.Tag())
	}
}
