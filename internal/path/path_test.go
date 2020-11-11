package path

import (
	"io"
	"fmt"
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
}
