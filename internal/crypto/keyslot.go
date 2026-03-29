package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
)

// Slot types.
const (
	SlotPassword = 0x01
	SlotKeypair  = 0x02
)

// SlotSize is the fixed size of a serialized key slot (padded to 96 bytes).
const SlotSize = 96

// KeySlot holds the encrypted DEK for one credential.
type KeySlot struct {
	Type          byte   // SlotPassword or SlotKeypair
	SaltOrPubkey  [32]byte // PBKDF2 salt (password) or ephemeral pubkey (keypair)
	Nonce         [12]byte // AES-GCM nonce for DEK encryption
	EncryptedDEK  [48]byte // 32-byte DEK + 16-byte GCM tag
}

// NewPasswordSlot creates a key slot that encrypts the DEK with a password.
func NewPasswordSlot(password string, dek []byte) (*KeySlot, error) {
	salt, err := generateSalt()
	if err != nil {
		return nil, err
	}

	kek := deriveKEKFromPassword(password, salt)
	encrypted, err := aesGCMEncrypt(kek, dek)
	if err != nil {
		return nil, fmt.Errorf("encrypt DEK: %w", err)
	}

	slot := &KeySlot{Type: SlotPassword}
	copy(slot.SaltOrPubkey[:], salt)
	// encrypted = nonce(12) || ciphertext+tag(48)
	copy(slot.Nonce[:], encrypted[:12])
	copy(slot.EncryptedDEK[:], encrypted[12:])
	return slot, nil
}

// NewKeypairSlot creates a key slot that encrypts the DEK to a recipient's
// public key using ephemeral X25519 ECDH.
func NewKeypairSlot(recipientPub *ecdh.PublicKey, dek []byte) (*KeySlot, error) {
	// Generate ephemeral keypair for this slot
	ephemeral, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key: %w", err)
	}

	kek, err := deriveKEKFromKeypair(ephemeral, recipientPub)
	if err != nil {
		return nil, err
	}

	encrypted, err := aesGCMEncrypt(kek, dek)
	if err != nil {
		return nil, fmt.Errorf("encrypt DEK: %w", err)
	}

	slot := &KeySlot{Type: SlotKeypair}
	copy(slot.SaltOrPubkey[:], ephemeral.PublicKey().Bytes())
	copy(slot.Nonce[:], encrypted[:12])
	copy(slot.EncryptedDEK[:], encrypted[12:])
	return slot, nil
}

// DecryptDEK attempts to decrypt the DEK from this slot using a password.
func (s *KeySlot) DecryptDEKWithPassword(password string) ([]byte, error) {
	if s.Type != SlotPassword {
		return nil, fmt.Errorf("not a password slot")
	}

	kek := deriveKEKFromPassword(password, s.SaltOrPubkey[:])

	// Reconstruct nonce || ciphertext+tag
	data := make([]byte, 12+48)
	copy(data[:12], s.Nonce[:])
	copy(data[12:], s.EncryptedDEK[:])

	return aesGCMDecrypt(kek, data)
}

// DecryptDEKWithPrivateKey attempts to decrypt the DEK using a private key.
func (s *KeySlot) DecryptDEKWithPrivateKey(privKey *ecdh.PrivateKey) ([]byte, error) {
	if s.Type != SlotKeypair {
		return nil, fmt.Errorf("not a keypair slot")
	}

	ephPub, err := UnmarshalPublicKey(s.SaltOrPubkey[:])
	if err != nil {
		return nil, err
	}

	kek, err := deriveKEKForDecrypt(privKey, ephPub)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 12+48)
	copy(data[:12], s.Nonce[:])
	copy(data[12:], s.EncryptedDEK[:])

	return aesGCMDecrypt(kek, data)
}

// Marshal serializes a key slot to exactly SlotSize bytes.
func (s *KeySlot) Marshal() []byte {
	buf := make([]byte, SlotSize)
	buf[0] = s.Type
	copy(buf[1:33], s.SaltOrPubkey[:])
	copy(buf[33:45], s.Nonce[:])
	copy(buf[45:93], s.EncryptedDEK[:])
	// bytes 93-95 are padding (zeros)
	return buf
}

// UnmarshalKeySlot deserializes a key slot from SlotSize bytes.
func UnmarshalKeySlot(data []byte) (*KeySlot, error) {
	if len(data) < SlotSize {
		return nil, fmt.Errorf("key slot too short: %d bytes", len(data))
	}

	slot := &KeySlot{Type: data[0]}
	if slot.Type != SlotPassword && slot.Type != SlotKeypair {
		return nil, fmt.Errorf("unknown slot type: 0x%02x", slot.Type)
	}
	copy(slot.SaltOrPubkey[:], data[1:33])
	copy(slot.Nonce[:], data[33:45])
	copy(slot.EncryptedDEK[:], data[45:93])
	return slot, nil
}
