package secrets

import (
	"fmt"
	"io"
	"os"

	"github.com/karimsa/secrets/internal/logger"
	"github.com/karimsa/secrets/internal/orderedmap"
)

type SimpleCipher interface {
	Encrypt(raw string) (string, error)
	Decrypt(encrypted string) (string, error)
}

type EnvFile struct {
	logger             logger.Logger
	rawValues          orderedmap.OrderedMap
	oldRawValues       map[string]string
	cipher             SimpleCipher
	securePaths        map[string]bool
	lastEncryptedValue map[string]string
}

type NewEnvOptions struct {
	Format      string
	Reader      io.Reader
	Cipher      SimpleCipher
	LogLevel    logger.LogLevel
	SecurePaths map[string]bool
}

func New(options NewEnvOptions) (*EnvFile, error) {
	rawValues, err := orderedmap.Parse(options.Format, options.Reader)
	if err != nil {
		return nil, err
	}

	logger := logger.New(options.LogLevel)
	logger.Debugf("Log level set to %d\n", options.LogLevel)

	return &EnvFile{
		logger:             logger,
		rawValues:          rawValues,
		oldRawValues:       map[string]string{},
		cipher:             options.Cipher,
		securePaths:        options.SecurePaths,
		lastEncryptedValue: map[string]string{},
	}, nil
}

type OpenEnvOptions struct {
	Format      string
	Reader      io.Reader
	Cipher      SimpleCipher
	SecurePaths map[string]bool
	LogLevel    logger.LogLevel
}

func Open(options OpenEnvOptions) (*EnvFile, error) {
	encryptedValues, err := orderedmap.Parse(options.Format, options.Reader)
	if err != nil {
		return nil, err
	}

	env := &EnvFile{
		logger: logger.New(options.LogLevel),
		rawValues: orderedmap.OrderedMap{
			KeyOrder: encryptedValues.KeyOrder,
			Values:   make(map[string]interface{}, len(encryptedValues.KeyOrder["."])),
		},
		oldRawValues:       map[string]string{},
		cipher:             options.Cipher,
		securePaths:        options.SecurePaths,
		lastEncryptedValue: map[string]string{},
	}

	// Populate lastEncryptedValue
	for path, _ := range env.securePaths {
		val, err := readPath(
			path,
			"",
			encryptedValues.Values,
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to initialize lastEncryptedValues: %s", err)
		}
		env.lastEncryptedValue[path] = val
	}

	err = env.encryptOrDecryptPaths(
		encryptedValues.Values,
		env.rawValues.Values,
		"",
		func(path, encrypted string) (string, error) {
			dec, err := options.Cipher.Decrypt(encrypted)
			if err != nil {
				return "", err
			}

			env.oldRawValues[path] = dec
			return dec, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (env *EnvFile) encryptOrDecryptPaths(input, output map[string]interface{}, currentPath string, mapValue func(string, string) (string, error)) error {
	for key, value := range input {
		keyPath := currentPath + "." + key
		strVal, isStr := value.(string)

		if isStr {
			if _, ok := env.securePaths[keyPath]; ok {
				encrypted, err := mapValue(keyPath, strVal)
				if err != nil {
					return err
				}
				output[key] = encrypted
			} else {
				env.logger.Debugf("Skipping encrypt at: %s", keyPath)
				output[key] = strVal
			}
		} else {
			switch v := value.(type) {
			case float64:
				env.logger.Debugf("Copying value at: %s", keyPath)
				output[key] = value
			case bool:
				env.logger.Debugf("Copying value at: %s", keyPath)
				output[key] = value

			case map[string]interface{}:
				outputMap := make(map[string]interface{})
				output[key] = outputMap

				err := env.encryptOrDecryptPaths(
					v,
					outputMap,
					keyPath,
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

func readPath(path, currentPath string, values map[string]interface{}) (string, error) {
	for key, val := range values {
		keyPath := currentPath + "." + key
		if keyPath == path {
			strVal, isStr := val.(string)
			if isStr {
				return strVal, nil
			}
			return "", fmt.Errorf("Found %T at %s: %#v", val, path, val)
		} else if subMap, isMap := val.(map[string]interface{}); isMap {
			return readPath(path, keyPath, subMap)
		}
	}
	return "", fmt.Errorf("No value found at: %s", path)
}

func (env *EnvFile) UpdateFrom(format string, reader io.Reader) error {
	updatedValues, err := orderedmap.Parse(format, reader)
	if err != nil {
		return err
	}
	env.rawValues = updatedValues
	return nil
}

func (env *EnvFile) exportWithMapper(format string, mapValue func(string, string) (string, error)) ([]byte, error) {
	encrypted := orderedmap.OrderedMap{
		KeyOrder: env.rawValues.KeyOrder,
		Values:   make(map[string]interface{}),
	}
	err := env.encryptOrDecryptPaths(
		env.rawValues.Values,
		encrypted.Values,
		"",
		mapValue,
	)
	if err != nil {
		return nil, err
	}
	return encrypted.Export(format)
}

func (env *EnvFile) Export(format string) ([]byte, error) {
	return env.exportWithMapper(format, func(path, val string) (string, error) {
		oldVal, ok := env.oldRawValues[path]
		lastEnc, hasEnc := env.lastEncryptedValue[path]

		if ok && hasEnc && val == oldVal {
			env.logger.Debugf("Keeping value at: %s (unchanged)", path)
			return lastEnc, nil
		}

		env.logger.Debugf("Re-encrypting value at: %s (changed)", path)
		return env.cipher.Encrypt(val)
	})
}

func (env *EnvFile) UnsafeRawExport(format string) ([]byte, error) {
	return env.exportWithMapper(format, func(path, val string) (string, error) {
		return val, nil
	})
}

func (env *EnvFile) ExportFile(format, path string, flag int) error {
	buff, err := env.Export(format)
	if err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, flag|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = outFile.Write(buff)
	if err != nil {
		return err
	}

	err = outFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
