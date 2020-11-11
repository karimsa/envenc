package path

import (
	"io"
	"fmt"
	"strconv"
)

type token struct {
	key string
	index int64
}

func nextToken(str string) (token, string, error) {
	var err error
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

		tok.index, err = strconv.ParseInt(idx, 10, 64)
		if err != nil {
			return tok, str, fmt.Errorf("Unexpected non-integer '%s' in: '%s'", idx, str)
		}

	default:
		return tok, str, fmt.Errorf("Unexpected '%s' in: '%s'", string(str[i]), str)
	}

	return tok, str[i:], nil
}
