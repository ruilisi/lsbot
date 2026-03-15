package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyPairGenerateAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.pem")
	priv1, err := GenerateOrLoadKeyPair(path)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	priv2, err := GenerateOrLoadKeyPair(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if priv1.PublicKey().Equal(priv2.PublicKey()) == false {
		t.Error("loaded key differs from generated key")
	}
}

func TestKeyPairInvalidPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.pem")
	os.WriteFile(path, []byte("not pem"), 0600)
	_, err := GenerateOrLoadKeyPair(path)
	if err == nil {
		t.Error("expected error for invalid PEM")
	}
}

func TestPublicKeyEncoding(t *testing.T) {
	priv, err := GenerateOrLoadKeyPair(filepath.Join(t.TempDir(), "k.pem"))
	if err != nil {
		t.Fatal(err)
	}
	b64 := PublicKeyToBase64(priv.PublicKey())
	pub, err := PublicKeyFromBase64(b64)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !pub.Equal(priv.PublicKey()) {
		t.Error("decoded pubkey differs")
	}
}

func TestDeriveSessionKey(t *testing.T) {
	dir := t.TempDir()
	privA, _ := GenerateOrLoadKeyPair(filepath.Join(dir, "a.pem"))
	privB, _ := GenerateOrLoadKeyPair(filepath.Join(dir, "b.pem"))

	keyA, err := DeriveSessionKey(privA, privB.PublicKey())
	if err != nil {
		t.Fatalf("derive A: %v", err)
	}
	keyB, err := DeriveSessionKey(privB, privA.PublicKey())
	if err != nil {
		t.Fatalf("derive B: %v", err)
	}
	if string(keyA) != string(keyB) {
		t.Error("keys don't match")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	dir := t.TempDir()
	privA, _ := GenerateOrLoadKeyPair(filepath.Join(dir, "a.pem"))
	privB, _ := GenerateOrLoadKeyPair(filepath.Join(dir, "b.pem"))

	keyA, _ := DeriveSessionKey(privA, privB.PublicKey())

	plaintext := "Hello, E2EE world!"
	ciphertext, err := Encrypt(keyA, []byte(plaintext))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	keyB, _ := DeriveSessionKey(privB, privA.PublicKey())
	decrypted, err := Decrypt(keyB, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(decrypted) != plaintext {
		t.Errorf("got %q, want %q", string(decrypted), plaintext)
	}
}

func TestFingerprint(t *testing.T) {
	priv, _ := GenerateOrLoadKeyPair(filepath.Join(t.TempDir(), "k.pem"))
	fp := Fingerprint(priv.PublicKey())
	if len(fp) != len("sha256:")+16 {
		t.Errorf("unexpected fingerprint length: %q", fp)
	}
	if fp[:7] != "sha256:" {
		t.Errorf("unexpected fingerprint prefix: %q", fp)
	}
}
