package factory

import (
	"fmt"

	"github.com/erashaperavm/BartmossRoar/core/crypto"
	"github.com/erashaperavm/BartmossRoar/core/crypto/circl_sign/bls/BLS12_381"
)

func NewBLSSigner(algo crypto.BlsAlgo) (crypto.BlsSigner, crypto.BlsPublicKey, error) {
	switch algo {
	case crypto.Bls12381:
		signer, pk, err := BLS12_381.Scheme{}.GenerateKeyPair()
		if err != nil {
			return nil, nil, err
		}
		return signer, pk, nil
	default:
		return nil, nil, fmt.Errorf("unknown bls algo: %d", algo)
	}
}
