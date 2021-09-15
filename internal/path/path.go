package path

import (
	"fmt"
	"io"
	"strconv"
)

type tokenType int

const (
	tokenUnknown tokenType = iota
	tokenKey
	tokenIndex
)

type token struct {
	tokenType tokenType
	key       string
	index     int
}

func nextToken(str string) (token, string, error) {
	tok := token{}
	i := 1

	if str == "" {
		return tok, str, io.EOF
	}

	// treating '.[' the same as '['
	if str[0:2] == ".[" {
		str = str[1:]
	}

	switch str[0] {
	case '.':
		for ; i < len(str); i++ {
			if str[i] == '[' || str[i] == '.' {
				break
			}

			tok.tokenType = tokenKey
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
			tok.tokenType = tokenKey
			tok.key = idx[1 : len(idx)-1]
		} else {
			tok.tokenType = tokenIndex
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

type Path struct {
	tokens   []token
	asString string
}

func (p Path) String() string {
	return p.asString
}

func New(strPath string) (Path, error) {
	tokens := make([]token, 0, 10)
	asString := strPath

	for {
		tok, str, err := nextToken(strPath)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Path{}, err
		}

		if tok.tokenType == tokenUnknown {
			return Path{}, fmt.Errorf("Syntax error at column %d of '%s'", len(asString)-len(strPath), asString)
		}
		if tok.tokenType == tokenKey && tok.key == "" {
			if len(tokens) == 0 {
				continue
			}
			return Path{}, fmt.Errorf("Syntax error at column %d of '%s'", len(asString)-len(strPath), asString)
		}

		tokens = append(tokens, tok)
		strPath = str
	}

	return Path{
		tokens:   tokens,
		asString: asString,
	}, nil
}

func (path Path) AppendKey(key string) Path {
	return Path{
		tokens: append(path.tokens, token{
			tokenType: tokenKey,
			key:       key,
		}),
		asString: fmt.Sprintf("%s['%s']", path.String(), key),
	}
}

func (path Path) AppendIndex(index int) Path {
	return Path{
		tokens: append(path.tokens, token{
			tokenType: tokenIndex,
			index:     index,
		}),
		asString: fmt.Sprintf("%s[%d]", path.String(), index),
	}
}

func (path Path) Equals(compared Path) bool {
	if len(path.tokens) != len(compared.tokens) {
		return false
	}

	for i, left := range path.tokens {
		right := compared.tokens[i]
		if left.key != right.key || left.index != right.index {
			return false
		}
	}

	return true
}

func (path Path) ReadFrom(val interface{}) (string, error) {
	visited := "."
	pathLeft := path.tokens

	for {
		if len(pathLeft) == 0 {
			break
		}

		tok := pathLeft[0]
		pathLeft = pathLeft[1:]

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

	if len(pathLeft) > 0 {
		return "", fmt.Errorf("Key not found: '%s'", path)
	}
	return val.(string), nil
}
