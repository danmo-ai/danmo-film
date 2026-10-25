package scriptparse

import "strings"

// outputProtocol builds the shared three-part output contract:
// placeholders schema + field notes + one filled minimal example.
func outputProtocol(schemaJSON, fieldNotes, exampleJSON string) string {
	var b strings.Builder
	b.WriteString("## Output (critical)\n")
	b.WriteString(jsonOutputCritical)
	b.WriteString("\n\n")
	b.WriteString(jsonSchemaInstruction)
	b.WriteString("\n\n## Schema (placeholders)\n```json\n")
	b.WriteString(strings.TrimSpace(schemaJSON))
	b.WriteString("\n```\n\n## Field notes\n")
	b.WriteString(strings.TrimSpace(fieldNotes))
	b.WriteString("\n\n## Example (filled — shape only; replace values from the user message)\n```json\n")
	b.WriteString(strings.TrimSpace(exampleJSON))
	b.WriteString("\n```")
	return b.String()
}

// inputLayout documents the labeled sections present in the user message.
func inputLayout(sections ...string) string {
	if len(sections) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Input (user message contains)\n")
	for _, s := range sections {
		b.WriteString("- ")
		b.WriteString(s)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
