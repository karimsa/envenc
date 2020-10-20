package encrypt

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestPadding(t *testing.T) {
	data := []byte{1, 1, 1}
	paddedBuff := pkcs7Pad(data, 16)
	padded := hex.EncodeToString(paddedBuff)
	padSize := 12
	padHex := strconv.FormatInt(int64(padSize), 16)
	if len(padHex) == 1 {
		padHex = "0" + padHex
	}

	expected := "010101" + strings.Repeat("00", padSize) + padHex
	if padded != expected {
		t.Error(fmt.Sprintf("Bad padding - got %s, expected %s", padded, expected))
		return
	}

	unpadded := pkcs7Unpad(paddedBuff)
	if !bytes.Equal(unpadded, data) {
		t.Error(fmt.Sprintf("Failed to unpad: %#v(%d) (expected: %#v(%d))", unpadded, len(unpadded), data, len(data)))
		return
	}
}

func TestSymmetricEncrypt(t *testing.T) {
	data := "foobar - some hello world text blah blah"
	cipher, err := NewSymmetricCipher([]byte("testing"))
	if err != nil {
		t.Error(err)
		return
	}

	encrypted, err := cipher.Encrypt(data)
	if err != nil {
		t.Error(err)
		return
	}

	if encrypted == hex.EncodeToString([]byte(data)) {
		t.Error(fmt.Sprintf("Data not encrypted: %s", encrypted))
		return
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Error(err)
		return
	}

	if decrypted != data {
		t.Error(fmt.Sprintf("Decryption failed: '%s' (%d)", decrypted, len(decrypted)))
		return
	}
}
