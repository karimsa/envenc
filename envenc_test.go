package envenc

import (
	"encoding/json"
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"
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

	output := make(map[string]interface{})
	err = encryptPaths(
		input,
		output,
		"",
		map[string]bool{
			".top": true,
		},
		&badCipher{},
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
			Data:   []byte("hello: world\nthis: is\na: test"),
			Cipher: &badCipher{},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	// should trigger encryption
	err = handler.Set(".hello", "world")
	if err != nil {
		t.Error(err.Error())
		return
	}

	data, err := handler.Export("yaml")
	if err != nil {
		t.Error(err.Error())
		return
	}

	var initMap map[string]interface{}
	err = yaml.Unmarshal(data, &initMap)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if fmt.Sprintf("%#v", initMap) != `map[string]interface {}{"a":"test", "hello":"encrypt(world)", "this":"is"}` {
		t.Error(fmt.Errorf("Unexpected exported yaml env: %#v", initMap))
		return
	}
}
