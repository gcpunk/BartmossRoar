package crypto_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
	"github.com/erashaperavm/BartmossRoar/core/crypto/factory"
)

// algoKemName 将 KemAlgo 转成可读名称，避免子测试名出现 \x00。
func algoKemName(algo crypto.KemAlgo) string {
	switch algo {
	case crypto.X25519MLKEM768:
		return "X25519MLKEM768"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}

// ---------- 主测试 ----------
// 测试从双方密钥生成、封装、密钥协商

func TestKEM(t *testing.T) {
	algorithms := []crypto.KemAlgo{
		crypto.X25519MLKEM768,
	}

	for _, algo := range algorithms {
		t.Run(algoKemName(algo), func(t *testing.T) {
			impl, err := factory.NewKEM(algo)
			if err != nil {
				t.Fatalf("NewKEM(%v) returned err: %v", algoKemName(algo), err)
			}
			if impl == nil {
				t.Fatalf("NewKEM(%v) returned nil", algoKemName(algo))
			}

			t.Run("KEM", func(t *testing.T) {
				fixedCases := []struct {
					name string
					seed []byte
				}{
					{"zeroSeed", make([]byte, impl.SeedSize())},
					{"onesSeed", bytes.Repeat([]byte{1}, impl.SeedSize())},
					{"patternSeed", bytes.Repeat([]byte{1, 2, 3, 4}, impl.SeedSize()/4+1)[:impl.SeedSize()]},
				}

				for _, tc := range fixedCases {
					t.Run(tc.name, func(t *testing.T) {
						pk, sk := impl.DeriveKeyPair(tc.seed)
						if pk == nil || sk == nil {
							t.Fatal("DeriveKeyPair returned a nil key")
						}
						if !pk.Equal(sk.Public()) {
							t.Fatal("derived private key does not correspond to derived public key")
						}

						pkBytes, err := pk.MarshalBinary()
						if err != nil {
							t.Fatalf("marshal public key: %v", err)
						}
						skBytes, err := sk.MarshalBinary()
						if err != nil {
							t.Fatalf("marshal private key: %v", err)
						}

						restoredPK, err := impl.UnmarshalBinaryPublicKey(pkBytes)
						if err != nil {
							t.Fatalf("unmarshal public key: %v", err)
						}
						restoredSK, err := impl.UnmarshalBinaryPrivateKey(skBytes)
						if err != nil {
							t.Fatalf("unmarshal private key: %v", err)
						}
						if !pk.Equal(restoredPK) {
							t.Fatal("public key changed after marshal/unmarshal")
						}
						if !restoredSK.Public().Equal(pk) {
							t.Fatal("private key changed after marshal/unmarshal")
						}

						ciphertext, senderSecret, err := impl.Encapsulate(restoredPK)
						if err != nil {
							t.Fatalf("Encapsulate: %v", err)
						}
						receiverSecret, err := impl.Decapsulate(restoredSK, ciphertext)
						if err != nil {
							t.Fatalf("Decapsulate: %v", err)
						}
						if !bytes.Equal(senderSecret, receiverSecret) {
							t.Errorf("encapsulated and decapsulated shared secrets differ")
						}
					})
				}
			})
		})
	}
}
