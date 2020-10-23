package orderedmap

import (
	"bufio"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

type OrderedMap struct {
	KeyOrder map[string][]string
	Values   map[string]interface{}
}

func (om OrderedMap) addKey(path, key string) {
	if om.KeyOrder[path] == nil {
		om.KeyOrder[path] = make([]string, 0, 10)
	}
	om.KeyOrder[path] = append(om.KeyOrder[path], key)
}

func (om OrderedMap) Export(format string) ([]byte, error) {
	switch format {
	case "yaml":
		doc, err := om.toMapSlice(".", om.Values)
		if err != nil {
			return nil, err
		}
		return yaml.Marshal(doc)

	case "dotenv":
		output := ""
		for _, key := range om.KeyOrder["."] {
			output += fmt.Sprintf("%s=%s\n", key, om.Values[key].(string))
		}
		return []byte(output), nil

	default:
		return nil, fmt.Errorf("Unsupported export format: %s", format)
	}
}

func (om OrderedMap) toMapSlice(currentPath string, currentMap map[string]interface{}) (yaml.MapSlice, error) {
	mapSlice := make(yaml.MapSlice, len(currentMap))

	keys, keysExist := om.KeyOrder[currentPath]
	if !keysExist {
		return mapSlice, fmt.Errorf("Failed to find key order at '%s'", currentPath)
	}
	if len(keys) != len(currentMap) {
		return mapSlice, fmt.Errorf("Found mismatched map size at %s: %d != %d", currentPath, len(keys), len(currentMap))
	}

	for i, key := range keys {
		mapSlice[i] = yaml.MapItem{
			Key: key,
		}

		if subMap, isMap := currentMap[key].(map[string]interface{}); isMap {
			subMS, err := om.toMapSlice(currentPath+"."+key, subMap)
			if err != nil {
				return mapSlice, err
			}
			mapSlice[i].Value = subMS
		} else {
			mapSlice[i].Value = currentMap[key]
		}
	}

	return mapSlice, nil
}

func mapSliceToOrderedMap(mapSlice yaml.MapSlice, orderedMap OrderedMap, currentPath string, currentMap map[string]interface{}) error {
	for _, entry := range mapSlice {
		switch key := entry.Key.(type) {
		case string:
			keyPath := currentPath + "." + key
			if currentPath == "." {
				keyPath = keyPath[1:]
			}
			orderedMap.addKey(currentPath, key)

			switch value := entry.Value.(type) {
			case int:
				currentMap[key] = value
			case float64:
				currentMap[key] = value
			case bool:
				currentMap[key] = value
			case string:
				currentMap[key] = value

			case yaml.MapSlice:
				nextMap := make(map[string]interface{}, len(value))
				currentMap[key] = nextMap
				if err := mapSliceToOrderedMap(value, orderedMap, keyPath, nextMap); err != nil {
					return err
				}

			default:
				return fmt.Errorf("Unrecognized type %T at %s: %#v", value, keyPath, value)
			}

		case nil:
			// skip nil keys

		default:
			return fmt.Errorf("Unexpected key of type %T at %s: %#v", key, currentPath, entry)
		}
	}

	return nil
}

var (
	supportedFormats = map[string]func(io.Reader) (OrderedMap, error){
		// json cannot be supported right now, not until this issue is dealt
		// with: https://github.com/golang/go/issues/27179
		// or a custom/third-party parser is used

		"dotenv": func(reader io.Reader) (OrderedMap, error) {
			doc := OrderedMap{
				KeyOrder: make(map[string][]string, 100),
				Values:   make(map[string]interface{}, 100),
			}
			bufReader := bufio.NewReader(reader)
			lineNumber := 1

			var err error
			var line string

			for err != io.EOF {
				line, err = bufReader.ReadString('\n')
				if err != nil && err != io.EOF {
					return doc, err
				}
				if len(line) > 0 && line[len(line)-1] == '\n' {
					line = line[:len(line)-1]
				}

				if len(line) > 0 && line[0] != '#' {

					equals := strings.IndexRune(line, '=')
					if equals < 0 {
						return doc, fmt.Errorf("Unexpected syntax on line %d: '%s'", lineNumber, line)
					}

					key := line[:equals]
					if matched, err := regexp.MatchString("^[a-zA-Z0-9_\\.]+$", key); err != nil {
						return doc, fmt.Errorf("Invalid key on line %d: %s (%s)", lineNumber, key, err)
					} else if !matched {
						return doc, fmt.Errorf("Invalid key on line %d: %s", lineNumber, key)
					}

					value := line[equals+1:]
					if value[len(value)-1] == '\n' {
						value = value[:len(value)-1]
					}

					doc.Values[key] = value
					doc.KeyOrder["."] = append(doc.KeyOrder["."], key)
				}

				lineNumber++
			}

			return doc, nil
		},

		"yaml": func(reader io.Reader) (OrderedMap, error) {
			orderedMap := OrderedMap{
				KeyOrder: make(map[string][]string, 100),
				Values:   make(map[string]interface{}, 100),
			}

			vals := make(yaml.MapSlice, 10)
			data, err := ioutil.ReadAll(reader)
			if err != nil {
				return orderedMap, err
			}
			err = yaml.Unmarshal(data, &vals)
			if err != nil {
				return orderedMap, err
			}

			return orderedMap, mapSliceToOrderedMap(vals, orderedMap, ".", orderedMap.Values)
		},
	}
)

func Parse(format string, reader io.Reader) (OrderedMap, error) {
	if parser, ok := supportedFormats[format]; ok {
		return parser(reader)
	} else {
		return OrderedMap{}, fmt.Errorf("Unrecognized env file format: %s", format)
	}
}
