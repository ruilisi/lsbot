package e2e

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

const hkdfSalt = "lsbot-e2e-v1"

// GenerateOrLoadKeyPair loads a P-256 key pair from path if it exists,
// otherwise generates one, saves it, and returns it.
func GenerateOrLoadKeyPair(path string) (*ecdh.PrivateKey, error) {
	if data, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("e2e: invalid PEM in %s", path)
		}
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("e2e: failed to parse key from %s: %w", path, err)
		}
		switch k := parsed.(type) {
		case *ecdh.PrivateKey:
			return k, nil
		case *ecdsa.PrivateKey:
			priv, err := k.ECDH()
			if err != nil {
				return nil, fmt.Errorf("e2e: ECDSA→ECDH conversion failed: %w", err)
			}
			return priv, nil
		default:
			return nil, fmt.Errorf("e2e: key in %s is not an EC key", path)
		}
	}

	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("e2e: failed to generate key: %w", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("e2e: failed to marshal key: %w", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		return nil, fmt.Errorf("e2e: failed to save key to %s: %w", path, err)
	}

	return priv, nil
}

// LoadKeyPair loads a P-256 private key from path. Returns an error if the file
// does not exist or cannot be parsed (unlike GenerateOrLoadKeyPair, it never generates).
func LoadKeyPair(path string) (*ecdh.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("e2e: cannot read key from %s: %w", path, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("e2e: invalid PEM in %s", path)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("e2e: failed to parse key from %s: %w", path, err)
	}
	switch k := parsed.(type) {
	case *ecdh.PrivateKey:
		return k, nil
	case *ecdsa.PrivateKey:
		priv, err := k.ECDH()
		if err != nil {
			return nil, fmt.Errorf("e2e: ECDSA→ECDH conversion failed: %w", err)
		}
		return priv, nil
	default:
		return nil, fmt.Errorf("e2e: key in %s is not an EC key", path)
	}
}

// PublicKeyToBase64 encodes an ECDH public key as base64 (uncompressed point).
func PublicKeyToBase64(pub *ecdh.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pub.Bytes())
}

// PublicKeyFromBase64 decodes a base64 ECDH P-256 public key.
func PublicKeyFromBase64(s string) (*ecdh.PublicKey, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("e2e: invalid base64 pubkey: %w", err)
	}
	pub, err := ecdh.P256().NewPublicKey(b)
	if err != nil {
		return nil, fmt.Errorf("e2e: invalid P-256 public key: %w", err)
	}
	return pub, nil
}

// DeriveSessionKey performs ECDH and derives a 32-byte AES key via HKDF-SHA256.
func DeriveSessionKey(priv *ecdh.PrivateKey, peerPub *ecdh.PublicKey) ([]byte, error) {
	secret, err := priv.ECDH(peerPub)
	if err != nil {
		return nil, fmt.Errorf("e2e: ECDH failed: %w", err)
	}
	key, err := hkdf.Key(sha256.New, secret, []byte(hkdfSalt), "", 32)
	if err != nil {
		return nil, fmt.Errorf("e2e: HKDF failed: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random 12-byte nonce.
// Returns base64(nonce || ciphertext).
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("e2e: AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("e2e: GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("e2e: nonce: %w", err)
	}
	ct := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt decrypts a base64(nonce || ciphertext) produced by Encrypt.
func Decrypt(key []byte, b64 string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("e2e: base64 decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("e2e: AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("e2e: GCM: %w", err)
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return nil, fmt.Errorf("e2e: ciphertext too short")
	}
	nonce, ct := data[:ns], data[ns:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("e2e: decrypt: %w", err)
	}
	return plain, nil
}

// Fingerprint returns "sha256:<first 16 hex chars of SHA-256(pubkey bytes)>".
func Fingerprint(pub *ecdh.PublicKey) string {
	h := sha256.Sum256(pub.Bytes())
	return "sha256:" + hex.EncodeToString(h[:])[:16]
}
