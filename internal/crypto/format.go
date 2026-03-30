package crypto

import (
	"crypto/ecdh"
	"fmt"
)

// File format constants.
var magic = [4]byte{'B', 'U', 'J', 'O'}

const formatVersion = 0x01

// IsEncrypted checks if data starts with the BUJO magic header.
func IsEncrypted(data []byte) bool {
	return len(data) >= 4 && data[0] == magic[0] && data[1] == magic[1] &&
		data[2] == magic[2] && data[3] == magic[3]
}

// EncryptFile encrypts plaintext into the full file format:
// magic(4) + version(1) + slotCount(1) + slots(N*96) + nonce+ciphertext.
func EncryptFile(vault *Vault, slots []*KeySlot, plaintext []byte) ([]byte, error) {
	if len(slots) == 0 {
		return nil, fmt.Errorf("at least one key slot is required")
	}
	if len(slots) > 255 {
		return nil, fmt.Errorf("too many key slots (max 255)")
	}

	encrypted, err := vault.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	// Header: magic(4) + version(1) + slotCount(1)
	headerSize := 6
	slotsSize := len(slots) * SlotSize
	totalSize := headerSize + slotsSize + len(encrypted)

	buf := make([]byte, totalSize)
	copy(buf[0:4], magic[:])
	buf[4] = formatVersion
	buf[5] = byte(len(slots)) // #nosec G115 -- len(slots) is bounded to 0-255 by the check on line 31

	offset := headerSize
	for _, slot := range slots {
		copy(buf[offset:offset+SlotSize], slot.Marshal())
		offset += SlotSize
	}

	copy(buf[offset:], encrypted)
	return buf, nil
}

// DecryptFileWithPassword decrypts a file encrypted with EncryptFile,
// trying the password against all password slots.
func DecryptFileWithPassword(data []byte, password string) ([]byte, *Vault, []*KeySlot, error) {
	slots, ciphertext, err := parseFileHeader(data)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, slot := range slots {
		if slot.Type != SlotPassword {
			continue
		}
		dek, err := slot.DecryptDEKWithPassword(password)
		if err == nil {
			vault := &Vault{dek: dek}
			plaintext, err := vault.Decrypt(ciphertext)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("DEK decrypted but data corrupt: %w", err)
			}
			return plaintext, vault, slots, nil
		}
	}

	return nil, nil, nil, fmt.Errorf("no password slot matched")
}

// DecryptFileWithPrivateKey decrypts a file encrypted with EncryptFile,
// trying the private key against all keypair slots.
func DecryptFileWithPrivateKey(data []byte, privKey *ecdh.PrivateKey) ([]byte, *Vault, []*KeySlot, error) {
	slots, ciphertext, err := parseFileHeader(data)
	if err != nil {
		return nil, nil, nil, err
	}

	for _, slot := range slots {
		if slot.Type != SlotKeypair {
			continue
		}
		dek, err := slot.DecryptDEKWithPrivateKey(privKey)
		if err == nil {
			vault := &Vault{dek: dek}
			plaintext, err := vault.Decrypt(ciphertext)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("DEK decrypted but data corrupt: %w", err)
			}
			return plaintext, vault, slots, nil
		}
	}

	return nil, nil, nil, fmt.Errorf("no keypair slot matched")
}

// ParseSlots extracts key slots from an encrypted file without decrypting.
// Useful for key management (list, add, remove slots).
func ParseSlots(data []byte) ([]*KeySlot, error) {
	slots, _, err := parseFileHeader(data)
	return slots, err
}

// ParseFileRaw extracts the format version, slots, and raw ciphertext
// without attempting decryption. The ciphertext includes the nonce prefix.
func ParseFileRaw(data []byte) (version byte, slotCount int, slots []*KeySlot, ciphertext []byte, err error) {
	s, ct, e := parseFileHeader(data)
	if e != nil {
		return 0, 0, nil, nil, e
	}
	return data[4], len(s), s, ct, nil
}

// parseFileHeader validates the header and extracts slots and ciphertext.
func parseFileHeader(data []byte) ([]*KeySlot, []byte, error) {
	if !IsEncrypted(data) {
		return nil, nil, fmt.Errorf("not an encrypted file (missing BUJO header)")
	}
	if len(data) < 6 {
		return nil, nil, fmt.Errorf("file too short")
	}
	if data[4] != formatVersion {
		return nil, nil, fmt.Errorf("unsupported format version: %d", data[4])
	}

	slotCount := int(data[5])
	headerSize := 6
	slotsEnd := headerSize + slotCount*SlotSize

	if len(data) < slotsEnd {
		return nil, nil, fmt.Errorf("file truncated: expected %d bytes for slots", slotsEnd)
	}

	slots := make([]*KeySlot, slotCount)
	for i := range slotCount {
		offset := headerSize + i*SlotSize
		slot, err := UnmarshalKeySlot(data[offset : offset+SlotSize])
		if err != nil {
			return nil, nil, fmt.Errorf("slot %d: %w", i, err)
		}
		slots[i] = slot
	}

	ciphertext := data[slotsEnd:]
	return slots, ciphertext, nil
}

// NewVault creates a new Vault with a fresh random DEK.
func NewVault() (*Vault, error) {
	dek, err := generateDEK()
	if err != nil {
		return nil, err
	}
	return &Vault{dek: dek}, nil
}

// VaultFromDEK creates a Vault from an existing DEK (used after decryption).
func VaultFromDEK(dek []byte) *Vault {
	d := make([]byte, len(dek))
	copy(d, dek)
	return &Vault{dek: d}
}
