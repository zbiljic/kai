package cmd

import (
	"fmt"
	"strings"

	"github.com/zbiljic/kai/pkg/llm"
)

// getHunksForCommit retrieves the hunks for a specific commit from the hunk map
func getHunksForCommit(hunkIDs []string, hunkMap map[string]*llm.Hunk) []*llm.Hunk {
	var hunks []*llm.Hunk
	for _, id := range hunkIDs {
		if hunk, exists := hunkMap[id]; exists {
			hunks = append(hunks, hunk)
		}
	}
	return hunks
}

// convertHunksToPatch converts a list of hunks to a unified patch format
func convertHunksToPatch(hunks []*llm.Hunk) string {
	if len(hunks) == 0 {
		return ""
	}

	// Group hunks by file path to create proper patch structure
	fileHunks := make(map[string][]*llm.Hunk)
	for _, hunk := range hunks {
		fileHunks[hunk.FilePath] = append(fileHunks[hunk.FilePath], hunk)
	}

	var patchBuilder strings.Builder

	for filePath, hunks := range fileHunks {
		// Add git diff header for the file
		patchBuilder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", filePath, filePath))
		patchBuilder.WriteString("index 0000000..0000000 100644\n")
		patchBuilder.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
		patchBuilder.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))

		// Add each hunk's content
		for _, hunk := range hunks {
			// The hunk.Content should already contain the proper diff format
			// including the @@ header and line changes
			patchBuilder.WriteString(hunk.Content)
			if !strings.HasSuffix(hunk.Content, "\n") {
				patchBuilder.WriteString("\n")
			}
		}
	}

	return patchBuilder.String()
}
