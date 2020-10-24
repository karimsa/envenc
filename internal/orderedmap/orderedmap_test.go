package orderedmap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TODO: Test YAML with multiple documents

func TestParseYAML(t *testing.T) {
	doc, err := Parse("yaml", bytes.NewReader([]byte("# this comment should be preserved\nhello: world\na: test\n")))
	if err != nil {
		t.Error(fmt.Errorf("%s: %#v", err, doc))
		return
	}

	if doc.Values["hello"].(string) != "world" {
		t.Error(fmt.Errorf("YAML parsed incorrectly: %#v", doc))
		return
	}

	if strings.Join(doc.KeyOrder["."], ",") != "hello,a" {
		t.Error(fmt.Errorf("Failed to preserve key order: %#v", doc.KeyOrder))
		return
	}

	buff, err := doc.Export("yaml")
	if err != nil {
		t.Error(fmt.Errorf("%s: %#v", err, doc))
		return
	}

	if string(buff) != "hello: world\na: test\n" {
		t.Error(fmt.Errorf("Failed to preserve key order on export:\n%s", buff))
		return
	}
}

func TestParseNestedYAML(t *testing.T) {
	configStr := strings.Join([]string{
		"kind: List",
		"spec:",
		"- kind: ConfigMap",
		"  data:",
		"    HELLO: world",
		"    TEST: stuff",
		"",
	}, "\n")
	doc, err := Parse("yaml", strings.NewReader(configStr))
	if err != nil {
		t.Error(fmt.Errorf("%s: %#v", err, doc))
		return
	}

	if data, err := json.Marshal(doc.Values); err != nil {
		t.Error(fmt.Errorf("Failed to marshal doc to json: %s", err))
	} else if string(data) != `{"kind":"List","spec":[{"data":{"HELLO":"world","TEST":"stuff"},"kind":"ConfigMap"}]}` {
		t.Error(fmt.Errorf("\nYAML parsed incorrectly:\n\nDoc:\n\t%#v\n\nJSON:\n\t%s\n", doc, data))
		return
	}

	if data, err := json.Marshal(doc.KeyOrder); err != nil {
		t.Error(fmt.Errorf("Failed to marshal doc to json: %s", err))
	} else if string(data) != `{".":["kind","spec"],".spec[0]":["kind","data"],".spec[0].data":["HELLO","TEST"]}` {
		t.Error(fmt.Errorf("\nYAML parsed incorrectly:\n\nKeyOrder:\n\t%#v\n\nJSON:\n\t%s\n", doc, data))
		return
	}

	buff, err := doc.Export("yaml")
	if err != nil {
		t.Error(fmt.Errorf("%s: %#v", err, doc))
		return
	}

	if string(buff) != configStr {
		t.Error(fmt.Errorf("Failed to preserve key order on export:\n%s", buff))
		return
	}
}

func TestParseDotenv(t *testing.T) {
	doc, err := Parse("dotenv", bytes.NewReader([]byte("# this comment should be preserved\nhello=world\na=test\n")))
	if err != nil {
		t.Error(err)
		return
	}

	if doc.Values["hello"].(string) != "world" {
		t.Error(fmt.Errorf("YAML parsed incorrectly: %#v", doc))
		return
	}

	if strings.Join(doc.KeyOrder["."], ",") != "hello,a" {
		t.Error(fmt.Errorf("Failed to preserve key order: %#v", doc))
		return
	}

	buff, err := doc.Export("dotenv")
	if err != nil {
		t.Error(err)
		return
	}

	if string(buff) != "hello=world\na=test\n" {
		t.Error(fmt.Errorf("Failed to preserve comments:\n%s", buff))
		return
	}
}
