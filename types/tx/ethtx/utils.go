package ethtx

import (
	"errors"
	"fmt"
	"math/big"

	cosmossdk_io_math "cosmossdk.io/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"math"
)

var Big0 = big.NewInt(0)
var Uint2560 = uint256.NewInt(0)
var Big1 = big.NewInt(1)
var Big2 = big.NewInt(2)
var Big8 = big.NewInt(8)
var Big27 = big.NewInt(27)
var Big35 = big.NewInt(35)
var BigMaxI64 = big.NewInt(math.MaxInt64)
var BigMaxU64 = new(big.Int).SetUint64(math.MaxUint64)

var Sdk0 = cosmossdk_io_math.NewInt(0)

// Effective gas price is the smaller of base fee + tip limit vs total fee limit
func EffectiveGasPrice(baseFee, feeCap, tipCap *big.Int) *big.Int {
	return BigMin(new(big.Int).Add(tipCap, baseFee), feeCap)
}

// BigMin returns the smaller of x or y.
func BigMin(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return y
	}
	return x
}

// Convert a value with the provided converter and set it using the provided setter
func SetConvertIfPresent[U comparable, V any](orig U, converter func(U) V, setter func(V)) {
	var nilU U
	if orig == nilU {
		return
	}

	setter(converter(orig))
}

// validate a ethtypes.Transaction for sdk.Int overflow
func ValidateEthTx(tx *ethtypes.Transaction) error {
	if !IsValidInt256(tx.Value()) {
		return errors.New("value overflow")
	}
	if !IsValidInt256(tx.GasPrice()) {
		return errors.New("gas price overflow")
	}
	if !IsValidInt256(tx.GasFeeCap()) {
		return errors.New("gas fee cap overflow")
	}
	if !IsValidInt256(tx.GasTipCap()) {
		return errors.New("gas tip cap overflow")
	}
	if !IsValidInt256(tx.BlobGasFeeCap()) {
		return errors.New("blob gas fee cap overflow")
	}
	return nil
}

func DecodeSignature(sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != crypto.SignatureLength {
		err = fmt.Errorf("wrong size for signature: got %d, want %d", len(sig), crypto.SignatureLength)
		return
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

func FilterUInt64Slice(slice []uint64, item uint64) []uint64 {
	res := []uint64{}
	for _, i := range slice {
		if i != item {
			res = append(res, i)
		}
	}
	return res
}

func Map[I any, O any](input []I, lambda func(i I) O) []O {
	res := []O{}
	for _, i := range input {
		res = append(res, lambda(i))
	}
	return res
}

func SliceCopy[T any](slice []T) []T {
	return append([]T{}, slice...)
}

func Reduce[I, O any](input []I, reducer func(I, O) O, initial O) O {
	for _, i := range input {
		initial = reducer(i, initial)
	}
	return initial
}

func Filter[T any](slice []T, lambda func(t T) bool) []T {
	res := []T{}
	for _, t := range slice {
		if lambda(t) {
			res = append(res, t)
		}
	}
	return res
}
