package payload

import "strings"

// Only discard a closing delimiter when no corresponding opener is active.
// Mismatched nesting with an active opener is left to the repair library.
func removeUnmatchedClosers(text string) string {
	var output strings.Builder
	var stack []byte
	var quote byte
	var objects, arrays int
	for i := 0; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			output.WriteByte(char)
			if char == '\\' && i+1 < len(text) {
				i++
				output.WriteByte(text[i])
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if char == '/' && i+1 < len(text) {
			if text[i+1] == '/' {
				end := strings.IndexByte(text[i:], '\n')
				if end < 0 {
					output.WriteString(text[i:])
					break
				}
				output.WriteString(text[i : i+end])
				i += end - 1
				continue
			}
			if text[i+1] == '*' {
				end := strings.Index(text[i+2:], "*/")
				if end < 0 {
					output.WriteString(text[i:])
					break
				}
				end += i + 4
				output.WriteString(text[i:end])
				i = end - 1
				continue
			}
		}
		switch char {
		case '\'', '"':
			quote = char
		case '{':
			objects++
			stack = append(stack, char)
		case '[':
			arrays++
			stack = append(stack, char)
		case '}', ']':
			opener, active := byte('{'), objects
			if char == ']' {
				opener, active = '[', arrays
			}
			if active == 0 {
				continue
			}
			if len(stack) == 0 || stack[len(stack)-1] != opener {
				return text
			}
			stack = stack[:len(stack)-1]
			if char == '}' {
				objects--
			} else {
				arrays--
			}
		}
		output.WriteByte(char)
	}
	return output.String()
}
