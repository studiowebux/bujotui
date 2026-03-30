// Package crypto provides optional encryption at rest for bujotui data.
//
// It supports two credential types for unlocking encrypted files:
//   - Password-based: PBKDF2 derives a key encryption key (KEK)
//   - Keypair-based: X25519 ECDH + HKDF derives a KEK
//
// Both use AES-256-GCM for data encryption. A random data encryption key
// (DEK) is generated per vault and encrypted to each credential as a
// "key slot." Any valid credential can unlock the DEK, which then
// decrypts the data.
//
// This package imports nothing from the rest of the project.
package crypto
