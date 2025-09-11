package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
	"fmt"
	"strings"
)

// x/evm module sentinel errors
var (
	ErrInvalidSigner     = sdkerrors.Register(ModuleName, 1100, "expected gov account as only signer for proposal message")
	ErrInvalidGasManager = sdkerrors.Register(ModuleName, 1102, "expected gas manager account as only signer for proposal message")
	ErrBlackListsUpdate  = sdkerrors.Register(ModuleName, 1103, "update blacklists error")
	ErrSample            = sdkerrors.Register(ModuleName, 1101, "sample error")
	// ErrTxTooLarge defines an ABCI typed error where tx is too large.
	ErrTxTooLarge = sdkerrors.Register(ModuleName, 21, "tx too large")
)

type AssociationMissingErr struct {
	Address string
}

func NewAssociationMissingErr(address string) AssociationMissingErr {
	return AssociationMissingErr{Address: address}
}

func (e AssociationMissingErr) Error() string {
	return fmt.Sprintf("address %s is not linked", e.Address)
}

func (e AssociationMissingErr) AddressType() string {
	if strings.HasPrefix(e.Address, "0x") {
		return "evm"
	}
	return "eni"
}
