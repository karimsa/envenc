package secrets

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/karimsa/secrets/internal/logger"
	pathReader "github.com/karimsa/secrets/internal/path"
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
	if len(str) < 9 {
		return "", fmt.Errorf("Cannot decrypt using randCipher: %s", str)
	}
	return str[9:], nil
}

type badCipher struct{}

func (badCipher) Encrypt(str string) (string, error) {
	return fmt.Sprintf("encrypt(%s)", str), nil
}
func (badCipher) Decrypt(str string) (string, error) {
	str = str[len("encrypt("):]
	str = str[:len(str)-1]
	return str, nil
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

	env, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader([]byte{}),
			Cipher: &randCipher{},
			SecurePaths: []string{
				".top",
			},
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	_, err = env.encryptOrDecryptPaths(
		input,
		pathReader.Path{},
		func(_ pathReader.Path, val string) (string, error) {
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
			SecurePaths: []string{
				".top",
			},
			LogLevel: logger.LevelDebug,
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	output := make(map[string]interface{})
	res, err := env.encryptOrDecryptPaths(
		input,
		pathReader.Path{},
		func(_ pathReader.Path, val string) (string, error) {
			return (&randCipher{}).Encrypt(val)
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}
	output = res.(map[string]interface{})

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

	if str, err := (&randCipher{}).Decrypt(values["top"].(string)); err != nil {
		t.Error(err)
		return
	} else if str != "level" {
		t.Error(fmt.Sprintf("Incorrectly encrypted: %s", data))
		return
	}
}

func TestNewFromYAML(t *testing.T) {
	handler, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader([]byte("hello: world\na: test")),
			Cipher: &randCipher{},
			SecurePaths: []string{
				".hello",
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
			SecurePaths: []string{
				".hello",
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
	data, err = handler.exportWithMapper("yaml", func(_ pathReader.Path, val string) (string, error) {
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
			SecurePaths: []string{
				".hello",
				".a",
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
			SecurePaths: []string{
				".hello",
				".a",
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

func assertSliceEqual(t *testing.T, left []string, right []string) string {
	if strings.Join(left, "\n") != strings.Join(right, "\n") {
		diff := make([]string, 0, len(left)+len(right))
		for i := range left {
			if i < len(right) {
				if left[i] != right[i] {
					diff = append(diff, fmt.Sprintf("- %s", left[i]), fmt.Sprintf("+ %s", right[i]))
				} else {
					diff = append(diff, left[i])
				}
			} else {
				diff = append(diff, fmt.Sprintf("- %s", left[i]))
			}
		}

		return strings.Join(diff, "\n")
	}
	return ""
}

func TestNestedYAML(t *testing.T) {
	configStr := strings.Join([]string{
		"kind: List",
		"spec:",
		"- kind: ConfigMap",
		"  data:",
		"    HELLO: world",
		"    TEST: foobar",
		"    .key.with.dots.single.quote: floof",
		"    .key.with.dots.double.quote: fluffernutter",
	}, "\n")
	handler, err := New(
		NewEnvOptions{
			Format: "yaml",
			Reader: strings.NewReader(configStr),
			Cipher: badCipher{},
			SecurePaths: []string{
				".spec[0].data.HELLO",
				".spec[0].data['.key.with.dots.single.quote']",
				".spec[0].data['.key.with.dots.double.quote']",
			},
			LogLevel: logger.LevelDebug,
		},
	)
	if err != nil {
		t.Error(err.Error())
		return
	}

	data, err := handler.Export("yaml")
	if err != nil {
		t.Error(err)
		return
	}

	var values map[string]interface{}
	err = yaml.Unmarshal(data, &values)
	if err != nil {
		t.Error(err.Error())
		return
	}

	if diff := assertSliceEqual(t, strings.Split(string(data), "\n"), []string{
		"kind: List",
		"spec:",
		"- kind: ConfigMap",
		"  data:",
		"    HELLO: encrypt(world)",
		"    TEST: foobar",
		"    .key.with.dots.single.quote: encrypt(floof)",
		"    .key.with.dots.double.quote: encrypt(fluffernutter)",
		"",
	}); diff != "" {
		t.Error(fmt.Errorf("Incorrectly encrypted output file\n\n%s\n", diff))
		return
	}

	// Re-open/decrypt
	handler, err = Open(
		OpenEnvOptions{
			Format: "yaml",
			Reader: bytes.NewReader(data),
			Cipher: &badCipher{},
			SecurePaths: []string{
				".spec[0].data.HELLO",
				".spec[0].data['.key.with.dots.single.quote']",
				".spec[0].data['.key.with.dots.double.quote']",
			},
			LogLevel: logger.LevelDebug,
		},
	)
	if err != nil {
		t.Error(err)
		return
	}

	data, err = handler.UnsafeRawExport("yaml")
	if err != nil {
		t.Error(err)
		return
	}

	if diff := assertSliceEqual(t, strings.Split(string(data), "\n"), []string{
		"kind: List",
		"spec:",
		"- kind: ConfigMap",
		"  data:",
		"    HELLO: world",
		"    TEST: foobar",
		"    .key.with.dots.single.quote: floof",
		"    .key.with.dots.double.quote: fluffernutter",
		"",
	}); diff != "" {
		t.Error(fmt.Errorf("Incorrectly decrypted file\n\n%s\n", diff))
		return
	}
}
