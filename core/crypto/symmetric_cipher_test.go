package crypto_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
	"github.com/erashaperavm/BartmossRoar/core/crypto/factory"
)

// algoSymmetricCipherName 将 EncAlgo 转成可读名称，避免子测试名出现 \x00。
func algoSymmetricCipherName(algo crypto.EncAlgo) string {
	switch algo {
	case crypto.EncChaCha20Poly1305:
		return "EncChaCha20Poly1305"
	case crypto.EncXChaCha20Poly1305:
		return "EncXChaCha20Poly1305"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}

func newFixedSymmetricCipher(t *testing.T, algo crypto.EncAlgo) crypto.SymmetricCipher {
	t.Helper()

	pk := crypto.SymmetricCipherKey(bytes.Repeat([]byte{0x42}, 32))
	cipher, err := factory.NewSymmetricCipher(algo, pk)
	if err != nil {
		t.Fatalf("NewSymmetricCipher(%v, %v) returned err: %v", algoSymmetricCipherName(algo), pk, err)
	}
	return cipher
}

func TestSymmetricCipher(t *testing.T) {
	algorithms := []crypto.EncAlgo{
		crypto.EncChaCha20Poly1305,
		crypto.EncXChaCha20Poly1305,
	}

	singleCases := []struct {
		name string
		data []byte
	}{
		{"ordinary", []byte("hello world from TestSymmetricCipher")},
		{"empty", []byte("")},
		{"fat", bytes.Repeat([]byte("abc"), 1000)},
	}

	for _, algo := range algorithms {
		t.Run(algoSymmetricCipherName(algo), func(t *testing.T) {
			cipher := newFixedSymmetricCipher(t, algo)

			for _, sc := range singleCases {
				ciphertext, err := cipher.Encrypt(sc.data)
				if err != nil {
					t.Fatalf("Encrypt returned err: %v", err)
				}
				plaintext, err := cipher.Decrypt(ciphertext)
				if err != nil {
					t.Fatalf("Decrypt returned err: %v", err)
				}

				if !bytes.Equal(plaintext, sc.data) {
					t.Error("plaintext is different from origin data")
				}
			}
		})
	}
}
