package crypto_test

import (
	"fmt"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
)

// algoSignName 将 SignAlgo 转成可读名称，避免子测试名出现 \x00。
func algoSymCipherName(algo crypto.EncAlgo) string {
	switch algo {
	case crypto.EncChaCha20Poly1305:
		return "EncChaCha20Poly1305"
	case crypto.EncXChaCha20Poly1305:
		return "EncXChaCha20Poly1305"
	default:
		return fmt.Sprintf("Unknown(%d)", algo)
	}
}