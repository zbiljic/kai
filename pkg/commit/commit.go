package commit

import (
	"fmt"
	"strings"
)

type Message struct {
	Type          string
	Scope         string
	Breaking      bool
	CommitMessage string
}

// ToString converts the Message struct into a string representation.
func (m Message) ToString() string {
	var out string
	if m.Type != "" {
		if strings.HasSuffix(m.Type, "!") {
			m.Type = m.Type[:len(m.Type)-1]
			m.Breaking = true
		}
		m.Type = strings.TrimSpace(m.Type)
		out += m.Type
		if m.Scope != "" {
			if strings.HasSuffix(m.Scope, "!") {
				m.Scope = m.Scope[:len(m.Scope)-1]
				m.Breaking = true
			}
			m.Scope = strings.TrimSpace(m.Scope)
			out += fmt.Sprintf("(%s)", m.Scope)
		}
		if m.Breaking {
			out += "!"
		}
		out += ": "
	}
	m.CommitMessage = strings.TrimSpace(m.CommitMessage)
	out += m.CommitMessage
	return out
}

type Type int

const (
	// SimpleType denotes a basic type of commit without any specific
	// format or structure.
	SimpleType Type = iota
	// ConventionalType represents a commit type that adheres to the
	// conventional commit format.
	ConventionalType
)

var TypeIds = map[Type][]string{
	SimpleType:       {"simple"},
	ConventionalType: {"conventional"},
}

// ParseType parses a string and returns the corresponding Type.
// It returns an error if the string doesn't match any known Type.
func ParseType(s string) (Type, error) {
	for t, ids := range TypeIds {
		for _, id := range ids {
			if strings.EqualFold(id, s) {
				return t, nil
			}
		}
	}
	return Type(0), fmt.Errorf("unknown type: %s", s)
}

// ToString converts the Type value to a string representation.
func (t Type) ToString() string {
	if val, ok := TypeIds[t]; ok {
		return val[0]
	}
	return fmt.Sprintf("UnknownType(%d)", t)
}

// commitTypeFormats provides format templates for different commit types.
var commitTypeFormats = map[Type]string{
	// SimpleType uses a basic commit message format without any additional
	// structure.
	SimpleType: "<commit message>",
	// ConventionalType follows the conventional commit format where a type
	// and optional scope are specified.
	ConventionalType: "<type>(<optional scope>): <commit message>",
}

// CommitFormat returns the format template associated with the commit type.
func (t Type) CommitFormat() string {
	// Retrieve the format template for the provided Type from the
	// commitTypeFormats map. It's assumed that the Type has a
	// corresponding entry in the map.
	return commitTypeFormats[t]
}
