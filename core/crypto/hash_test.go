package crypto_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
	"github.com/erashaperavm/BartmossRoar/core/crypto/factory"
)

// algoName 将 HashAlgo 转成可读名称，避免子测试名出现 \x00。
func algoHashName(algo crypto.HashAlgo) string {
	switch algo {
	case crypto.HashBlake3:
		return "Blake3"
	case crypto.HashSHA256:
		return "SHA256"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}

// ---------- 块级单数据 ----------

func testHash(t *testing.T, impl crypto.Hasher, data []byte) {
	t.Helper()
	hash0 := impl.Hash(data)
	hash1 := impl.Hash(data)

	if !bytes.Equal(hash0[:], hash1[:]) {
		t.Errorf("mismatch:  first:  %q\n  second: %q", hash0[:], hash1[:])
	}
}

// ---------- 块级多数据 ----------

func testHashMany(t *testing.T, impl crypto.Hasher, datas [][]byte) {
	t.Helper()
	for i, data := range datas {
		hash0 := impl.Hash(data)
		hash1 := impl.Hash(data)

		if !bytes.Equal(hash0[:], hash1[:]) {
			t.Errorf("index %d mismatch:  first:  %q\n  second: %q",i,  hash0[:], hash1[:])
		}
	}
}

// ---------- 主测试 ----------

func TestHash(t *testing.T) {
	algorithms := []crypto.HashAlgo{
		crypto.HashBlake3,
		crypto.HashSHA256,
	}

	singleCases := []struct {
		name string
		data []byte
	}{
		{"ordinary", []byte("hello world from TestHash")},
		{"empty", []byte("")},
		{"fat", bytes.Repeat([]byte("abc"), 1000)},
	}

	multiCases := []struct {
		name string
		data [][]byte
	}{
		{"ordinary", [][]byte{
			[]byte("hello world from TestHash"),
			[]byte("hej world from TestHash"),
			[]byte("hi world from TestHash"),
		}},
		{"empty", [][]byte{
			[]byte(""),
			[]byte(""),
			[]byte(""),
		}},
		{"fat", [][]byte{
			bytes.Repeat([]byte("abc"), 1000),
			bytes.Repeat([]byte("def"), 1000),
			bytes.Repeat([]byte("ghi"), 1000),
		}},
	}

	for _, algo := range algorithms {
		t.Run(algoHashName(algo), func(t *testing.T) {
			impl, err := factory.NewHasher(algo)
			if err != nil {
				t.Fatalf("NewHasher(%v) returned err: %e", algoHashName(algo), err)
			}
			if impl == nil {
				t.Fatalf("NewHasher(%v) returned nil", algoHashName(algo))
			}

			t.Run("Hash", func(t *testing.T) {
				for _, tc := range singleCases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						testHash(t, impl, tc.data)
					})
				}
			})

			t.Run("HashMany", func(t *testing.T) {
				for _, tc := range multiCases {
					tc := tc
					t.Run(tc.name, func(t *testing.T) {
						testHashMany(t, impl, tc.data)
					})
				}
			})

		})
	}
}
