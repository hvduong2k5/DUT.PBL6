package persistence

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
)

var ErrCipher = errors.New("INVALID_CORE_CIPHERTEXT")

// Cipher uses an explicit key ID and purpose-bound AEAD envelope. Key material
// is copied into cipher instances, never stored in SQL or returned by this API.
type Cipher struct {
	current string
	keys    map[string]cipher.AEAD
}

func NewCipher(current string, keys map[string][]byte) (*Cipher, error) {
	c := &Cipher{current: current, keys: make(map[string]cipher.AEAD)}
	for id, key := range keys {
		if len(id) == 0 || len(id) > 255 || len(key) != 32 {
			return nil, ErrCipher
		}
		b, err := aes.NewCipher(key)
		if err != nil {
			return nil, ErrCipher
		}
		a, err := cipher.NewGCM(b)
		if err != nil {
			return nil, ErrCipher
		}
		c.keys[id] = a
	}
	if c.keys[current] == nil {
		return nil, ErrCipher
	}
	return c, nil
}

func (c *Cipher) Seal(purpose string, data []byte) ([]byte, error) {
	if c == nil || c.keys[c.current] == nil || purpose == "" || len(data) > 1<<20 {
		return nil, ErrCipher
	}
	a := c.keys[c.current]
	header := make([]byte, 3+len(c.current))
	header[0] = 1
	binary.BigEndian.PutUint16(header[1:3], uint16(len(c.current)))
	copy(header[3:], c.current)
	nonce := make([]byte, a.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	aad := append(append([]byte("order-core-1:"+purpose+":"), header...), nonce...)
	return a.Seal(append(header, nonce...), nonce, data, aad), nil
}

func (c *Cipher) Open(purpose string, data []byte) ([]byte, error) {
	if c == nil || purpose == "" || len(data) < 3 || len(data) > 1<<20+1024 || data[0] != 1 {
		return nil, ErrCipher
	}
	n := int(binary.BigEndian.Uint16(data[1:3]))
	if n < 1 || n > 255 || len(data) < 3+n {
		return nil, ErrCipher
	}
	a := c.keys[string(data[3:3+n])]
	if a == nil {
		return nil, ErrCipher
	}
	head := 3 + n
	end := head + a.NonceSize()
	if len(data) < end+a.Overhead() {
		return nil, ErrCipher
	}
	aad := append(append([]byte("order-core-1:"+purpose+":"), data[:head]...), data[head:end]...)
	out, err := a.Open(nil, data[head:end], data[end:], aad)
	if err != nil {
		return nil, ErrCipher
	}
	return out, nil
}
