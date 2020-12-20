package orderedmap

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	orderedJson "github.com/iancoleman/orderedmap"
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

	case "json":
		doc, err := om.toJSON(".", om.Values)
		if err != nil {
			return nil, err
		}
		return json.MarshalIndent(doc, "", "\t")

	default:
		return nil, fmt.Errorf("Unsupported export format: %s", format)
	}
}

func pathJoin(path, key string) string {
	if path == "" {
		return key
	}

	keyPath := path + "." + key
	if keyPath[len(keyPath)-1] == '.' {
		keyPath = keyPath[:len(keyPath)-1]
	}
	if keyPath[0:2] == ".." {
		keyPath = keyPath[1:]
	}
	return keyPath
}

func (om OrderedMap) toJSONItem(val interface{}, currentPath string) interface{} {
	switch v := val.(type) {
	case []interface{}:
		sliceCopy := make([]interface{}, len(v))
		for i, elm := range v {
			sliceCopy[i] = om.toJSONItem(
				elm,
				fmt.Sprintf("%s[%d]", currentPath, i),
			)
		}
		return sliceCopy

	case map[string]interface{}:
		outJson := orderedJson.New()
		for key, elm := range v {
			outJson.Set(key, om.toJSONItem(
				elm,
				fmt.Sprintf("%s.%s", currentPath, key),
			))
		}
		return outJson

	default:
		return v
	}
}

func (om OrderedMap) toJSON(currentPath string, currentMap map[string]interface{}) (*orderedJson.OrderedMap, error) {
	outJson := orderedJson.New()

	keys, keysExist := om.KeyOrder[currentPath]
	if !keysExist {
		return outJson, fmt.Errorf("Failed to find key order at '%s'", currentPath)
	}
	if len(keys) != len(currentMap) {
		return outJson, fmt.Errorf("Found mismatched map size at %s: %d != %d", currentPath, len(keys), len(currentMap))
	}

	for _, key := range keys {
		outJson.Set(key, om.toJSONItem(
			currentMap[key],
			fmt.Sprintf("%s.%s", currentPath, key),
		))
	}

	return outJson, nil
}

func (om OrderedMap) toMapItem(val interface{}, key string, currentPath string, currentMap map[string]interface{}) (yaml.MapItem, error) {
	mapItem := yaml.MapItem{
		Key: key,
	}
	keyPath := pathJoin(currentPath, key)

	if subMap, isMap := val.(map[string]interface{}); isMap {
		subMS, err := om.toMapSlice(keyPath, subMap)
		if err != nil {
			return mapItem, err
		}
		mapItem.Value = subMS
	} else if subList, isList := val.([]interface{}); isList {
		outList := make([]interface{}, len(subList))
		mapItem.Value = outList

		for i, elm := range subList {
			subSlice, err := om.toMapItem(
				elm,
				"",
				fmt.Sprintf("%s[%d]", keyPath, i),
				currentMap,
			)
			if err != nil {
				return mapItem, err
			}
			outList[i] = subSlice.Value
		}
	} else {
		mapItem.Value = val
	}

	return mapItem, nil
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
		res, err := om.toMapItem(
			currentMap[key],
			key,
			currentPath,
			currentMap,
		)
		if err != nil {
			return mapSlice, err
		}
		mapSlice[i] = res
	}

	return mapSlice, nil
}

func copyValue(anyVal interface{}, orderedMap OrderedMap, keyPath string, currentMap map[string]interface{}) (interface{}, error) {
	switch value := anyVal.(type) {
	case int:
		return value, nil
	case float64:
		return value, nil
	case bool:
		return value, nil
	case string:
		return value, nil

	case []interface{}:
		nextSlice := make([]interface{}, len(value))
		for i, elm := range value {
			res, err := copyValue(
				elm,
				orderedMap,
				fmt.Sprintf("%s[%d]", keyPath, i),
				currentMap,
			)
			if err != nil {
				return nil, err
			}
			nextSlice[i] = res
		}
		return nextSlice, nil

	case yaml.MapSlice:
		nextMap := make(map[string]interface{}, len(value))
		return nextMap, mapSliceToOrderedMap(
			value,
			orderedMap,
			keyPath,
			nextMap,
		)
	}

	return nil, fmt.Errorf("Unrecognized value of type %T at %s: %#v", anyVal, keyPath, anyVal)
}

func copyJSONValue(anyVal interface{}, orderedMap OrderedMap, keyPath string, currentMap map[string]interface{}) (interface{}, error) {
	switch value := anyVal.(type) {
	case int:
		return value, nil
	case float64:
		return value, nil
	case bool:
		return value, nil
	case string:
		return value, nil

	case []interface{}:
		nextSlice := make([]interface{}, len(value))
		for i, elm := range value {
			res, err := copyValue(
				elm,
				orderedMap,
				fmt.Sprintf("%s[%d]", keyPath, i),
				currentMap,
			)
			if err != nil {
				return nil, err
			}
			nextSlice[i] = res
		}
		return nextSlice, nil

	case *orderedJson.OrderedMap:
		nextMap := make(map[string]interface{}, len(value.Keys()))
		return nextMap, jsonToOrderedMap(
			value,
			orderedMap,
			keyPath,
			nextMap,
		)

	case orderedJson.OrderedMap:
		nextMap := make(map[string]interface{}, len(value.Keys()))
		return nextMap, jsonToOrderedMap(
			&value,
			orderedMap,
			keyPath,
			nextMap,
		)
	}

	return nil, fmt.Errorf("Unrecognized value of type %T at %s: %#v", anyVal, keyPath, anyVal)
}

func mapSliceToOrderedMap(mapSlice yaml.MapSlice, orderedMap OrderedMap, currentPath string, currentMap map[string]interface{}) error {
	for _, entry := range mapSlice {
		switch key := entry.Key.(type) {
		case string:
			keyPath := pathJoin(currentPath, key)
			orderedMap.addKey(currentPath, key)

			res, err := copyValue(
				entry.Value,
				orderedMap,
				keyPath,
				currentMap,
			)
			if err != nil {
				return err
			}
			currentMap[key] = res

		case nil:
			// skip nil keys

		default:
			return fmt.Errorf("Unexpected key of type %T at %s: %#v", key, currentPath, entry)
		}
	}

	return nil
}

func jsonToOrderedMap(om *orderedJson.OrderedMap, orderedMap OrderedMap, currentPath string, currentMap map[string]interface{}) error {
	for _, key := range om.Keys() {
		keyPath := pathJoin(currentPath, key)
		orderedMap.addKey(currentPath, key)
		value, _ := om.Get(key)

		res, err := copyJSONValue(
			value,
			orderedMap,
			keyPath,
			currentMap,
		)
		if err != nil {
			return err
		}
		currentMap[key] = res
	}

	return nil
}

var (
	supportedFormats = map[string]func(io.Reader) (OrderedMap, error){
		"json": func(reader io.Reader) (OrderedMap, error) {
			doc := OrderedMap{
				KeyOrder: make(map[string][]string, 0),
				Values:   make(map[string]interface{}, 0),
			}
			om := orderedJson.New()

			buffer, err := ioutil.ReadAll(reader)
			if err != nil {
				return doc, err
			}

			if err := json.Unmarshal(buffer, &om); err != nil {
				return doc, err
			}

			return doc, jsonToOrderedMap(om, doc, ".", doc.Values)
		},

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
