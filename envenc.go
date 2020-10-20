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

func encryptOrDecryptPaths(input, output map[string]interface{}, currentPath string, paths map[string]bool, mapValue func(string) (string, error)) error {
	for key, value := range input {
		keyPath := currentPath + "." + key
		strVal, isStr := value.(string)

		if isStr {
			if _, ok := paths[keyPath]; ok {
				encrypted, err := mapValue(strVal)
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

				err := encryptOrDecryptPaths(
					v,
					outputMap,
					keyPath,
					paths,
					mapValue,
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

type EnvFile struct {
	rawValues    map[string]interface{}
	updatedPaths map[string]bool
	cipher SimpleCipher
}

type NewEnvOptions struct {
	Format string
	Data   []byte
	Cipher SimpleCipher
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

type OpenEnvOptions struct {
	Format string
	Data   []byte
	Cipher SimpleCipher
	SecurePaths map[string]bool
}

func Open(options OpenEnvOptions) (*EnvFile, error) {
	encryptedValues, err := parseEnvFile(options.Format, options.Data)
	if err != nil {
		return nil, err
	}

	rawValues := make(map[string]interface{})
	err = encryptOrDecryptPaths(
		encryptedValues,
		rawValues,
		"",
		options.SecurePaths,
		func(encrypted string) (string, error) {
			return options.Cipher.Decrypt(encrypted)
		},
	)
	fmt.Printf("encryptedMap = %#v\nraw = %#v\n", encryptedValues, rawValues)
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

func (env *EnvFile) exportWithMapper(format string, mapValue func(string) (string, error)) ([]byte, error) {
	encrypted := make(map[string]interface{})
	err := encryptOrDecryptPaths(
		env.rawValues,
		encrypted,
		"",
		env.updatedPaths,
		mapValue,
	)
	if err != nil {
		return nil, err
	}
	return exportEnvFile(format, encrypted)
}

func (env *EnvFile) Export(format string) ([]byte, error) {
	return env.exportWithMapper(format, func(val string) (string, error) {
		return env.cipher.Encrypt(val)
	})
}
