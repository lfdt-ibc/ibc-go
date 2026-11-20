// SPDX-License-Identifier: Apache-2.0

package blsverifier

import (
	"errors"
	"fmt"

	blst "github.com/supranational/blst/bindings/go"
)

// dst is the Ethereum consensus proof-of-possession ciphersuite domain separation tag.
var dst = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

// aggregatePublicKeys validates the compressed public keys and returns their compressed aggregate.
func aggregatePublicKeys(publicKeys [][]byte) ([]byte, error) {
	pk, err := aggregatePublicKeyPoint(publicKeys)
	if err != nil {
		return nil, err
	}
	return pk.Compress(), nil
}

// VerifySignature verifies a compressed signature over message against the aggregate of publicKeys.
func VerifySignature(signature []byte, message [32]byte, publicKeys [][]byte) (bool, error) {
	pk, err := aggregatePublicKeyPoint(publicKeys)
	if err != nil {
		return false, fmt.Errorf("failed to aggregate public keys %w", err)
	}

	if len(signature) != blst.BLST_P2_COMPRESS_BYTES {
		return false, fmt.Errorf("could not create signature from byte slice: signature must be %d bytes", blst.BLST_P2_COMPRESS_BYTES)
	}
	sig := new(blst.P2Affine).Uncompress(signature)
	if sig == nil {
		return false, errors.New("could not create signature from byte slice: could not unmarshal bytes into signature")
	}
	// Subgroup check only: an aggregate signature may be the point at infinity.
	if !sig.SigValidate(false) {
		return false, errors.New("signature not in group")
	}

	// Signature and key are already validated above.
	return sig.Verify(false, pk, false, message[:], dst), nil
}

func aggregatePublicKeyPoint(publicKeys [][]byte) (*blst.P1Affine, error) {
	if len(publicKeys) == 0 {
		return nil, errors.New("nil or empty public keys")
	}

	agg := new(blst.P1Aggregate)
	for _, bz := range publicKeys {
		if len(bz) != blst.BLST_P1_COMPRESS_BYTES {
			return nil, fmt.Errorf("public key must be %d bytes", blst.BLST_P1_COMPRESS_BYTES)
		}
		pk := new(blst.P1Affine).Uncompress(bz)
		if pk == nil {
			return nil, errors.New("could not unmarshal bytes into public key")
		}
		// Rejects the point at infinity and points outside the subgroup.
		if !pk.KeyValidate() {
			// Same message as prysm, although it also covers the subgroup check.
			return nil, errors.New("received an infinite public key")
		}
		agg.Add(pk, false)
	}

	return agg.ToAffine(), nil
}
