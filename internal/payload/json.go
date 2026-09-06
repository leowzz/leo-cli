package payload

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaptinlin/jsonrepair"
)

// FormatJSON repairs JSON-like text and unwraps serialized top-level payloads.
// Strings inside objects and arrays retain their original type.
func FormatJSON(text string, compact bool) (string, error) {
	value, err := ParseJSON(text)
	if err != nil {
		return "", err
	}
	return MarshalJSON(value, compact)
}

// ParseJSON accepts a payload on its own or an object/array embedded in a log.
func ParseJSON(text string) (any, error) {
	text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "\ufeff"))
	if !looksEncoded(text) {
		if extracted, ok := extractJSON(text); ok {
			return extracted, nil
		}
	}
	value, err := parseDocument(text)
	if err == nil {
		if _, scalar := value.(string); !scalar || looksEncoded(text) {
			return value, nil
		}
	}
	if extracted, ok := extractJSON(text); ok {
		return extracted, nil
	}
	return value, err
}

func parseDocument(text string) (any, error) {
	text = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(text), "\ufeff"))
	if text == "" {
		return nil, fmt.Errorf("payload is empty")
	}

	var value any
	for depth := 0; ; depth++ {
		if depth > 32 {
			return nil, fmt.Errorf("payload exceeds 32 string encoding layers")
		}
		// Python repr uses single quotes and escapes that JSON does not accept.
		if unwrapped, ok := pythonString(text); ok && looksEncoded(unwrapped) {
			text = strings.TrimSpace(unwrapped)
			continue
		}

		repaired := text
		if !json.Valid([]byte(text)) {
			if unwrapped, ok := bareEscapedJSON(text); ok {
				text = strings.TrimSpace(unwrapped)
				continue
			}
			repaired = normalizePythonStrings(normalizeNestedQuoteEscapes(text))
			if !json.Valid([]byte(repaired)) {
				source := repaired
				var err error
				repaired, err = jsonrepair.Repair(repaired)
				if err != nil || !json.Valid([]byte(repaired)) {
					if fixed := removeUnmatchedClosers(source); fixed != source {
						repaired, err = fixed, nil
						if !json.Valid([]byte(repaired)) {
							repaired, err = jsonrepair.Repair(repaired)
						}
					}
				}
				if err != nil {
					return nil, fmt.Errorf("repair JSON: %w", err)
				}
			}
		}
		decoder := json.NewDecoder(strings.NewReader(repaired))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil || !json.Valid([]byte(repaired)) {
			return nil, fmt.Errorf("repair did not produce valid JSON")
		}
		unwrapped, ok := value.(string)
		if !ok || !looksEncoded(unwrapped) || strings.TrimSpace(unwrapped) == text {
			break
		}
		text = strings.TrimSpace(unwrapped)
	}

	return value, nil
}

// MarshalJSON formats a parsed value without converting json.Number to float64.
func MarshalJSON(value any, compact bool) (string, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if !compact {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("format JSON: %w", err)
	}
	return strings.TrimSuffix(output.String(), "\n"), nil
}

func looksEncoded(text string) bool {
	text = strings.TrimSpace(text)
	return text != "" && (strings.ContainsRune("{[\"'\\", rune(text[0])) || strings.HasPrefix(text, "```"))
}

func pythonString(text string) (string, bool) {
	if len(text) < 2 || (text[0] != '\'' && text[0] != '"') || text[len(text)-1] != text[0] {
		return "", false
	}
	// JSON handles surrogate pairs and escaped slashes differently from Go.
	var decoded string
	if text[0] == '"' && json.Unmarshal([]byte(text), &decoded) == nil {
		return decoded, true
	}
	remaining := text[1 : len(text)-1]
	var output strings.Builder
	for remaining != "" {
		// Markdown copies can escape punctuation that is not a JSON escape.
		if len(remaining) >= 2 && remaining[0] == '\\' && strings.ContainsRune("!#$%&()*+,-.:;<=>?@[]^_`{|}~", rune(remaining[1])) {
			output.WriteByte(remaining[1])
			remaining = remaining[2:]
			continue
		}
		if strings.HasPrefix(remaining, `\'`) || strings.HasPrefix(remaining, `\"`) {
			output.WriteByte(remaining[1])
			remaining = remaining[2:]
			continue
		}
		char, _, tail, err := strconv.UnquoteChar(remaining, text[0])
		if err != nil {
			return "", false
		}
		output.WriteRune(char)
		remaining = tail
	}
	return output.String(), true
}

func bareEscapedJSON(text string) (string, bool) {
	start := strings.TrimLeft(text, " \t\r\n{[")
	if !strings.HasPrefix(start, `\"`) {
		return "", false
	}
	var decoded string
	err := json.Unmarshal([]byte(`"`+text+`"`), &decoded)
	return decoded, err == nil && decoded != text
}

// Convert complete Python string literals before repair can drop escapes such
// as \x1b or \U0001f600. Incomplete strings are left to the repair library.
func normalizePythonStrings(text string) string {
	var output strings.Builder
	for i := 0; i < len(text); {
		quote := text[i]
		if quote != '\'' && quote != '"' {
			output.WriteByte(text[i])
			i++
			continue
		}
		start := i
		i++
		for i < len(text) && text[i] != quote {
			if text[i] == '\\' && i+1 < len(text) {
				i++
			}
			i++
		}
		if i == len(text) {
			output.WriteString(text[start:])
			break
		}
		i++
		literal := text[start:i]
		if decoded, ok := pythonString(literal); ok {
			encoded, _ := json.Marshal(decoded)
			output.Write(encoded)
		} else {
			output.WriteString(literal)
		}
	}
	return output.String()
}
