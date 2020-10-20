package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/argon2"
)

type EncryptionStrategy int

const (
	// StrategySymmetric refers to utilizing symmetric encryption
	// (i.e. a single password to encrypt/decrypt data)
	StrategySymmetric EncryptionStrategy = iota

	// StrategyAsymmetric refers to using an RSA private/public keypair
	// for encryption
	StrategyAsymmetric

	// StrategyKeyring refers to using a keyring with many RSA private/public
	// keypairs
	StrategyKeyring

	saltLength = 16
	hmacLength = 256 / 8
)

func initCipher(passphrase, salt []byte) (cipher.Block, []byte, error) {
	key := argon2.Key(
		passphrase,
		salt,
		3,
		32*1024,
		4,
		32,
	)
	block, err := aes.NewCipher(key)
	return block, key, err
}

type SimpleSymmetricCipher struct {
	pass []byte
}

func NewSymmetricCipher(pass []byte) SimpleSymmetricCipher {
	return SimpleSymmetricCipher{
		pass: pass,
	}
}

func sign(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(data)
	return m.Sum(nil)
}

func verify(key, data, expected []byte) bool {
	return hmac.Equal(sign(key, data), expected)
}

type symmetricEnvelope struct {
	buffer     []byte
	iv         []byte
	salt       []byte
	signature  []byte
	cipherText []byte
}

func newSymmetricEnvelope(dataSize int) symmetricEnvelope {
	buffer := make([]byte, aes.BlockSize+dataSize+saltLength+hmacLength)
	return openSymmetricEnvelope(buffer)
}

func openSymmetricEnvelope(buffer []byte) symmetricEnvelope {
	return symmetricEnvelope{
		buffer: buffer,

		iv:        buffer[0:aes.BlockSize],
		salt:      buffer[aes.BlockSize : aes.BlockSize+saltLength],
		signature: buffer[aes.BlockSize+saltLength : aes.BlockSize+saltLength+hmacLength],

		cipherText: buffer[aes.BlockSize+saltLength+hmacLength:],
	}
}

func (s symmetricEnvelope) export() string {
	return hex.EncodeToString(s.buffer)
}

func (s SimpleSymmetricCipher) Encrypt(str string) (string, error) {
	raw := pkcs7Pad([]byte(str), aes.BlockSize)
	e := newSymmetricEnvelope(len(raw))

	_, err := rand.Read(e.salt)
	if err != nil {
		return "", err
	}

	block, key, err := initCipher(s.pass, e.salt)
	if err != nil {
		return "", err
	}

	if _, err := rand.Read(e.iv); err != nil {
		return "", err
	}

	cbc := cipher.NewCBCEncrypter(block, e.iv)
	cbc.CryptBlocks(e.cipherText, raw)

	signature := sign(key, e.cipherText)
	copy(e.signature, signature)

	return e.export(), nil
}

func (s SimpleSymmetricCipher) Decrypt(encrypted string) (decrypted string, err error) {
	defer func() {
		if r := recover(); err == nil && r != nil {
			if e, isErr := r.(error); isErr {
				err = e
			} else {
				panic(r)
			}
		}
	}()

	buffer, err := hex.DecodeString(encrypted)
	if err != nil {
		return
	}
	e := openSymmetricEnvelope(buffer)

	block, key, err := initCipher(s.pass, e.salt)
	if err != nil {
		return
	}

	if !verify(key, e.cipherText, e.signature) {
		err = fmt.Errorf("Failed to decrypt value")
		return
	}

	text := make([]byte, len(e.cipherText))
	cbc := cipher.NewCBCDecrypter(block, buffer[:aes.BlockSize])
	cbc.CryptBlocks(text, e.cipherText)
	decrypted = string(pkcs7Unpad(text))

	return
}

func pkcs7Pad(data []byte, blkSize int) []byte {
	padSize := blkSize - ((len(data) + 1) % blkSize)
	result := make([]byte, len(data)+padSize+1)
	copy(result[:len(data)+1], data)
	result[len(result)-1] = byte(padSize)
	return result
}

func pkcs7Unpad(padded []byte) []byte {
	padSize := 1 + int(padded[len(padded)-1])
	return padded[:len(padded)-padSize]
}
