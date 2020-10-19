package dotenv

import (
	"fmt"
	"strings"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	var val map[string]interface{}
	err := Unmarshal([]byte(strings.Join([]string{
		"A=B",
		"",
		"C=D # part of the value",
		" ",
		"# not a key=value pair",
		" ",
	}, "\n")), &val)
	if err != nil {
		t.Error(err)
		return
	}
	if `map[string]interface {}{"A":"B", "C":"D # part of the value"}` != fmt.Sprintf("%#v", val) {
		t.Error(fmt.Errorf("Unexpected unmarshalled value: %#v", val))
		return
	}
}

func TestMarshal(t *testing.T) {
	var val map[string]interface{}
	data := []byte(strings.Join([]string{
		"A=B",
		"",
		"C=D # part of the value",
		" ",
		"# not a key=value pair",
		" ",
	}, "\n"))
	err := Unmarshal(data, &val)
	if err != nil {
		t.Error(err)
		return
	}

	str, err := Marshal(val)
	if err != nil {
		t.Error(err)
		return
	}

	if string(str) != "A=B\nC=D # part of the value\n" && string(str) != "C=D # part of the value\nA=B\n" {
		t.Error(fmt.Errorf("Unexpected marshalled value:\n'%s'", str))
		return
	}
}
