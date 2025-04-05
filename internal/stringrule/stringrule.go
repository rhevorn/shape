// Package stringrule contains string predicates shared by fluent validators
// and compiled struct tags.
package stringrule

import (
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Email reports whether value is one plain mailbox address without display
// text or line breaks.
func Email(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value && !strings.ContainsAny(value, "\r\n")
}

// URL reports whether value is an absolute request URI with a host.
func URL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

// UUID reports whether value has the canonical hyphenated UUID shape.
func UUID(value string) bool { return uuidPattern.MatchString(value) }

// IP reports whether value is an IPv4 or IPv6 address.
func IP(value string) bool { return net.ParseIP(value) != nil }
