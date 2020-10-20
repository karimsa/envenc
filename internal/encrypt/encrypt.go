package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
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
)

func initCipher(passphrase, salt []byte) (cipher.Block, error) {
	key := argon2.Key(
		passphrase,
		salt,
		3,
		32*1024,
		4,
		32,
	)
	return aes.NewCipher(key)
}

type SimpleSymmetricCipher struct {
	pass []byte
}

func NewSymmetricCipher(pass []byte) SimpleSymmetricCipher {
	return SimpleSymmetricCipher{
		pass: pass,
	}
}

func (s *SimpleSymmetricCipher) Encrypt(str string) (string, error) {
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	block, err := initCipher(s.pass, salt)
	if err != nil {
		return "", err
	}

	raw := pkcs7Pad([]byte(str), aes.BlockSize)
	encrypted := make([]byte, aes.BlockSize+len(raw))
	iv := encrypted[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	cbc := cipher.NewCBCEncrypter(block, iv)
	cbc.CryptBlocks(encrypted[aes.BlockSize:], raw)
	return hex.EncodeToString(
		append(encrypted, salt...),
	), nil
}

func (s *SimpleSymmetricCipher) Decrypt(encrypted string) (string, error) {
	buffer, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	salt := buffer[len(buffer)-saltLength:]
	buffer = buffer[:len(buffer)-saltLength]

	block, err := initCipher(s.pass, salt)
	if err != nil {
		return "", err
	}

	text := make([]byte, len(buffer)-aes.BlockSize)
	cbc := cipher.NewCBCDecrypter(block, buffer[:aes.BlockSize])
	cbc.CryptBlocks(text, buffer[aes.BlockSize:])
	return string(pkcs7Unpad(text)), nil
}

func pkcs7Pad(data []byte, blkSize int) []byte {
	padSize := blkSize - ((len(data) + 1) % blkSize)
	result := make([]byte, len(data)+padSize+1)
	copy(result[:len(data)+1], data)
	result[len(result)-1] = byte(padSize)
	return result
}

func pkcs7Unpad(padded []byte) []byte {
	fmt.Printf("debug: %s\n", padded)
	padSize := 1 + int(padded[len(padded)-1])
	return padded[:len(padded)-padSize]
}
