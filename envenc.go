package envenc

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
	"github.com/karimsa/envenc/dotenv"
)

func parseEnvFile(format string, data []byte) (map[string]interface{}, error) {
	var values map[string]interface{}

	switch format {
	case "yaml":
		return values, yaml.Unmarshal(data, &values)
	case "json":
		return values, json.Unmarshal(data, &values)
	case ".env":
		return values, dotenv.Unmarshal(data, &values)
	}

	return values, fmt.Errorf("Unrecognized env file format: %s", format)
}

func exportEnvFile(format string, values map[string]interface{}) ([]byte, error) {
	switch format {
	case "yaml":
		return yaml.Marshal(values)
	case "json":
		return json.Marshal(values)
	case ".env":
		return dotenv.Marshal(values)
	}

	return nil, fmt.Errorf("Unrecognized env file format: %s", format)
}

type SimpleCipher interface {
	Encrypt(raw string) (string, error)
	Decrypt(encrypted string) (string, error)
}

func encryptPaths(input, output map[string]interface{}, currentPath string, paths map[string]bool, sc SimpleCipher) error {
	for key, value := range input {
		keyPath := currentPath + "." + key
		strVal, isStr := value.(string)

		if isStr {
			if _, ok := paths[keyPath]; ok {
				encrypted, err := sc.Encrypt(strVal)
				if err != nil {
					return err
				}
				output[key] = encrypted
			} else {
				fmt.Printf("skipping encrypt for %s (not in %#v)\n", keyPath, paths)
				output[key] = strVal
			}
		} else {
			switch v := value.(type) {
			case float64:
				fmt.Printf("copying value %s\n", keyPath)
				output[key] = value
			case bool:
				fmt.Printf("copying value %s\n", keyPath)
				output[key] = value

			case map[string]interface{}:
				outputMap := make(map[string]interface{})
				output[key] = outputMap

				err := encryptPaths(
					v,
					outputMap,
					keyPath,
					paths,
					sc,
				)
				if err != nil {
					return err
				}

			default:
				return fmt.Errorf("Unexpected %T at path: %s (%#v)", value, keyPath, value)
			}
		}
	}

	return nil
}

type NewEnvOptions struct {
	Format string
	Data   []byte
	Cipher SimpleCipher
}

type EnvFile struct {
	rawValues    map[string]interface{}
	updatedPaths map[string]bool
	cipher SimpleCipher
}

func New(options NewEnvOptions) (*EnvFile, error) {
	rawValues, err := parseEnvFile(options.Format, options.Data)
	if err != nil {
		return nil, err
	}

	return &EnvFile{
		rawValues:    rawValues,
		updatedPaths: make(map[string]bool),
		cipher: options.Cipher,
	}, nil
}

func (env *EnvFile) Touch(path string) {
	env.updatedPaths[path] = true
}

func (env *EnvFile) Set(path, value string) error {
	if path[0] != '.' {
		return fmt.Errorf("Invalid key path: %s", path)
	}

	targetMap := env.rawValues
	pathBits := strings.Split(path, ".")[1:]
	currentPath := "."

	for _, step := range pathBits[:len(pathBits)-1] {
		currentPath += "." + step
		v, ok := targetMap[step]
		if ok {
			nextMap, isMap := v.(map[string]interface{})
			if !isMap {
				return fmt.Errorf("Non-map value (of type %T) found at %s: %#v", v, currentPath, v)
			}
			targetMap = nextMap
		} else {
			nextMap := make(map[string]interface{})
			targetMap[step] = nextMap
			targetMap = nextMap
		}
	}

	targetMap[pathBits[len(pathBits)-1]] = value
	env.Touch(path)
	return nil
}

func (env *EnvFile) Export(format string) ([]byte, error) {
	encrypted := make(map[string]interface{})
	err := encryptPaths(
		env.rawValues,
		encrypted,
		"",
		env.updatedPaths,
		env.cipher,
	)
	if err != nil {
		return nil, err
	}
	return exportEnvFile(format, encrypted)
}
