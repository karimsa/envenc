package path

import (
	"fmt"
	"io"
	"testing"
)

func TestPathParsing(t *testing.T) {
	str := ".spec[0].key.a.b[1].foo"
	expected := []token{
		{
			key: "spec",
		},
		{
			index: 0,
		},
		{
			key: "key",
		},
		{
			key: "a",
		},
		{
			key: "b",
		},
		{
			index: 1,
		},
		{
			key: "foo",
		},
	}

	var tok token
	var err error
	for _, exp := range expected {
		tok, str, err = nextToken(str)
		if err != nil {
			t.Error(err)
			return
		}
		if tok.key != exp.key || tok.index != exp.index {
			t.Error(fmt.Errorf("Unexpected token read: %#v", tok))
			return
		}
	}

	if _, _, err = nextToken(str); err != io.EOF {
		t.Error(fmt.Errorf("Unexpected error: %s (expected EOF)", err))
		return
	}

	if _, err := New(".foo..bar"); err == nil {
		t.Error(fmt.Errorf("Expected syntax error when parsing: .foo..bar"))
		return
	}
}

func TestMapRead(t *testing.T) {
	p, err := New(".spec[0].data")
	if err != nil {
		t.Error(err)
		return
	}

	v, err := p.ReadFrom(map[string]interface{}{
		"spec": []interface{}{
			map[string]interface{}{
				"data": "testing",
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	if v != "testing" {
		t.Error(fmt.Errorf("Wrong value read from map: %s", v))
		return
	}
}

func TestPathParseStringKeyDoubleQuote(t *testing.T) {
	p, err := New(".test[\".nested.key\"]")
	if err != nil {
		t.Error(err)
		return
	}
	v, err := p.ReadFrom(map[string]interface{}{
		"test": map[string]interface{}{
			".nested.key": "testing",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	if v != "testing" {
		t.Error(fmt.Errorf("Wrong value read from map: %s", v))
		return
	}
}

func TestPathParseStringKeySingleQuote(t *testing.T) {
	p, err := New(".test['.nested.key']")
	if err != nil {
		t.Error(err)
		return
	}
	v, err := p.ReadFrom(map[string]interface{}{
		"test": map[string]interface{}{
			".nested.key": "testing",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	if v != "testing" {
		t.Error(fmt.Errorf("Wrong value read from map: %s", v))
		return
	}
}

func TestPathCompare(t *testing.T) {
	p, err := New(".obj.nested['key']")
	if err != nil {
		t.Error(fmt.Errorf("Failed to parse testpath: %s", err))
		return
	}

	equal := []string{
		".obj.nested.key",
		".['obj'].nested.key",
		".obj['nested'].key",
	}
	for _, str := range equal {
		compared, err := New(str)
		if err != nil {
			t.Error(fmt.Errorf("Failed to parse testpath '%s': %s", str, err))
			return
		}
		if !p.Equals(compared) {
			t.Error(fmt.Errorf("Failed to compare path: %s (should be equal)\n\npath => %#v\ncompared => %#v", str, p, compared))
			return
		}
	}

	notEqual := []string{
		".obj.nested",
		".['obj'].nested",
		".obj['nested']",
	}
	for _, str := range notEqual {
		compared, err := New(str)
		if err != nil {
			t.Error(fmt.Errorf("Failed to parse testpath '%s': %s", str, err))
			return
		}
		if p.Equals(compared) {
			t.Error(fmt.Errorf("Failed to compare path: %s (should not be equal)\n\npath => %#v\ncompared => %#v", str, p, compared))
			return
		}
	}
}
