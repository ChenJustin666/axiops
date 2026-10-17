package secretbox

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	box, err := New("operator-secret")
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, err := box.Encrypt(`{"token":"sensitive"}`)
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == `{"token":"sensitive"}` || len(ciphertext) <= len("v1:") {
		t.Fatal("plaintext was returned instead of ciphertext")
	}
	plaintext, err := box.Decrypt(ciphertext)
	if err != nil || plaintext != `{"token":"sensitive"}` {
		t.Fatalf("Decrypt() plaintext=%q error=%v", plaintext, err)
	}
}

func TestCiphertextIsRandomAndTamperResistant(t *testing.T) {
	box, _ := New("operator-secret")
	one, _ := box.Encrypt("same")
	two, _ := box.Encrypt("same")
	if one == two {
		t.Fatal("encrypting the same value must use a fresh nonce")
	}
	mutatedBytes := []byte(one)
	mutationIndex := len(prefix) + 2
	if mutatedBytes[mutationIndex] == 'A' {
		mutatedBytes[mutationIndex] = 'B'
	} else {
		mutatedBytes[mutationIndex] = 'A'
	}
	mutated := string(mutatedBytes)
	if _, err := box.Decrypt(mutated); err == nil {
		t.Fatal("tampered ciphertext was accepted")
	}
}

func TestKeyIsRequired(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("empty operator secret should be rejected")
	}
}
