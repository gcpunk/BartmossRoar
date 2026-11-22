package crypto_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
	"github.com/erashaperavm/BartmossRoar/core/crypto/factory"
)

// algoSignName 将 SignAlgo 转成可读名称，避免子测试名出现 \x00。
func algoSignName(algo crypto.SignAlgo) string {
	switch algo {
	case crypto.SignMLDSA44:
		return "SignMLDSA44"
	case crypto.SignMLDSA65:
		return "SignMLDSA65"
	case crypto.SignMLDSA87:
		return "SignMLDSA87"
	case crypto.SignEd25519:
		return "SignEd25519"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}

func newFixedSigner(t *testing.T, algo crypto.SignAlgo) (crypto.Signer, crypto.PublicKey) {
	t.Helper()

	originalReader := rand.Reader
	var signer crypto.Signer
	var publicKey crypto.PublicKey
	var err error
	func() {
		defer func() {
			rand.Reader = originalReader
		}()
		rand.Reader = bytes.NewReader(bytes.Repeat([]byte{0x42}, 32))
		signer, publicKey, err = factory.NewSigner(algo)
	}()

	if err != nil {
		t.Fatalf("NewSigner(%v) returned err: %v", algoSignName(algo), err)
	}
	if signer == nil {
		t.Fatalf("NewSigner(%v) returned nil signer", algoSignName(algo))
	}
	if publicKey == nil {
		t.Fatalf("NewSigner(%v) returned nil public key", algoSignName(algo))
	}
	return signer, publicKey
}

func testSignWithCtx(t *testing.T, signer crypto.Signer, publicKey crypto.PublicKey, message []byte, ctx []byte) {
	t.Helper()

	signature, err := signer.Sign(message, ctx)
	if err != nil {
		t.Fatalf("Sign returned err: %v", err)
	}
	if !publicKey.Verify(message, ctx, signature) {
		t.Errorf("signature did not verify, message: , ")
	}

	secondSignature, err := signer.Sign(message, ctx)
	if err != nil {
		t.Fatalf("second Sign returned err: %v", err)
	}
	if !bytes.Equal(signature, secondSignature) {
		t.Errorf("signing the same message with the fixed key produced different signatures")
	}

	if publicKey.Verify([]byte("modified message"), context, signature) {
		t.Fatal("signature verified for a modified message")
	}
	if publicKey.Verify(message, []byte("modified context"), signature) {
		t.Fatal("signature verified for a modified context")
	}
}

func testSignWithoutCtx(t *testing.T, implSign crypto.Signer, implPK crypto.PublicKey, message []byte) {
	t.Helper()
}

func TestSign(t *testing.T) {
	algorithms := []crypto.SignAlgo{
		crypto.SignMLDSA44,
		crypto.SignMLDSA65,
		crypto.SignMLDSA87,
		crypto.SignEd25519,
	}

	singleCasesWithCtx := []struct {
		name string
		data []byte
		ctx  []byte
	}{
		{"ordinary", []byte("hello world from TestSign"), []byte("protocol-br")},
		{"empty", []byte(""), []byte("protocol-br")},
		{"fat", bytes.Repeat([]byte("abc"), 1000), []byte("protocol-br")},
	}

	singleCasesWithoutCtx := []struct {
		name string
		data []byte
	}{
		{"ordinary", []byte("hello world from TestSign")},
		{"empty", []byte("")},
		{"fat", bytes.Repeat([]byte("abc"), 1000)},
	}

	for _, algo := range algorithms {
		t.Run(algoSignName(algo), func(t *testing.T) {
			signer, publicKey := newFixedSigner(t, algo)

		})
	}
}
