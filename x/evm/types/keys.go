package types

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName defines the module name
	ModuleName = "evm"

	RouterKey = ModuleName

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey       = "mem_evm"
	TransientStoreKey = "evm_transient"
)

var (
	ParamsKey = []byte("p_evm")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

var (
	EVMAddressToEniAddressKeyPrefix            = []byte{0x01}
	EniAddressToEVMAddressKeyPrefix            = []byte{0x02}
	StateKeyPrefix                             = []byte{0x03}
	TransientStateKeyPrefix                    = []byte{0x04} // deprecated
	AccountTransientStateKeyPrefix             = []byte{0x05} // deprecated
	TransientModuleStateKeyPrefix              = []byte{0x06} // deprecated
	CodeKeyPrefix                              = []byte{0x07}
	CodeHashKeyPrefix                          = []byte{0x08}
	CodeSizeKeyPrefix                          = []byte{0x09}
	NonceKeyPrefix                             = []byte{0x0a}
	ReceiptKeyPrefix                           = []byte{0x0b}
	WhitelistedCodeHashesForBankSendPrefix     = []byte{0x0c}
	BlockBloomPrefix                           = []byte{0x0d}
	TxHashesPrefix                             = []byte{0x0e} // deprecated
	WhitelistedCodeHashesForDelegateCallPrefix = []byte{0x0f}

	// TxHashPrefix  = []byte{0x10}
	// TxBloomPrefix = []byte{0x11}

	ReplaySeenAddrPrefix = []byte{0x12}
	ReplayedHeight       = []byte{0x13}
	ReplayInitialHeight  = []byte{0x14}

	PointerRegistryPrefix        = []byte{0x15}
	PointerCWCodePrefix          = []byte{0x16}
	PointerReverseRegistryPrefix = []byte{0x17}

	AnteSurplusPrefix  = []byte{0x18} // transient
	DeferredInfoPrefix = []byte{0x19} // transient

	LegacyBlockBloomCutoffHeightKey = []byte{0x1a}
	BaseFeePerGasPrefix             = []byte{0x1b}
	NextBaseFeePerGasPrefix         = []byte{0x1c}
)

var (
	PointerERC20NativePrefix   = []byte{0x0}
	PointerERC20CW20Prefix     = []byte{0x1}
	PointerERC721CW721Prefix   = []byte{0x2}
	PointerCW20ERC20Prefix     = []byte{0x3}
	PointerCW721ERC721Prefix   = []byte{0x4}
	PointerERC1155CW1155Prefix = []byte{0x5}
	PointerCW1155ERC1155Prefix = []byte{0x6}
)

func EVMAddressToEniAddressKey(evmAddress common.Address) []byte {
	return append(EVMAddressToEniAddressKeyPrefix, evmAddress[:]...)
}

func EniAddressToEVMAddressKey(eniAddress sdk.AccAddress) []byte {
	return append(EniAddressToEVMAddressKeyPrefix, eniAddress...)
}

func StateKey(evmAddress common.Address) []byte {
	return append(StateKeyPrefix, evmAddress[:]...)
}

func ReceiptKey(txHash common.Hash) []byte {
	return append(ReceiptKeyPrefix, txHash[:]...)
}

func BlockBloomKey(height int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(height))
	return append(BlockBloomPrefix, bz...)
}

func TxHashesKey(height int64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(height))
	return append(TxHashesPrefix, bz...)
}

func PointerERC20NativeKey(token string) []byte {
	return append(
		append(PointerRegistryPrefix, PointerERC20NativePrefix...),
		[]byte(token)...,
	)
}

func PointerERC20CW20Key(cw20Address string) []byte {
	return append(
		append(PointerRegistryPrefix, PointerERC20CW20Prefix...),
		[]byte(cw20Address)...,
	)
}

func PointerERC721CW721Key(cw721Address string) []byte {
	return append(
		append(PointerRegistryPrefix, PointerERC721CW721Prefix...),
		[]byte(cw721Address)...,
	)
}

func PointerERC1155CW1155Key(cw1155Address string) []byte {
	return append(
		append(PointerRegistryPrefix, PointerERC1155CW1155Prefix...),
		[]byte(cw1155Address)...,
	)
}

func PointerCW20ERC20Key(erc20Addr common.Address) []byte {
	return append(
		append(PointerRegistryPrefix, PointerCW20ERC20Prefix...),
		erc20Addr[:]...,
	)
}

func PointerCW721ERC721Key(erc721Addr common.Address) []byte {
	return append(
		append(PointerRegistryPrefix, PointerCW721ERC721Prefix...),
		erc721Addr[:]...,
	)
}

func PointerCW1155ERC1155Key(erc1155Addr common.Address) []byte {
	return append(
		append(PointerRegistryPrefix, PointerCW1155ERC1155Prefix...),
		erc1155Addr[:]...,
	)
}

func PointerReverseRegistryKey(addr common.Address) []byte {
	return append(PointerReverseRegistryPrefix, addr[:]...)
}
