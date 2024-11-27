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
