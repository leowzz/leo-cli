package payload

import (
	"encoding/json"
	"strings"
)

func extractJSON(text string) (any, bool) {
	var candidates []string
	for i := 0; i < len(text); i++ {
		if text[i] != '{' && text[i] != '[' {
			continue
		}
		// A strict decoder finds the end even when log text follows the value.
		decoder := json.NewDecoder(strings.NewReader(text[i:]))
		decoder.UseNumber()
		var value any
		if decoder.Decode(&value) == nil && isContainer(value) {
			return value, true
		}
		// Scan repaired quote boundaries, otherwise escaped brackets in string
		// contents can be mistaken for the end of the surrounding object.
		text = normalizeNestedQuoteEscapes(text[i:])
		i = 0
		decoder = json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		if decoder.Decode(&value) == nil && isContainer(value) {
			return value, true
		}
		end := containerEnd(text, i)
		candidates = append(candidates, text[i:end])
		i = end - 1
	}
	// Prefer objects over log tags such as [INFO] when repair is needed.
	for _, opener := range []byte{'{', '['} {
		for _, candidate := range candidates {
			if candidate[0] != opener {
				continue
			}
			if value, err := parseDocument(candidate); err == nil && isContainer(value) {
				return value, true
			}
		}
	}
	return nil, false
}

func containerEnd(text string, start int) int {
	var stack []byte
	var quote byte
	for i := start; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			if char == '\\' {
				i++
			} else if char == quote {
				quote = 0
			}
			continue
		}
		switch char {
		case '\'', '"':
			quote = char
		case '{', '[':
			stack = append(stack, char)
		case '}', ']':
			if len(stack) == 0 {
				return i + 1
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return i + 1
			}
		}
	}
	return len(text)
}

func isContainer(value any) bool {
	switch value.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

// ParseStringJSON only offers objects or arrays as deeper expansion targets.
func ParseStringJSON(text string) (any, bool) {
	value, err := InspectStringJSON(text)
	return value, err == nil && isContainer(value)
}

// InspectStringJSON returns a container or its parse error. Ordinary strings
// return (nil, nil), allowing viewers to retain failed expansion candidates.
func InspectStringJSON(text string) (any, error) {
	if !looksEncoded(text) {
		return nil, nil
	}
	value, err := parseDocument(text)
	if err != nil {
		return nil, err
	}
	if !isContainer(value) {
		return nil, nil
	}
	return value, nil
}

// Some logs add a second backslash to quotes inside an encoded JSON field,
// while leaving the enclosing document unescaped. Repair only those fields.
func normalizeNestedQuoteEscapes(text string) string {
	var output strings.Builder
	var inString, encoded bool
	for i := 0; i < len(text); i++ {
		char := text[i]
		if inString && char == '\\' {
			start := i
			for i < len(text) && text[i] == '\\' {
				i++
			}
			count := i - start
			if i < len(text) && text[i] == '"' {
				if encoded && count%2 == 0 {
					count--
				}
				output.WriteString(strings.Repeat("\\", count))
				output.WriteByte('"')
				if count%2 == 0 {
					inString = false
				}
				continue
			}
			output.WriteString(text[start:i])
			i--
			continue
		}
		if char == '"' {
			inString = !inString
			if inString {
				rest := strings.TrimLeft(text[i+1:], " \t\r\n")
				encoded = strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, "[")
			}
		}
		output.WriteByte(char)
	}
	return output.String()
}
