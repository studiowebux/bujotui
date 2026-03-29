package crypto

import (
	"bytes"
	"testing"
)

func TestAESGCMRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := []byte("hello world, this is secret data")

	ciphertext, err := aesGCMEncrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := aesGCMDecrypt(key, ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestAESGCMWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	ciphertext, err := aesGCMEncrypt(key1, []byte("secret"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err = aesGCMDecrypt(key2, ciphertext)
	if err == nil {
		t.Fatal("expected error with wrong key")
	}
}

func TestAESGCMBadKeySize(t *testing.T) {
	_, err := aesGCMEncrypt([]byte("short"), []byte("data"))
	if err == nil {
		t.Fatal("expected error with short key")
	}
}

func TestGenerateDEK(t *testing.T) {
	dek1, err := generateDEK()
	if err != nil {
		t.Fatalf("generateDEK: %v", err)
	}
	if len(dek1) != 32 {
		t.Fatalf("DEK length: got %d, want 32", len(dek1))
	}

	dek2, _ := generateDEK()
	if bytes.Equal(dek1, dek2) {
		t.Fatal("two DEKs should not be equal")
	}
}

func TestPBKDF2Deterministic(t *testing.T) {
	salt := make([]byte, 32)
	k1 := deriveKEKFromPassword("password", salt)
	k2 := deriveKEKFromPassword("password", salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("same password+salt should produce same key")
	}

	k3 := deriveKEKFromPassword("different", salt)
	if bytes.Equal(k1, k3) {
		t.Fatal("different passwords should produce different keys")
	}
}

func TestPasswordSlotRoundtrip(t *testing.T) {
	dek, _ := generateDEK()
	password := "my-secret-password"

	slot, err := NewPasswordSlot(password, dek)
	if err != nil {
		t.Fatalf("NewPasswordSlot: %v", err)
	}

	if slot.Type != SlotPassword {
		t.Fatalf("type: got %d, want %d", slot.Type, SlotPassword)
	}

	// Marshal/unmarshal
	data := slot.Marshal()
	if len(data) != SlotSize {
		t.Fatalf("marshal size: got %d, want %d", len(data), SlotSize)
	}

	slot2, err := UnmarshalKeySlot(data)
	if err != nil {
		t.Fatalf("UnmarshalKeySlot: %v", err)
	}

	// Decrypt DEK
	recovered, err := slot2.DecryptDEKWithPassword(password)
	if err != nil {
		t.Fatalf("DecryptDEK: %v", err)
	}

	if !bytes.Equal(recovered, dek) {
		t.Fatal("recovered DEK doesn't match original")
	}
}

func TestPasswordSlotWrongPassword(t *testing.T) {
	dek, _ := generateDEK()
	slot, _ := NewPasswordSlot("correct", dek)

	_, err := slot.DecryptDEKWithPassword("wrong")
	if err == nil {
		t.Fatal("expected error with wrong password")
	}
}

func TestKeypairGeneration(t *testing.T) {
	kp, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	if len(kp.MarshalPrivateKey()) != 32 {
		t.Fatal("private key should be 32 bytes")
	}
	if len(kp.MarshalPublicKey()) != 32 {
		t.Fatal("public key should be 32 bytes")
	}
}

func TestKeypairMarshalRoundtrip(t *testing.T) {
	kp, _ := GenerateKeypair()
	privBytes := kp.MarshalPrivateKey()

	kp2, err := UnmarshalKeypair(privBytes)
	if err != nil {
		t.Fatalf("UnmarshalKeypair: %v", err)
	}

	if !bytes.Equal(kp.MarshalPublicKey(), kp2.MarshalPublicKey()) {
		t.Fatal("public keys should match after roundtrip")
	}
}

func TestKeypairSlotRoundtrip(t *testing.T) {
	dek, _ := generateDEK()
	recipient, _ := GenerateKeypair()

	slot, err := NewKeypairSlot(recipient.Public, dek)
	if err != nil {
		t.Fatalf("NewKeypairSlot: %v", err)
	}

	if slot.Type != SlotKeypair {
		t.Fatalf("type: got %d, want %d", slot.Type, SlotKeypair)
	}

	// Marshal/unmarshal
	data := slot.Marshal()
	slot2, err := UnmarshalKeySlot(data)
	if err != nil {
		t.Fatalf("UnmarshalKeySlot: %v", err)
	}

	recovered, err := slot2.DecryptDEKWithPrivateKey(recipient.Private)
	if err != nil {
		t.Fatalf("DecryptDEK: %v", err)
	}

	if !bytes.Equal(recovered, dek) {
		t.Fatal("recovered DEK doesn't match original")
	}
}

func TestKeypairSlotWrongKey(t *testing.T) {
	dek, _ := generateDEK()
	recipient, _ := GenerateKeypair()
	other, _ := GenerateKeypair()

	slot, _ := NewKeypairSlot(recipient.Public, dek)

	_, err := slot.DecryptDEKWithPrivateKey(other.Private)
	if err == nil {
		t.Fatal("expected error with wrong private key")
	}
}

func TestFileFormatPasswordRoundtrip(t *testing.T) {
	plaintext := []byte("journal entries go here\nwith multiple lines\n")
	password := "hunter2"

	vault, err := NewVault()
	if err != nil {
		t.Fatalf("NewVault: %v", err)
	}

	slot, err := NewPasswordSlot(password, vault.DEK())
	if err != nil {
		t.Fatalf("NewPasswordSlot: %v", err)
	}

	encrypted, err := EncryptFile(vault, []*KeySlot{slot}, plaintext)
	if err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	if !IsEncrypted(encrypted) {
		t.Fatal("IsEncrypted should return true")
	}

	decrypted, _, _, err := DecryptFileWithPassword(encrypted, password)
	if err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestFileFormatKeypairRoundtrip(t *testing.T) {
	plaintext := []byte("secret stuff")
	kp, _ := GenerateKeypair()

	vault, _ := NewVault()
	slot, _ := NewKeypairSlot(kp.Public, vault.DEK())

	encrypted, err := EncryptFile(vault, []*KeySlot{slot}, plaintext)
	if err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	decrypted, _, _, err := DecryptFileWithPrivateKey(encrypted, kp.Private)
	if err != nil {
		t.Fatalf("DecryptFile: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestFileFormatMultipleSlots(t *testing.T) {
	plaintext := []byte("shared data")

	vault, _ := NewVault()
	dek := vault.DEK()

	// Password slot
	pwSlot, _ := NewPasswordSlot("password1", dek)

	// Keypair slot
	kp, _ := GenerateKeypair()
	kpSlot, _ := NewKeypairSlot(kp.Public, dek)

	// Second password
	pw2Slot, _ := NewPasswordSlot("password2", dek)

	encrypted, err := EncryptFile(vault, []*KeySlot{pwSlot, kpSlot, pw2Slot}, plaintext)
	if err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	// Decrypt with first password
	dec1, _, _, err := DecryptFileWithPassword(encrypted, "password1")
	if err != nil {
		t.Fatalf("password1: %v", err)
	}
	if !bytes.Equal(dec1, plaintext) {
		t.Fatal("password1 decryption mismatch")
	}

	// Decrypt with second password
	dec2, _, _, err := DecryptFileWithPassword(encrypted, "password2")
	if err != nil {
		t.Fatalf("password2: %v", err)
	}
	if !bytes.Equal(dec2, plaintext) {
		t.Fatal("password2 decryption mismatch")
	}

	// Decrypt with keypair
	dec3, _, _, err := DecryptFileWithPrivateKey(encrypted, kp.Private)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	if !bytes.Equal(dec3, plaintext) {
		t.Fatal("keypair decryption mismatch")
	}

	// Wrong password fails
	_, _, _, err = DecryptFileWithPassword(encrypted, "wrong")
	if err == nil {
		t.Fatal("wrong password should fail")
	}

	// Wrong keypair fails
	other, _ := GenerateKeypair()
	_, _, _, err = DecryptFileWithPrivateKey(encrypted, other.Private)
	if err == nil {
		t.Fatal("wrong keypair should fail")
	}
}

func TestIsEncryptedPlaintext(t *testing.T) {
	if IsEncrypted([]byte("# 2026-03-28\n- . task")) {
		t.Fatal("plain markdown should not be detected as encrypted")
	}
	if IsEncrypted(nil) {
		t.Fatal("nil should not be encrypted")
	}
	if IsEncrypted([]byte("BU")) {
		t.Fatal("short data should not be encrypted")
	}
}

func TestParseSlots(t *testing.T) {
	vault, _ := NewVault()
	slot1, _ := NewPasswordSlot("pw", vault.DEK())
	kp, _ := GenerateKeypair()
	slot2, _ := NewKeypairSlot(kp.Public, vault.DEK())

	encrypted, _ := EncryptFile(vault, []*KeySlot{slot1, slot2}, []byte("data"))

	slots, err := ParseSlots(encrypted)
	if err != nil {
		t.Fatalf("ParseSlots: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("slot count: got %d, want 2", len(slots))
	}
	if slots[0].Type != SlotPassword {
		t.Fatalf("slot 0 type: got %d, want %d", slots[0].Type, SlotPassword)
	}
	if slots[1].Type != SlotKeypair {
		t.Fatalf("slot 1 type: got %d, want %d", slots[1].Type, SlotKeypair)
	}
}

func TestEmptyPlaintext(t *testing.T) {
	vault, _ := NewVault()
	slot, _ := NewPasswordSlot("pw", vault.DEK())

	encrypted, err := EncryptFile(vault, []*KeySlot{slot}, []byte{})
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}

	decrypted, _, _, err := DecryptFileWithPassword(encrypted, "pw")
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if len(decrypted) != 0 {
		t.Fatalf("expected empty, got %d bytes", len(decrypted))
	}
}
