package aes

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

func Encrypt(key []byte, text []byte) (string, string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}

	paddingText := pad(text, block.BlockSize())
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", "", err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(paddingText, paddingText)
	return base64.StdEncoding.EncodeToString(iv), base64.StdEncoding.EncodeToString(paddingText), nil

}
func pad(src []byte, blocksize int) []byte {
	pdSize := blocksize - (len(src) % blocksize)
	padBytes := bytes.Repeat([]byte{0x00}, pdSize)
	src = append(src, padBytes...)
	return src
}

func Decrypt(key []byte, iv64 string, data64 string) ([]byte, error) {
	iv, err := base64.StdEncoding.DecodeString(iv64) // iv base64
	if err != nil {
		return []byte{}, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(data64) // encrypt base64
	if err != nil {
		return []byte{}, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}
	if len(ciphertext) < aes.BlockSize {
		return []byte{}, errors.New("ciphertext too short")
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return []byte{}, errors.New("ciphertext is not a multiple of the block size")
	}
	aes := cipher.NewCBCDecrypter(block, iv)
	aes.CryptBlocks(ciphertext, ciphertext)
	return ciphertext, err
}
