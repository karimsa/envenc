package parser

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// TODO: Test YAML with multiple documents

func TestParseYAML(t *testing.T) {
	doc, err := Parse("yaml", bytes.NewReader([]byte("# this comment should be preserved\nhello: world\na: test\n")))
	if err != nil {
		t.Error(err)
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
		t.Error(err)
		return
	}

	if string(buff) != "hello: world\na: test\n" {
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
