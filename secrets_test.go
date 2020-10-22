package secrets

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/karimsa/secrets/internal/logger"
	"gopkg.in/yaml.v2"
)

type randCipher struct{}

func (*randCipher) Encrypt(str string) (string, error) {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + ":" + str, nil
}
func (*randCipher) Decrypt(str string) (string, error) {
	return str[9:], nil
}

func TestDecryptPaths(t *testing.T) {
	text, err := (&randCipher{}).Encrypt("level")
	if err != nil {
		t.Error(err)
		return
	}

	input := map[string]interface{}{
		"top": text,
		"a":   "b",
	}
	output := map[string]interface{}{}

	env, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader([]byte{}),
			Cipher: &randCipher{},
			SecurePaths: map[string]bool{
				".top": true,
			},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	err = env.encryptOrDecryptPaths(
		input,
		output,
		"",
		func(_, val string) (string, error) {
			return (&randCipher{}).Decrypt(val)
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestEncryptPaths(t *testing.T) {
	var input map[string]interface{}
	err := json.Unmarshal([]byte(`{
		"top": "level",
		"nested": {
			"a": {
				"b": "stuff"
			},
			"c": "d",
			"e": 1
		}
	}`), &input)
	if err != nil {
		t.Error(err.Error())
		return
	}

	env, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader([]byte{}),
			Cipher: &randCipher{},
			SecurePaths: map[string]bool{
				".top": true,
			},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	output := make(map[string]interface{})
	err = env.encryptOrDecryptPaths(
		input,
		output,
		"",
		func(_, val string) (string, error) {
			return (&randCipher{}).Encrypt(val)
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Error(err.Error())
		return
	}

	var values map[string]interface{}
	err = json.Unmarshal(data, &values)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if str, _ := (&randCipher{}).Decrypt(values["top"].(string)); str != "level" {
		t.Error(fmt.Sprintf("Incorrectly encrypted: %s", string(data)))
	}
}

func TestNewFromYAML(t *testing.T) {
	handler, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader([]byte("hello: world\na: test")),
			Cipher: &randCipher{},
			SecurePaths: map[string]bool{
				".hello": true,
			},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	data, err := handler.Export("yaml")
	if err != nil {
		t.Error(err.Error())
		return
	}

	var values map[string]interface{}
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if s, _ := (&randCipher{}).Decrypt(values["hello"].(string)); values["a"].(string) != "test" || s != "world" {
		t.Error(fmt.Errorf("Unexpected exported yaml env:\n%s", data))
		return
	}

	handler, err = Open(
		OpenEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader(data),
			Cipher: &randCipher{},
			SecurePaths: map[string]bool{
				".hello": true,
			},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	// Test re-export
	data, err = handler.Export("yaml")
	if err != nil {
		t.Error(err.Error())
		return
	}
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		t.Error(err.Error())
		return
	}
	if s, _ := (&randCipher{}).Decrypt(values["hello"].(string)); values["a"].(string) != "test" || s != "world" {
		t.Error(fmt.Errorf("Unexpected exported yaml env:\n%s", data))
		return
	}

	// Test export raw
	data, err = handler.exportWithMapper("yaml", func(_, val string) (string, error) {
		return val, nil
	})
	if err != nil {
		t.Error(err.Error())
		return
	}
	if "hello: world\na: test\n" != string(data) && "a: test\nhello: world\n" != string(data) {
		t.Error(fmt.Errorf("Unexpected re-exported data:\n%s", data))
		return
	}
}

func TestDiff(t *testing.T) {
	handler, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader([]byte("hello: world\na: test\nb: stuff\n")),
			Cipher: &randCipher{},
			SecurePaths: map[string]bool{
				".hello": true,
				".a":     true,
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	data, err := handler.Export("yaml")
	if err != nil {
		t.Error(err)
		return
	}

	encryptedVals := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &encryptedVals); err != nil {
		t.Error(err)
		return
	}

	// Test diffing
	handler, err = Open(
		OpenEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader(data),
			Cipher: &randCipher{},
			SecurePaths: map[string]bool{
				".hello": true,
				".a":     true,
			},
			LogLevel: logger.LevelDebug,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	err = handler.UpdateFrom("yaml", bytes.NewReader([]byte("hello: not-world\na: test\nb: stuff\n")))
	if err != nil {
		t.Error(err)
		return
	}

	data, err = handler.Export("yaml")
	if err != nil {
		t.Error(err)
		return
	}

	updatedVals := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &updatedVals); err != nil {
		t.Error(err)
		return
	}

	if updatedVals["hello"].(string) == encryptedVals["hello"].(string) {
		t.Error(fmt.Errorf("hello encrypted value was not updated"))
		return
	}
	if updatedVals["a"].(string) != encryptedVals["a"].(string) {
		t.Error(fmt.Errorf("a encrypted value was updated even though it was not changed"))
		return
	}
	if updatedVals["b"].(string) != encryptedVals["b"].(string) {
		t.Error(fmt.Errorf("b value was updated, even though it is insecure"))
		return
	}
}
