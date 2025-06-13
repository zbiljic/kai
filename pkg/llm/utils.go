package llm

import (
	"fmt"
	"sort"
	"strings"
)

func mapToString(m map[string]string) string {
	var entries []string
	for k, v := range m {
		entries = append(entries, fmt.Sprintf(`- "%s": "%s"`, k, v))
	}
	sort.Strings(entries)
	return strings.Join(entries, "\n")
}

// mapToTable formats a map as a markdown table with two columns: Type and Description
func mapToTable(m map[string]string) string {
	var sb strings.Builder

	// Add table header
	sb.WriteString("| Type | Description |\n")
	sb.WriteString("| ---- | ----------- |\n")

	// Get sorted keys for consistent output
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Add table rows
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("| %s | %s |\n", k, m[k]))
	}

	return sb.String()
}
