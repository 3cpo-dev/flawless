// Package config loads flawless configuration.
//
// Flawless deliberately has zero runtime dependencies, so instead of pulling
// in a YAML library it parses the small, documented subset of YAML that its
// config files use: nested maps by 2-space indentation, scalars, quoted
// strings, flow lists ([a, b]) and block lists (- a). Anchors, multi-line
// scalars, multi-document streams and other YAML arcana are intentionally
// not supported.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// parseYAML parses the supported YAML subset into nested map[string]any,
// []any and scalar (string/bool/int64) values.
func parseYAML(src string) (map[string]any, error) {
	root := map[string]any{}
	type frame struct {
		indent int
		m      map[string]any
	}
	stack := []frame{{indent: -1, m: root}}
	var pendingKey string // key whose value may be a nested block
	var pendingIndent int // indent of the pending key's line
	var pendingList []any // accumulating block list, nil if none
	lines := strings.Split(src, "\n")

	flushPending := func() {
		if pendingKey == "" {
			return
		}
		top := stack[len(stack)-1]
		if pendingList != nil {
			top.m[pendingKey] = pendingList
		} else {
			top.m[pendingKey] = map[string]any{} // "key:" with no children
		}
		pendingKey, pendingList = "", nil
	}

	for n, raw := range lines {
		line := stripComment(raw)
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if strings.HasPrefix(strings.TrimLeft(line, " "), "\t") {
			return nil, fmt.Errorf("line %d: tabs are not allowed for indentation", n+1)
		}
		content := strings.TrimSpace(line)

		// Block list item under a pending key.
		if strings.HasPrefix(content, "- ") || content == "-" {
			if pendingKey == "" || indent <= pendingIndent {
				return nil, fmt.Errorf("line %d: list item without a parent key", n+1)
			}
			item := strings.TrimSpace(strings.TrimPrefix(content, "-"))
			pendingList = append(pendingList, parseScalar(item))
			continue
		}

		// A key line. First resolve any pending key from before.
		key, rest, ok := splitKey(content)
		if !ok {
			return nil, fmt.Errorf("line %d: expected 'key: value', got %q", n+1, content)
		}

		if pendingKey != "" {
			if pendingList == nil && indent > pendingIndent {
				// The pending key opens a nested map; this line is its first child.
				child := map[string]any{}
				stack[len(stack)-1].m[pendingKey] = child
				stack = append(stack, frame{indent: indent, m: child})
				pendingKey = ""
			} else {
				flushPending()
			}
		}

		// Pop frames until this line's indent fits the current map.
		for len(stack) > 1 && indent < stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 1 && indent != stack[len(stack)-1].indent && indent != 0 {
			return nil, fmt.Errorf("line %d: inconsistent indentation", n+1)
		}
		if indent == 0 {
			stack = stack[:1]
		}

		if rest == "" {
			pendingKey, pendingIndent = key, indent
			continue
		}
		stack[len(stack)-1].m[key] = parseScalar(rest)
	}
	flushPending()
	return root, nil
}

// splitKey splits "key: value" respecting quoted keys are not supported;
// keys are bare words. Returns ok=false if there is no colon.
func splitKey(s string) (key, rest string, ok bool) {
	i := strings.Index(s, ":")
	if i < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(s[:i])
	rest = strings.TrimSpace(s[i+1:])
	if key == "" {
		return "", "", false
	}
	return key, rest, true
}

// stripComment removes a trailing # comment that is not inside quotes.
func stripComment(line string) string {
	inS, inD := false, false
	for i, r := range line {
		switch r {
		case '\'':
			if !inD {
				inS = !inS
			}
		case '"':
			if !inS {
				inD = !inD
			}
		case '#':
			if !inS && !inD && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
				return line[:i]
			}
		}
	}
	return line
}

// parseScalar interprets a scalar token: quoted string, bool, int, flow list
// or plain string.
func parseScalar(s string) any {
	if s == "" {
		return ""
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return []any{}
		}
		var out []any
		for _, part := range strings.Split(inner, ",") {
			out = append(out, parseScalar(strings.TrimSpace(part)))
		}
		return out
	}
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	switch s {
	case "true", "yes", "on":
		return true
	case "false", "no", "off":
		return false
	case "null", "~":
		return ""
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	return s
}
