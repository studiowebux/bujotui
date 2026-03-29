package crypto

import (
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

// Keypair holds an X25519 key pair for asymmetric encryption.
type Keypair struct {
	Private *ecdh.PrivateKey
	Public  *ecdh.PublicKey
}

// GenerateKeypair creates a new random X25519 key pair.
func GenerateKeypair() (*Keypair, error) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate X25519 key: %w", err)
	}
	return &Keypair{Private: priv, Public: priv.PublicKey()}, nil
}

// MarshalPrivateKey returns the 32-byte raw private key.
func (kp *Keypair) MarshalPrivateKey() []byte {
	return kp.Private.Bytes()
}

// MarshalPublicKey returns the 32-byte raw public key.
func (kp *Keypair) MarshalPublicKey() []byte {
	return kp.Public.Bytes()
}

// UnmarshalKeypair reconstructs a keypair from a 32-byte private key.
func UnmarshalKeypair(privBytes []byte) (*Keypair, error) {
	priv, err := ecdh.X25519().NewPrivateKey(privBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return &Keypair{Private: priv, Public: priv.PublicKey()}, nil
}

// UnmarshalPublicKey parses a 32-byte public key.
func UnmarshalPublicKey(pubBytes []byte) (*ecdh.PublicKey, error) {
	pub, err := ecdh.X25519().NewPublicKey(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	return pub, nil
}

// deriveKEKFromKeypair performs X25519 ECDH between an ephemeral private key
// and a recipient's public key, then derives a 256-bit KEK via HKDF-SHA256.
func deriveKEKFromKeypair(ephemeralPriv *ecdh.PrivateKey, recipientPub *ecdh.PublicKey) ([]byte, error) {
	shared, err := ephemeralPriv.ECDH(recipientPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH: %w", err)
	}

	kek, err := hkdf.Key(sha256.New, shared, nil, "bujotui-kek", 32)
	if err != nil {
		return nil, fmt.Errorf("HKDF: %w", err)
	}
	return kek, nil
}

// deriveKEKForDecrypt performs ECDH with the recipient's private key and
// the ephemeral public key stored in the key slot.
func deriveKEKForDecrypt(recipientPriv *ecdh.PrivateKey, ephemeralPub *ecdh.PublicKey) ([]byte, error) {
	return deriveKEKFromKeypair(recipientPriv, ephemeralPub)
}
