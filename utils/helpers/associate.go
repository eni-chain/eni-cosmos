package helpers

import (
	"context"

	"cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	//SendCoinsAndWei(ctx sdk.Context, from sdk.AccAddress, to sdk.AccAddress, amt math.Int, wei math.Int) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	//GetWeiBalance(ctx sdk.Context, addr sdk.AccAddress) math.Int
	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	GetSupply(ctx context.Context, denom string) sdk.Coin
	LockedCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

type EVMKeeper interface {
	GetEniAddress(sdk.Context, common.Address) (sdk.AccAddress, bool)
	GetEniAddressOrDefault(ctx sdk.Context, evmAddress common.Address) sdk.AccAddress // only used for getting precompile Eni addresses
	GetEVMAddress(sdk.Context, sdk.AccAddress) (common.Address, bool)
	SetAddressMapping(sdk.Context, sdk.AccAddress, common.Address)
	GetCodeHash(sdk.Context, common.Address) common.Hash
	GetPriorityNormalizer(ctx sdk.Context) math.LegacyDec
	GetBaseDenom(ctx sdk.Context) string
	SetERC20NativePointer(ctx sdk.Context, token string, addr common.Address) error
	GetERC20NativePointer(ctx sdk.Context, token string) (addr common.Address, version uint16, exists bool)
	//SetERC20CW20Pointer(ctx sdk.Context, cw20Address string, addr common.Address) error
	//GetERC20CW20Pointer(ctx sdk.Context, cw20Address string) (addr common.Address, version uint16, exists bool)
	//SetERC721CW721Pointer(ctx sdk.Context, cw721Address string, addr common.Address) error
	//GetERC721CW721Pointer(ctx sdk.Context, cw721Address string) (addr common.Address, version uint16, exists bool)
	//SetERC1155CW1155Pointer(ctx sdk.Context, cw1155Address string, addr common.Address) error
	//GetERC1155CW1155Pointer(ctx sdk.Context, cw1155Address string) (addr common.Address, version uint16, exists bool)
	SetCode(ctx sdk.Context, addr common.Address, code []byte)
	UpsertERCNativePointer(
		ctx sdk.Context, evm *vm.EVM, token string, metadata utils.ERCMetadata,
	) (contractAddr common.Address, err error)
	//UpsertERCCW20Pointer(
	//	ctx sdk.Context, evm *vm.EVM, cw20Addr string, metadata utils.ERCMetadata,
	//) (contractAddr common.Address, err error)
	//UpsertERCCW721Pointer(
	//	ctx sdk.Context, evm *vm.EVM, cw721Addr string, metadata utils.ERCMetadata,
	//) (contractAddr common.Address, err error)
	//UpsertERCCW1155Pointer(
	//	ctx sdk.Context, evm *vm.EVM, cw1155Addr string, metadata utils.ERCMetadata,
	//) (contractAddr common.Address, err error)
	GetEVMGasLimitFromCtx(ctx sdk.Context) uint64
	GetCosmosGasLimitFromEVMGas(ctx sdk.Context, evmGas uint64) uint64
}

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	SetAccount(ctx context.Context, acc sdk.AccountI)
	RemoveAccount(ctx context.Context, acc sdk.AccountI)
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}
type AssociationHelper struct {
	evmKeeper     EVMKeeper
	bankKeeper    BankKeeper
	accountKeeper AccountKeeper
}

func NewAssociationHelper(evmKeeper EVMKeeper, bankKeeper BankKeeper, accountKeeper AccountKeeper) *AssociationHelper {
	return &AssociationHelper{evmKeeper: evmKeeper, bankKeeper: bankKeeper, accountKeeper: accountKeeper}
}

func (p AssociationHelper) AssociateAddresses(ctx sdk.Context, eniAddr sdk.AccAddress, evmAddr common.Address, pubkey cryptotypes.PubKey) error {
	p.evmKeeper.SetAddressMapping(ctx, eniAddr, evmAddr)
	if acc := p.accountKeeper.GetAccount(ctx, eniAddr); acc.GetPubKey() == nil {
		if err := acc.SetPubKey(pubkey); err != nil {
			return err
		}
		p.accountKeeper.SetAccount(ctx, acc)
	}
	return p.MigrateBalance(ctx, evmAddr, eniAddr)
}

func (p AssociationHelper) MigrateBalance(ctx sdk.Context, evmAddr common.Address, eniAddr sdk.AccAddress) error {
	castAddr := sdk.AccAddress(evmAddr[:])
	castAddrBalances := p.bankKeeper.SpendableCoins(ctx, castAddr)
	if !castAddrBalances.IsZero() {
		if err := p.bankKeeper.SendCoins(ctx, castAddr, eniAddr, castAddrBalances); err != nil {
			return err
		}
	}
	//todo The corresponding implementation is added later
	//castAddrWei := p.bankKeeper.GetWeiBalance(ctx, castAddr)
	//if !castAddrWei.IsZero() {
	//	if err := p.bankKeeper.SendCoinsAndWei(ctx, castAddr, eniAddr, math.ZeroInt(), castAddrWei); err != nil {
	//		return err
	//	}
	//}
	if p.bankKeeper.LockedCoins(ctx, castAddr).IsZero() {
		//todo new implementation for eni
		if p.accountKeeper.HasAccount(ctx, castAddr) {
			p.accountKeeper.RemoveAccount(ctx, authtypes.NewBaseAccountWithAddress(castAddr))
		}
	}
	return nil
}
