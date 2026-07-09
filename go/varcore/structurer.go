package varcore

// structurer.go — groups scanned blocks into Examples, tracking heading scope
// and orphan attachments. Port of structurer.py / structurer.ts.

type scopeEntry struct {
	level int
	text  string
}

// Structure groups blocks into Examples, scoped by headings, with orphan
// attachments.
func Structure(path, source string, blocks []Block) VarDoc {
	examples := make([]Example, 0)
	orphanAttachments := make([]Block, 0)
	scopeStack := make([]scopeEntry, 0)
	lastExampleIdx := -1
	attachmentOpen := false

	for _, block := range blocks {
		switch b := block.(type) {
		case Heading:
			// Pop deeper-or-equal-level entries before pushing the new heading.
			for len(scopeStack) > 0 && scopeStack[len(scopeStack)-1].level >= b.Level {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
			scopeStack = append(scopeStack, scopeEntry{level: b.Level, text: b.Text})
			attachmentOpen = false

		case Paragraph, ListItem, Blockquote:
			// Gherkin shape: [paragraph, table, paragraph, fence] should be one
			// example. Merge when the previous example's last block is an
			// attachment (table/fence) AND there's no blank line between them.
			if attachmentOpen && lastExampleIdx >= 0 {
				prev := examples[lastExampleIdx]
				var prevLast Block
				if len(prev.Body) > 0 {
					prevLast = prev.Body[len(prev.Body)-1]
				}
				lastIsAttachment := prevLast != nil && (prevLast.kind() == "table" || prevLast.kind() == "fence")
				if lastIsAttachment {
					between := utf16Slice(source, prev.Sp.EndOffset, block.blockSpan().StartOffset)
					if !blankParaStop.MatchString(between) {
						examples[lastExampleIdx] = Example{
							ScopeStack: prev.ScopeStack,
							Sp:         spanFromOffsets(source, prev.Sp.StartOffset, block.blockSpan().EndOffset),
							Body:       append(append([]Block{}, prev.Body...), block),
						}
						continue
					}
				}
			}

			examples = append(examples, Example{
				ScopeStack: scopeTexts(scopeStack),
				Sp:         block.blockSpan(),
				Body:       []Block{block},
			})
			lastExampleIdx = len(examples) - 1
			attachmentOpen = true

		case Table, Fence:
			if attachmentOpen && lastExampleIdx >= 0 {
				prev := examples[lastExampleIdx]
				examples[lastExampleIdx] = Example{
					ScopeStack: prev.ScopeStack,
					Sp:         spanFromOffsets(source, prev.Sp.StartOffset, block.blockSpan().EndOffset),
					Body:       append(append([]Block{}, prev.Body...), block),
				}
			} else {
				orphanAttachments = append(orphanAttachments, block)
			}

		case ThematicBreak:
			attachmentOpen = false
		}
	}

	return VarDoc{
		Path:              path,
		Source:            source,
		Examples:          examples,
		OrphanAttachments: orphanAttachments,
	}
}

func scopeTexts(stack []scopeEntry) []string {
	out := make([]string, len(stack))
	for i, e := range stack {
		out[i] = e.text
	}
	return out
}
