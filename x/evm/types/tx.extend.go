package types

import (
	"errors"
	"math/big"

	"github.com/cosmos/cosmos-sdk/utils"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetEvmSender returns the sender address of the transaction.
func (m *MsgEVMTransaction) GetEvmSender() (common.Address, error) {
	if m.Derived != nil && m.Derived.SenderEVMAddr != (common.Address{}) {
		return m.Derived.SenderEVMAddr, nil
	}
	txData, err := UnpackTxData(m.Data)
	if err != nil {
		return common.Address{}, err
	}
	if atx, ok := txData.(*ethtx.AssociateTx); ok {
		V, R, S := atx.GetRawSignatureValues()
		V = new(big.Int).Add(V, utils.Big27)
		// Hash custom message passed in
		customMessageHash := crypto.Keccak256Hash([]byte(atx.CustomMessage))
		pubkey, err1 := RecoverPubkey(customMessageHash, R, S, V, true)
		if err1 != nil {
			return common.Address{}, err1
		}
		return PubkeyToEVMAddress(pubkey)
	}
	ethTx := ethtypes.NewTx(txData.AsEthereumData())
	chainID := ethTx.ChainId()
	signer := ethtypes.LatestSignerForChainID(chainID)
	sender, err := signer.Sender(ethTx)
	if err != nil {
		return common.Address{}, err
	}
	return sender, nil
}

// first half of go-ethereum/core/types/transaction_signing.go:recoverPlain
func RecoverPubkey(sighash common.Hash, R, S, Vb *big.Int, homestead bool) ([]byte, error) {
	if Vb.BitLen() > 8 {
		return []byte{}, ethtypes.ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, homestead) {
		return []byte{}, ethtypes.ErrInvalidSig
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, crypto.SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	// recover the public key from the signature
	return crypto.Ecrecover(sighash[:], sig)
}

// second half of go-ethereum/core/types/transaction_signing.go:recoverPlain
func PubkeyToEVMAddress(pub []byte) (common.Address, error) {
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}
