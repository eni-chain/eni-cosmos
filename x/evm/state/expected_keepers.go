package state

import (
	"math/big"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	evmtypes "github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
)

type EVMKeeper interface {
	PrefixStore(sdk.Context, []byte) storetypes.KVStore
	PurgePrefix(sdk.Context, []byte)
	GetEniAddress(sdk.Context, common.Address) (sdk.AccAddress, bool)
	GetEniAddressOrDefault(ctx sdk.Context, evmAddress common.Address) sdk.AccAddress
	BankKeeper() bankkeeper.Keeper
	GetBaseDenom(sdk.Context) string
	DeleteAddressMapping(sdk.Context, sdk.AccAddress, common.Address)
	GetCode(sdk.Context, common.Address) []byte
	SetCode(sdk.Context, common.Address, []byte)
	GetCodeHash(sdk.Context, common.Address) common.Hash
	GetCodeSize(sdk.Context, common.Address) int
	GetState(sdk.Context, common.Address, common.Hash) common.Hash
	SetState(sdk.Context, common.Address, common.Hash, common.Hash)
	AccountKeeper() evmtypes.AccountKeeper
	GetFeeCollectorAddress(sdk.Context) (common.Address, error)
	GetNonce(sdk.Context, common.Address) uint64
	SetNonce(sdk.Context, common.Address, uint64)
	GetBalance(ctx sdk.Context, addr sdk.AccAddress) *big.Int
}
