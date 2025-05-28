package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

func (k *Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	// check if the module account exists
	moduleAcc := k.accountKeeper.GetModuleAccount(ctx, types.ModuleName)
	if moduleAcc == nil {
		moduleAcc = authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	}
	balances := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if balances.IsZero() {
		k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
	}
	k.SetParams(ctx, genState.Params)

	eniAddrFc := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName) // feeCollector == coinbase
	k.SetAddressMapping(ctx, eniAddrFc, GetCoinbaseAddress())

	// The realization of abandonment
	//for _, addr := range genState.AddressAssociations {
	//	k.SetAddressMapping(ctx, sdk.MustAccAddressFromBech32(addr.EniAddress), common.HexToAddress(addr.EthAddress))
	//}

}
