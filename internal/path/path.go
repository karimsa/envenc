package path

import (
	"fmt"
	"io"
	"strconv"
)

type token struct {
	key   string
	index int
}

func nextToken(str string) (token, string, error) {
	tok := token{}
	i := 1

	if str == "" {
		return tok, str, io.EOF
	}

	switch str[0] {
	case '.':
		for ; i < len(str); i++ {
			if str[i] == '[' || str[i] == '.' {
				break
			}
			tok.key += string(str[i])
		}

	case '[':
		idx := ""
		for ; i < len(str); i++ {
			if str[i] == ']' {
				i++
				break
			}
			idx += string(str[i])
		}

		if len(idx) == 0 {
			return tok, str, fmt.Errorf("Unexpected empty key")
		}

		if idx[0] == idx[len(idx)-1] && (idx[0] == '"' || idx[0] == '\'') {
			tok.key = idx[1 : len(idx)-1]
		} else {
			intIdx, err := strconv.ParseInt(idx, 10, 64)
			if err != nil {
				return tok, str, fmt.Errorf("Unexpected non-integer '%s' in: '%s'", idx, str)
			}
			if intIdx < 0 {
				return tok, str, fmt.Errorf("Unexpected negative index '%d' in: '%s'", intIdx, str)
			}
			tok.index = int(intIdx)
		}

	default:
		return tok, str, fmt.Errorf("Unexpected '%s' in: '%s'", string(str[i]), str)
	}

	return tok, str[i:], nil
}

func ReadFrom(path string, val interface{}) (string, error) {
	var tok token
	var err error
	visited := "."
	pathLeft := path

	for {
		tok, pathLeft, err = nextToken(pathLeft)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if tok.key == "" {
			slice, ok := val.([]interface{})
			if !ok {
				return "", fmt.Errorf("Cannot index non-list at %s (while reading %s)", visited, path)
			}
			if tok.index >= len(slice) {
				return "", fmt.Errorf("Index in path is out-of-range: %s (%s has length %d)", path, visited, len(slice))
			}
			val = slice[tok.index]
			visited += fmt.Sprintf("[%d]", tok.index)
		} else {
			mmap, ok := val.(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("Cannot read from non-map at %s (while reading %s)", visited, path)
			}
			v, ok := mmap[tok.key]
			if !ok {
				return "", fmt.Errorf("Could not find key %s in %s (while reading %s)", tok.key, visited, path)
			}
			val = v
			visited += fmt.Sprintf(".%s", tok.key)
		}
	}

	if pathLeft != "" {
		return "", fmt.Errorf("Key not found: '%s'", path)
	}
	return val.(string), nil
}
