package envenc

import (
	// "crypto/aes"
	// "crypto/cipher"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v2"
)

// ...

// type EncryptOptions struct {
// 	// EncryptedData should be the last saved encrypted version of
// 	// this config file
// 	EncryptedData map[string]string

// 	// PlainDataRaw is the plaintext key->value map, without changes made
// 	// (it should be the raw unen)
// 	PlainDataRaw map[string]string

// 	// PlainDataUpdated is the plaintext key->value map, with changes made
// 	// from edits
// 	PlainDataUpdated map[string]string

// 	// Cipher is used to perform encryption/decryption on individual values
// 	Cipher cipher.Block
// }

// func encrypt(options EncryptOptions) (error, map[string]string) {
// 	result := make(map[string]string, len(options.PlainData))

// 	for key, value := range options.PlainDataUpdated {
// 		encValue, encKeyExists := options.EncryptedData[key]
// 		oldValue, oldValueExists := options.PlainDataRaw[key]

// 		if encKeyExists != oldValueExists {
// 			if encKeyExists {
// 				return fmt.Errorf("%s does not exist in the plain unedited map, but exists in the encrypted map", key), nil
// 			} else {
// 				return fmt.Errorf("%s does not exist in the encrypted map, but exists in the plain unedited map", key), nil
// 			}
// 		}

// 		if !oldValueExists || oldValue == value {
// 			result[key] = value
// 		} else {
// 			// ...

// 			results[key] = ""
// 		}
// 	}

// 	return nil, result
// }

type NewEnvOptions struct {
	Format string
	Data   []byte
}

type EnvFile struct {
	rawValues    map[string]interface{}
	updatedPaths [][]string
}

func parseEnvFile(format string, data []byte) (map[string]interface{}, error) {
	var values map[string]interface{}

	switch format {
	case "yaml":
		return values, yaml.Unmarshal(data, &values)
	case "json":
		return values, json.Unmarshal(data, &values)
	case ".env":
		return values, parseDotEnv(data, &values)
	}

	return values, fmt.Errorf("Unrecognized env file format: %s", format)
}

func New(options NewEnvOptions) (*EnvFile, error) {
	rawValues, err := parseEnvFile(options.Format, options.Data)
	if err != nil {
		return nil, err
	}

	return &EnvFile{
		rawValues:    rawValues,
		updatedPaths: make([][]string, 0),
	}, nil
}

type simpleCipher interface {
	Encrypt(raw string) (string, error)
	Decrypt(encrypted string) (string, error)
}

func encryptPaths(input, output map[string]interface{}, currentPath string, paths map[string]bool, sc simpleCipher) error {
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

func (env *EnvFile) Export(format string) ([]byte, error) {
	return nil, fmt.Errorf("fah blah: %s", format)
}
