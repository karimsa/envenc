package envenc

import (
	"encoding/json"
	"fmt"
	"testing"
)

type badCipher struct{}

func (*badCipher) Encrypt(str string) (string, error) {
	return "encrypt(" + str + ")", nil
}
func (*badCipher) Decrypt(str string) (string, error) {
	str = str[len("encrypt("):]
	str = str[:len(str)-1]
	return str, nil
}

func TestDecryptPaths(t *testing.T) {
	text, err := (&badCipher{}).Encrypt("level")
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
			Data:   []byte{},
			Cipher: &badCipher{},
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
		map[string]bool{
			".top": true,
		},
		func(val string) (string, error) {
			return (&badCipher{}).Decrypt(val)
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
			Data:   []byte{},
			Cipher: &badCipher{},
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
		map[string]bool{
			".top": true,
		},
		func(val string) (string, error) {
			return (&badCipher{}).Encrypt(val)
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
	strData := string(data)

	if strData != `{"nested":{"a":{"b":"stuff"},"c":"d","e":1},"top":"encrypt(level)"}` {
		t.Error(fmt.Sprintf("Incorrectly encrypted: %s", strData))
	}
}

func TestNewFromYAML(t *testing.T) {
	handler, err := New(
		NewEnvOptions{
			Format: "yaml",
			Data:   []byte("hello: world\na: test"),
			Cipher: &badCipher{},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	// should trigger encryption
	handler.Touch(".hello")

	data, err := handler.Export("yaml")
	if err != nil {
		t.Error(err.Error())
		return
	}
	if "hello: encrypt(world)\na: test\n" != string(data) && "a: test\nhello: encrypt(world)\n" != string(data) {
		t.Error(fmt.Errorf("Unexpected exported yaml env:\n%s", data))
		return
	}

	handler, err = Open(
		OpenEnvOptions{
			Format: "yaml",
			Data:   data,
			Cipher: &badCipher{},
			SecurePaths: map[string]bool{
				".hello": true,
			},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	handler.Touch(".hello")

	// Test re-export
	data, err = handler.Export("yaml")
	if err != nil {
		t.Error(err.Error())
		return
	}
	if "hello: encrypt(world)\na: test\n" != string(data) && "a: test\nhello: encrypt(world)\n" != string(data) {
		t.Error(fmt.Errorf("Unexpected re-exported data:\n%s", data))
		return
	}

	// Test export raw
	data, err = handler.exportWithMapper("yaml", func(val string) (string, error) {
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
