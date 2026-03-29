package crypto

// Vault provides encryption and decryption for data at rest.
// A nil Vault means no encryption — callers should check before using.
type Vault struct {
	dek []byte // data encryption key (32 bytes, never leaves memory)
}

// Encrypt encrypts plaintext using the vault's DEK.
// Returns the raw ciphertext (nonce + AES-256-GCM output).
// The caller is responsible for adding file headers/slots.
func (v *Vault) Encrypt(plaintext []byte) ([]byte, error) {
	return aesGCMEncrypt(v.dek, plaintext)
}

// Decrypt decrypts ciphertext that was produced by Encrypt.
// Input is nonce + AES-256-GCM ciphertext+tag.
func (v *Vault) Decrypt(ciphertext []byte) ([]byte, error) {
	return aesGCMDecrypt(v.dek, ciphertext)
}

// DEK returns a copy of the data encryption key (for creating new key slots).
func (v *Vault) DEK() []byte {
	out := make([]byte, len(v.dek))
	copy(out, v.dek)
	return out
}
