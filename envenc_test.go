package envenc

import (
	"fmt"
	"testing"
	"encoding/json"

	"gopkg.in/yaml.v2"
)

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

	if strData != `{"nested":{"a":{"b":"stuff"},"c":"d"},"top":"(encrypted)"}` {
		t.Error(fmt.Sprintf("Incorrectly encrypted: %s", strData))
	}
}

func TestNewFromYAML(t *testing.T) {
	handler, err := New(
		NewEnvOptions{
			Format: "yaml",
			Data: []byte("hello: world\nthis: is\na: test"),
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

	var initMap map[string]interface{}
	err = yaml.Unmarshal(data, initMap)
	if err != nil {
		t.Error(err.Error())
		return
	}

	fmt.Printf("init: %#v\n", initMap)
}
