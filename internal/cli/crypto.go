package cli

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	bujocrypto "github.com/studiowebux/bujotui/internal/crypto"
	"github.com/studiowebux/bujotui/internal/storage"
)

// cmdKeygen generates a new X25519 keypair and saves it to the config dir.
func cmdKeygen(configDir string, stdout io.Writer) error {
	kp, err := bujocrypto.GenerateKeypair()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Clean(configDir), 0o700); err != nil { // #nosec G703 -- configDir comes from DefaultConfigDir() or --dir flag, not arbitrary user input
		return fmt.Errorf("create config dir: %w", err)
	}

	privPath := filepath.Clean(filepath.Join(configDir, "bujotui.key"))
	pubPath := filepath.Clean(filepath.Join(configDir, "bujotui.pub"))

	// Don't overwrite existing keys
	if _, err := os.Stat(privPath); err == nil { // #nosec G703 -- privPath is configDir + constant filename, no user-controlled path components
		return fmt.Errorf("key already exists at %s (remove it first to regenerate)", privPath)
	}

	privHex := hex.EncodeToString(kp.MarshalPrivateKey())
	pubHex := hex.EncodeToString(kp.MarshalPublicKey())

	if err := os.WriteFile(privPath, []byte(privHex+"\n"), 0o600); err != nil { // #nosec G703 -- privPath is configDir + "bujotui.key"
		return fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, []byte(pubHex+"\n"), 0o600); err != nil { // #nosec G703 -- pubPath is configDir + "bujotui.pub"
		return fmt.Errorf("write public key: %w", err)
	}

	fmt.Fprintf(stdout, "Keypair generated:\n  Private: %s\n  Public:  %s\n", privPath, pubPath)
	fmt.Fprintf(stdout, "Public key: %s\n", pubHex)
	return nil
}

// cmdEncrypt encrypts all data files with a password.
func cmdEncrypt(args []string, store *storage.Store, configDir string, stdout io.Writer) error {
	if store.Vault != nil {
		return fmt.Errorf("data is already encrypted")
	}

	password := ""
	if len(args) > 0 {
		password = args[0]
	}
	if password == "" {
		return fmt.Errorf("usage: bujotui encrypt <password>")
	}

	vault, err := bujocrypto.NewVault()
	if err != nil {
		return err
	}

	slot, err := bujocrypto.NewPasswordSlot(password, vault.DEK())
	if err != nil {
		return err
	}

	// Also add keypair slot if key exists
	slots := []*bujocrypto.KeySlot{slot}
	privPath := filepath.Join(configDir, "bujotui.key")
	if kp, err := loadKeypair(privPath); err == nil {
		kpSlot, err := bujocrypto.NewKeypairSlot(kp.Public, vault.DEK())
		if err == nil {
			slots = append(slots, kpSlot)
			fmt.Fprintf(stdout, "Added keypair slot from %s\n", privPath)
		}
	}

	store.Vault = vault
	store.KeySlots = slots

	count, err := encryptAllFiles(store)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Encrypted %d files.\n", count)
	return nil
}

// cmdDecrypt decrypts all data files.
func cmdDecrypt(args []string, store *storage.Store, configDir string, stdout io.Writer) error {
	password := ""
	if len(args) > 0 {
		password = args[0]
	}

	// Try password
	if password != "" {
		vault, slots, err := openVaultFromFiles(store.Dir, password, "")
		if err != nil {
			return fmt.Errorf("cannot open vault: %w", err)
		}
		store.Vault = vault
		store.KeySlots = slots
	} else {
		// Try keypair
		privPath := filepath.Join(configDir, "bujotui.key")
		vault, slots, err := openVaultFromFiles(store.Dir, "", privPath)
		if err != nil {
			return fmt.Errorf("cannot open vault (no password given, keypair failed): %w", err)
		}
		store.Vault = vault
		store.KeySlots = slots
	}

	count, err := decryptAllFiles(store)
	if err != nil {
		return err
	}

	store.Vault = nil
	store.KeySlots = nil

	fmt.Fprintf(stdout, "Decrypted %d files.\n", count)
	return nil
}

// cmdKeyList shows key slots in the vault.
func cmdKeyList(store *storage.Store, stdout io.Writer) error {
	slots, err := findFirstEncryptedSlots(store.Dir)
	if err != nil {
		return err
	}
	if len(slots) == 0 {
		fmt.Fprintln(stdout, "No encrypted files found.")
		return nil
	}

	fmt.Fprintf(stdout, "Key slots (%d):\n", len(slots))
	for i, slot := range slots {
		slotType := "password"
		if slot.Type == bujocrypto.SlotKeypair {
			slotType = "keypair"
		}
		id := hex.EncodeToString(slot.SaltOrPubkey[:8])
		fmt.Fprintf(stdout, "  %d. %s  %s...\n", i, slotType, id)
	}
	return nil
}

// loadKeypair reads a private key from a hex-encoded file.
func loadKeypair(path string) (*bujocrypto.Keypair, error) {
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304
	if err != nil {
		return nil, err
	}
	privBytes, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid key format: %w", err)
	}
	return bujocrypto.UnmarshalKeypair(privBytes)
}

// openVaultFromFiles opens the vault by finding any encrypted file and
// trying the given credentials.
func openVaultFromFiles(dataDir, password, privKeyPath string) (*bujocrypto.Vault, []*bujocrypto.KeySlot, error) {
	encFile, err := findFirstEncryptedFile(dataDir)
	if err != nil {
		return nil, nil, err
	}

	data, err := os.ReadFile(filepath.Clean(encFile)) // #nosec G304
	if err != nil {
		return nil, nil, err
	}

	if password != "" {
		_, vault, slots, err := bujocrypto.DecryptFileWithPassword(data, password)
		if err == nil {
			return vault, slots, nil
		}
	}

	if privKeyPath != "" {
		kp, err := loadKeypair(privKeyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("load private key: %w", err)
		}
		_, vault, slots, err := bujocrypto.DecryptFileWithPrivateKey(data, kp.Private)
		if err == nil {
			return vault, slots, nil
		}
	}

	return nil, nil, fmt.Errorf("no valid credentials")
}

// findFirstEncryptedFile walks the data directory for any encrypted file.
func findFirstEncryptedFile(dataDir string) (string, error) {
	var found string
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error { // #nosec G122 G703 -- Walk on user-configured data dir to detect encrypted files; TOCTOU accepted
		if err != nil || found != "" || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 G122 -- reading user data dir to detect encryption
		if err != nil {
			return nil
		}
		if bujocrypto.IsEncrypted(data) {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("no encrypted files found")
	}
	return found, nil
}

// findFirstEncryptedSlots reads slots from the first encrypted file found.
func findFirstEncryptedSlots(dataDir string) ([]*bujocrypto.KeySlot, error) {
	encFile, err := findFirstEncryptedFile(dataDir)
	if err != nil {
		return nil, nil // no encrypted files = no slots
	}
	data, err := os.ReadFile(filepath.Clean(encFile)) // #nosec G304
	if err != nil {
		return nil, err
	}
	return bujocrypto.ParseSlots(data)
}

// encryptAllFiles encrypts all .md files in the data directory.
func encryptAllFiles(store *storage.Store) (int, error) {
	count := 0
	return count, filepath.Walk(store.Dir, func(path string, info os.FileInfo, err error) error { // #nosec G122 -- encrypting user's own data dir; TOCTOU risk accepted
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		// Skip .versions directory
		if strings.Contains(path, ".versions") {
			return nil
		}

		data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 G122 -- encrypting user's own data files
		if err != nil || bujocrypto.IsEncrypted(data) {
			return nil // skip already encrypted or unreadable
		}

		encrypted, err := bujocrypto.EncryptFile(store.Vault, store.KeySlots, data)
		if err != nil {
			return fmt.Errorf("encrypt %s: %w", path, err)
		}

		if err := storage.AtomicWriteFile(path, encrypted, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		count++
		return nil
	})
}

// decryptAllFiles decrypts all encrypted .md files in the data directory.
func decryptAllFiles(store *storage.Store) (int, error) {
	count := 0
	return count, filepath.Walk(store.Dir, func(path string, info os.FileInfo, err error) error { // #nosec G122 -- decrypting user's own data dir; TOCTOU risk accepted
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		if strings.Contains(path, ".versions") {
			return nil
		}

		data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 G122 -- decrypting user's own data files
		if err != nil || !bujocrypto.IsEncrypted(data) {
			return nil
		}

		plaintext, err := store.Vault.Decrypt(extractCiphertext(data))
		if err != nil {
			return fmt.Errorf("decrypt %s: %w", path, err)
		}

		if err := storage.AtomicWriteFile(path, plaintext, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		count++
		return nil
	})
}

// extractCiphertext extracts the ciphertext portion after header+slots.
func extractCiphertext(data []byte) []byte {
	_, _, _, ct, err := bujocrypto.ParseFileRaw(data)
	if err != nil {
		return nil
	}
	return ct
}
