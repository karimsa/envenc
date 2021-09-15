package secrets

import (
	"fmt"
	"io"
	"os"

	"github.com/karimsa/secrets/internal/logger"
	"github.com/karimsa/secrets/internal/orderedmap"
	pathReader "github.com/karimsa/secrets/internal/path"
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
	securePaths        []pathReader.Path
	lastEncryptedValue map[string]string
}

type NewEnvOptions struct {
	Format      string
	Reader      io.Reader
	Cipher      SimpleCipher
	LogLevel    logger.LogLevel
	SecurePaths []string
}

func makeSecurePaths(paths []string) ([]pathReader.Path, error) {
	var err error
	parsedPaths := make([]pathReader.Path, len(paths))

	for i, strPath := range paths {
		parsedPaths[i], err = pathReader.New(strPath)
		if err != nil {
			return nil, err
		}
	}

	return parsedPaths, nil
}

func New(options NewEnvOptions) (*EnvFile, error) {
	rawValues, err := orderedmap.Parse(options.Format, options.Reader)
	if err != nil {
		return nil, err
	}

	logger := logger.New(options.LogLevel)
	logger.Debugf("Log level set to %d\n", options.LogLevel)

	securePaths, err := makeSecurePaths(options.SecurePaths)
	if err != nil {
		return nil, err
	}

	return &EnvFile{
		logger:             logger,
		rawValues:          rawValues,
		oldRawValues:       map[string]string{},
		cipher:             options.Cipher,
		securePaths:        securePaths,
		lastEncryptedValue: map[string]string{},
	}, nil
}

type OpenEnvOptions struct {
	Format      string
	Reader      io.Reader
	Cipher      SimpleCipher
	SecurePaths []string
	LogLevel    logger.LogLevel
}

func Open(options OpenEnvOptions) (*EnvFile, error) {
	encryptedValues, err := orderedmap.Parse(options.Format, options.Reader)
	if err != nil {
		return nil, err
	}

	securePaths, err := makeSecurePaths(options.SecurePaths)
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
		securePaths:        securePaths,
		lastEncryptedValue: map[string]string{},
	}

	// Populate lastEncryptedValue
	for _, path := range env.securePaths {
		val, err := path.ReadFrom(encryptedValues.Values)
		if err != nil {
			return nil, fmt.Errorf("Failed to initialize lastEncryptedValues: %s", err)
		}
		env.lastEncryptedValue[path.String()] = val
	}

	res, err := env.encryptOrDecryptPaths(
		encryptedValues.Values,
		pathReader.Path{},
		func(path pathReader.Path, encrypted string) (string, error) {
			dec, err := options.Cipher.Decrypt(encrypted)
			if err != nil {
				return "", err
			}

			env.oldRawValues[path.String()] = dec
			return dec, nil
		},
	)
	if err != nil {
		return nil, err
	}
	env.rawValues.Values = res.(map[string]interface{})

	return env, nil
}

func (env *EnvFile) isSecurePath(compared pathReader.Path) bool {
	for _, path := range env.securePaths {
		if path.Equals(compared) {
			fmt.Printf("%s = %s\n", path, compared)
			return true
		}
	}

	return false
}

func (env *EnvFile) getLastEncryptedValue(path pathReader.Path) (string, bool) {
	for storedPath, value := range env.lastEncryptedValue {
		storedPathParsed, err := pathReader.New(storedPath)
		if err != nil {
			panic(fmt.Errorf("failed to parse existing path: %s", storedPath))
		}

		if path.Equals(storedPathParsed) {
			return value, true
		}
	}
	return "", false
}

func (env *EnvFile) encryptOrDecryptPaths(untypedInput interface{}, currentPath pathReader.Path, mapValue func(pathReader.Path, string) (string, error)) (interface{}, error) {
	switch input := untypedInput.(type) {
	case map[string]interface{}:
		env.logger.Debugf("Copying map at %s\n", currentPath)
		mapCopy := make(map[string]interface{}, len(input))
		for key, val := range input {
			res, err := env.encryptOrDecryptPaths(
				val,
				currentPath.AppendKey(key),
				mapValue,
			)
			if err != nil {
				return nil, err
			}
			mapCopy[key] = res
		}
		return mapCopy, nil

	case []interface{}:
		env.logger.Debugf("Copying slice at %s\n", currentPath)
		sliceCopy := make([]interface{}, len(input))
		for i, elm := range input {
			res, err := env.encryptOrDecryptPaths(
				elm,
				currentPath.AppendIndex(i),
				mapValue,
			)
			if err != nil {
				return nil, err
			}
			sliceCopy[i] = res
		}
		return sliceCopy, nil

	case string:
		if env.isSecurePath(currentPath) {
			env.logger.Debugf("Encrypting value at %s (%s)\n", currentPath, input)
			return mapValue(currentPath, input)
		} else {
			env.logger.Debugf("Skipping key %s, not a secure path: %#v\n", currentPath, env.securePaths)
		}
		return input, nil

	default:
		env.logger.Debugf("Copying value at %s\n", currentPath)
		return input, nil
	}
}

func (env *EnvFile) UpdateFrom(format string, reader io.Reader) error {
	updatedValues, err := orderedmap.Parse(format, reader)
	if err != nil {
		return err
	}
	env.rawValues = updatedValues
	return nil
}

func (env *EnvFile) exportWithMapper(format string, mapValue func(pathReader.Path, string) (string, error)) ([]byte, error) {
	encrypted := orderedmap.OrderedMap{
		KeyOrder: env.rawValues.KeyOrder,
		Values:   make(map[string]interface{}),
	}
	res, err := env.encryptOrDecryptPaths(
		env.rawValues.Values,
		pathReader.Path{},
		mapValue,
	)
	if err != nil {
		return nil, err
	}
	encrypted.Values = res.(map[string]interface{})
	return encrypted.Export(format)
}

func (env *EnvFile) Export(format string) ([]byte, error) {
	return env.exportWithMapper(format, func(path pathReader.Path, val string) (string, error) {
		oldVal, ok := env.oldRawValues[path.String()]
		lastEnc, hasEnc := env.getLastEncryptedValue(path)

		if ok && hasEnc && val == oldVal {
			env.logger.Debugf("Keeping value at: %s (unchanged)", path)
			return lastEnc, nil
		}

		reason := "added"
		if hasEnc {
			reason = "changed"
		}

		env.logger.Debugf("Re-encrypting value at: %s (%s)", path, reason)
		return env.cipher.Encrypt(val)
	})
}

func (env *EnvFile) UnsafeRawExport(format string) ([]byte, error) {
	return env.exportWithMapper(format, func(path pathReader.Path, val string) (string, error) {
		return val, nil
	})
}

func (env *EnvFile) ExportFile(format, path string, flag int) error {
	buff, err := env.Export(format)
	if err != nil {
		return err
	}

	outFile, err := os.OpenFile(path, flag|os.O_TRUNC, 0644)
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
