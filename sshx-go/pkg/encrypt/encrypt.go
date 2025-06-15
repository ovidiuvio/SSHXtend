// Package encrypt provides encryption utilities using Argon2 + AES-CTR.
package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// Salt used for key derivation - must match the Rust implementation.
const salt = "This is a non-random salt for sshx.io, since we want to stretch the security of 83-bit keys!"

// Encrypt handles stream encryption using Argon2 + AES-CTR.
type Encrypt struct {
	aesKey [16]byte
}

// New creates a new encryptor from a password string.
func New(key string) *Encrypt {
	// Parameters must match the Rust implementation:
	// Argon2id, memory=19*1024, iterations=2, parallelism=1, keyLen=16
	aesKey := argon2.IDKey([]byte(key), []byte(salt), 2, 19*1024, 1, 16)
	
	var keyArray [16]byte
	copy(keyArray[:], aesKey)
	
	return &Encrypt{
		aesKey: keyArray,
	}
}

// Zeros returns the encrypted zero block for client verification.
func (e *Encrypt) Zeros() []byte {
	zeros := make([]byte, 16)
	
	block, err := aes.NewCipher(e.aesKey[:])
	if err != nil {
		panic(fmt.Sprintf("failed to create AES cipher: %v", err))
	}
	
	// Use zero IV for the zero block
	iv := make([]byte, 16)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(zeros, zeros)
	
	return zeros
}

// Segment encrypts a data segment from a stream.
// streamNum must be non-zero for security.
// offset specifies the byte offset within the stream.
func (e *Encrypt) Segment(streamNum uint64, offset uint64, data []byte) []byte {
	if streamNum == 0 {
		panic("stream number must be nonzero")
	}
	
	block, err := aes.NewCipher(e.aesKey[:])
	if err != nil {
		panic(fmt.Sprintf("failed to create AES cipher: %v", err))
	}
	
	// Construct IV: stream number (8 bytes big-endian) + counter offset (8 bytes big-endian)
	// The counter offset is offset / 16 (since AES block size is 16 bytes)
	iv := make([]byte, 16)
	binary.BigEndian.PutUint64(iv[0:8], streamNum)
	binary.BigEndian.PutUint64(iv[8:16], offset/16)
	
	stream := cipher.NewCTR(block, iv)
	
	// Handle partial block offset within the current counter block
	blockOffset := offset % 16
	if blockOffset > 0 {
		// We need to advance within the current block
		skipBuf := make([]byte, blockOffset)
		stream.XORKeyStream(skipBuf, skipBuf)
	}
	
	// Encrypt the actual data
	result := make([]byte, len(data))
	copy(result, data)
	stream.XORKeyStream(result, result)
	
	return result
}