// SPDX-License-Identifier: Apache-2.0

package blsverifier

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The success vectors were checked against two independent implementations:
// github.com/OffchainLabs/prysm/v6@v6.1.4 crypto/bls, which this package wrapped
// before, and github.com/consensys/gnark-crypto. Both produce aggregatedPublicKey
// from publicKeys and accept aggregatedSignature over message.
//
// The not-in-subgroup vectors are on-curve points outside the prime-order
// subgroup (G1: x = 4, G2: x = 2), generated with gnark-crypto. They decode as
// curve points, so only the subgroup checks reject them.
var (
	publicKeys = [][]byte{
		mustDecodeHex("84fe268400062ae6255010a421204dad5b0a8bd93522a4abc023037fa7a769f6ef29acc14c7fae8c5c0cd1df21ae1f52"),
		mustDecodeHex("8b850e23cf188473e04feaa5d39f40f54b8c0e717f7fccb77d0efb5326802274bda863911efa8775a08674b887e0e766"),
		mustDecodeHex("b52684603da6cb2902092310deab2ef3116a0a2b449c0f44d82d1d30bf12f7474492534f518af541ca9da110ee417132"),
	}
	aggregatedPublicKey = mustDecodeHex("912cf816853d0984e448f065d0bdb34b1e26be1d3c50bbc2f0d39b0ff1c2963a97b5dfd442a496763f3c6e5c54375b1e")
	aggregatedSignature = mustDecodeHex("8af25b4a2786b3305f80528520544071a143e80ef4091dfca32e347eeabeef71eeb8673a78e8ff2875883ab5498d8a7b1888a80878bfd82c2d27e93ed74e979014e95d7f2ef55b1d28070a4a2e6ce9df3b8f07c6444c4799c64d654c2810c36d")
	message             = [32]byte{3, 1, 2, 3}

	g1NotInSubgroup = mustDecodeHex("800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004")
	g2NotInSubgroup = mustDecodeHex("800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002")
)

func mustDecodeHex(s string) []byte {
	bz, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return bz
}

// negate flips the sign bit of a compressed point.
func negate(point []byte) []byte {
	bz := append([]byte{}, point...)
	bz[0] ^= 0x20
	return bz
}

func infinityPoint(size int) []byte {
	bz := make([]byte, size)
	bz[0] = 0xc0
	return bz
}

func TestAggregatePublicKeys(t *testing.T) {
	testCases := []struct {
		name        string
		publicKeys  [][]byte
		expected    []byte
		expErrorMsg string
	}{
		{"success: single key", publicKeys[:1], publicKeys[0], ""},
		{"success: multiple keys", publicKeys, aggregatedPublicKey, ""},
		{"failure: no keys", nil, nil, "nil or empty public keys"},
		{"failure: wrong length", [][]byte{publicKeys[0][:47]}, nil, "public key must be 48 bytes"},
		{"failure: not a curve point", [][]byte{make([]byte, 48)}, nil, "could not unmarshal bytes into public key"},
		{"failure: point at infinity", [][]byte{infinityPoint(48)}, nil, "received an infinite public key"},
		{"failure: point not in subgroup", [][]byte{g1NotInSubgroup}, nil, "received an infinite public key"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pk, err := aggregatePublicKeys(tc.publicKeys)
			if tc.expErrorMsg != "" {
				require.ErrorContains(t, err, tc.expErrorMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, pk)
		})
	}
}

func TestVerifySignature(t *testing.T) {
	testCases := []struct {
		name        string
		signature   []byte
		message     [32]byte
		publicKeys  [][]byte
		expValid    bool
		expErrorMsg string
	}{
		{"success: valid aggregate signature", aggregatedSignature, message, publicKeys, true, ""},
		{"invalid: wrong message", aggregatedSignature, [32]byte{9}, publicKeys, false, ""},
		{"invalid: missing signer", aggregatedSignature, message, publicKeys[:2], false, ""},
		{"invalid: signature at infinity", infinityPoint(96), message, publicKeys, false, ""},
		{"failure: wrong signature length", aggregatedSignature[:95], message, publicKeys, false, "signature must be 96 bytes"},
		{"failure: signature not a curve point", make([]byte, 96), message, publicKeys, false, "could not unmarshal bytes into signature"},
		{"failure: signature not in subgroup", g2NotInSubgroup, message, publicKeys, false, "signature not in group"},
		{"invalid: keys cancel to infinity", infinityPoint(96), message, [][]byte{publicKeys[0], negate(publicKeys[0])}, false, ""},
		{"failure: invalid public key", aggregatedSignature, message, [][]byte{infinityPoint(48)}, false, "failed to aggregate public keys"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, err := VerifySignature(tc.signature, tc.message, tc.publicKeys)
			if tc.expErrorMsg != "" {
				require.ErrorContains(t, err, tc.expErrorMsg)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.expValid, valid)
		})
	}
}

func TestCustomQuerier(t *testing.T) {
	querier := CustomQuerier()

	request, err := json.Marshal(CustomQuery{
		Aggregate: &QueryAggregate{PublicKeys: publicKeys},
	})
	require.NoError(t, err)
	res, err := querier(sdk.Context{}, request)
	require.NoError(t, err)
	expected, err := json.Marshal(aggregatedPublicKey)
	require.NoError(t, err)
	require.Equal(t, expected, res)

	request, err = json.Marshal(CustomQuery{
		AggregateVerify: &QueryAggregateVerify{
			PublicKeys: publicKeys,
			Signature:  aggregatedSignature,
			Message:    message[:],
		},
	})
	require.NoError(t, err)
	res, err = querier(sdk.Context{}, request)
	require.NoError(t, err)
	require.Equal(t, []byte("true"), res)
}
