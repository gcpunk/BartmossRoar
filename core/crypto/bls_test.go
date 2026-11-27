package crypto_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
	"github.com/erashaperavm/BartmossRoar/core/crypto/circl_sign/bls/BLS12_381"
	"github.com/erashaperavm/BartmossRoar/core/crypto/factory"
)

// algoBlsName converts a BLS algorithm identifier into a readable test name.
func algoBlsName(algo crypto.BlsAlgo) string {
	switch algo {
	case crypto.Bls12381:
		return "Bls12381"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}

// newFixedBlsSigner creates a signer deterministically for repeatable test results.
func newFixedBlsSigner(t *testing.T, algo crypto.BlsAlgo) (crypto.BlsSigner, crypto.BlsPublicKey) {
	t.Helper()

	// Save the original random source so it can be restored after key generation.
	originalReader := rand.Reader
	var signer crypto.BlsSigner
	var pk crypto.BlsPublicKey
	var err error
	func() {
		defer func() {
			// Restore the process-wide random source before returning.
			rand.Reader = originalReader
		}()
		// Supply a fixed 32-byte input for deterministic key generation.
		rand.Reader = bytes.NewReader(bytes.Repeat([]byte{0x42}, 32))
		signer, pk, err = factory.NewBLSSigner(algo)
	}()
	// Stop the test when key generation reports an error.
	if err != nil {
		t.Fatalf("NewBLSSigner(%v) returned err: %v", algoBlsName(algo), err)
	}
	// Check that key generation returns a usable private-key object.
	if signer == nil {
		t.Fatalf("NewBLSSigner(%v) returned nil signer", algoBlsName(algo))
	}
	// Check that key generation returns a usable public-key object.
	if pk == nil {
		t.Fatalf("NewBLSSigner(%v) returned nil bls public key", algoBlsName(algo))
	}
	return signer, pk
}

// TestBLS verifies signing, key serialization, equality, and aggregation.
func TestBLS(t *testing.T) {
	algorithms := []crypto.BlsAlgo{
		crypto.Bls12381,
	}

	singleCases := []struct {
		name string
		data []byte
	}{
		{"ordinary", []byte("hello world from TestSign")},
		{"empty", []byte("")},
		{"fat", bytes.Repeat([]byte("abc"), 1000)},
	}

	for _, algo := range algorithms {
		t.Run(algoBlsName(algo), func(t *testing.T) {
			signer, pk := newFixedBlsSigner(t, algo)

			// Verify the generated private key and public key.
			t.Run("key validation", func(t *testing.T) {
				if !signer.Validate() {
					t.Error("generated signer should validate")
				}
				if !pk.Validate() {
					t.Error("generated public key should validate")
				}

				if !pk.Equal(signer.Public()) {
					t.Error("public key should equal the signer's public key")
				}

				otherSigner, otherPK := newFixedBlsSigner(t, algo)
				if !signer.Equal(otherSigner) {
					t.Error("signers generated from the same fixed input should be equal")
				}
				if !pk.Equal(otherPK) {
					t.Error("public keys generated from the same fixed input should be equal")
				}
			})

			// Verify private- and public-key round trips through binary serialization.
			t.Run("key serialization", func(t *testing.T) {
				privateBytes, err := signer.MarshalBinary()
				if err != nil {
					t.Fatalf("marshal signer: %v", err)
				}
				publicBytes, err := pk.MarshalBinary()
				if err != nil {
					t.Fatalf("marshal public key: %v", err)
				}

				restoredSigner, restoredPK := newFixedBlsSigner(t, algo)
				if err := restoredSigner.UnmarshalBinary(privateBytes); err != nil {
					t.Fatalf("unmarshal signer: %v", err)
				}
				if err := restoredPK.UnmarshalBinary(publicBytes); err != nil {
					t.Fatalf("unmarshal public key: %v", err)
				}

				if !signer.Equal(restoredSigner) {
					t.Error("private key changed after marshal/unmarshal")
				}
				if !pk.Equal(restoredPK) {
					t.Error("public key changed after marshal/unmarshal")
				}
				if !restoredPK.Equal(restoredSigner.Public()) {
					t.Error("restored public key should match the restored signer")
				}
			})

			// Sign, verify, and reject modified messages for ordinary, empty, and large inputs.
			for _, sc := range singleCases {
				sc := sc
				t.Run("sign/"+sc.name, func(t *testing.T) {
					signature, err := signer.Sign(sc.data)
					if err != nil {
						t.Fatalf("Sign returned err: %v", err)
					}
					if len(signature) == 0 {
						t.Fatal("Sign returned an empty signature")
					}

					secondSignature, err := signer.Sign(sc.data)
					if err != nil {
						t.Fatalf("second Sign returned err: %v", err)
					}
					if !bytes.Equal(signature, secondSignature) {
						t.Fatal("signing the same message produced different signatures")
					}

					if !pk.Verify(sc.data, signature) {
						t.Fatal("signature verification failed")
					}

					changedMessage := append([]byte(nil), sc.data...)
					if len(changedMessage) == 0 {
						changedMessage = []byte{0}
					} else {
						changedMessage[len(changedMessage)-1] ^= 0xff
					}
					if pk.Verify(changedMessage, signature) {
						t.Fatal("signature unexpectedly verified for a modified message")
					}

					changedSignature := append([]byte(nil), signature...)
					changedSignature[len(changedSignature)-1] ^= 0xff
					if pk.Verify(sc.data, changedSignature) {
						t.Fatal("public key unexpectedly verified a modified signature")
					}
				})
			}

			// Exercise the concrete scheme implementation for algorithm and aggregation behavior.
			t.Run("scheme", func(t *testing.T) {
				scheme := BLS12_381.Scheme{}
				if scheme.Algorithm() != algo {
					t.Fatalf("scheme algorithm = %v, want %v", scheme.Algorithm(), algo)
				}

				firstSigner, firstPK, err := scheme.GenerateKeyPair()
				if err != nil {
					t.Fatalf("generate first key pair: %v", err)
				}
				secondSigner, secondPK, err := scheme.GenerateKeyPair()
				if err != nil {
					t.Fatalf("generate second key pair: %v", err)
				}

				firstMessage := []byte("first aggregate message")
				secondMessage := []byte("second aggregate message")
				firstSignature, err := firstSigner.Sign(firstMessage)
				if err != nil {
					t.Fatalf("sign first aggregate message: %v", err)
				}
				secondSignature, err := secondSigner.Sign(secondMessage)
				if err != nil {
					t.Fatalf("sign second aggregate message: %v", err)
				}

				aggregated, err := scheme.Aggregate([]crypto.Signature{firstSignature, secondSignature})
				if err != nil {
					t.Fatalf("aggregate signatures: %v", err)
				}
				if len(aggregated) == 0 {
					t.Fatal("aggregate returned an empty signature")
				}

				publicKeys := []crypto.BlsPublicKey{firstPK, secondPK}
				messages := [][]byte{firstMessage, secondMessage}
				if !scheme.VerifyAggregate(publicKeys, messages, aggregated) {
					t.Fatal("aggregate signature verification failed")
				}

				if scheme.VerifyAggregate(publicKeys, [][]byte{secondMessage, secondMessage}, aggregated) {
					t.Fatal("aggregate signature unexpectedly verified for wrong messages")
				}
				if scheme.VerifyAggregate(publicKeys[:1], messages, aggregated) {
					t.Fatal("aggregate verification unexpectedly accepted mismatched key and message counts")
				}
			})
		})
	}

	// Verify that the factory rejects algorithms outside the supported BLS set.
	t.Run("unknown algorithm", func(t *testing.T) {
		signer, pk, err := factory.NewBLSSigner(crypto.BlsAlgo(255))
		if err == nil {
			t.Fatal("NewBLSSigner should reject an unknown algorithm")
		}
		if signer != nil {
			t.Fatal("NewBLSSigner returned a signer for an unknown algorithm")
		}
		if pk != nil {
			t.Fatal("NewBLSSigner returned a public key for an unknown algorithm")
		}
	})
}
