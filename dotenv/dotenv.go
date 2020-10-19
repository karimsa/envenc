package dotenv

import (
	"fmt"
	"bufio"
	"bytes"
	"strings"
)

func Unmarshal(data []byte, values *map[string]interface{}) error {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	valMap := make(map[string]interface{})
	*values = valMap
	lineNumber := 1

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimLeft(line, " ")

		if len(line) > 0 && line[0] != '#' {
			equals := strings.IndexRune(line, '=')
			if equals < 0 {
				return fmt.Errorf("Unexpected syntax on line %d: %s", lineNumber, line)
			}

			valMap[line[:equals]] = line[equals+1:]
		}

		lineNumber++
	}

	return nil
}

func Marshal(values map[string]interface{}) ([]byte, error) {
	serialized := ""
	for key, val := range values {
		strVal, isStr := val.(string)
		if !isStr {
			return nil, fmt.Errorf("Unexpected %T at %s: %#v", val, key, val)
		}
		serialized += fmt.Sprintf("%s=%s\n", key, strVal)
	}
	return []byte(serialized), nil
}
