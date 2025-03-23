package keeper

import (
	"context"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/cosmos-sdk/x/evm/types"
)

type AccountKeeperMutex struct {
	ak    evmtypes.AccountKeeper
	mutex *sync.RWMutex
}

var _ evmtypes.AccountKeeper = (*AccountKeeperMutex)(nil)

func NewAccountKeeperMutex(ak evmtypes.AccountKeeper) *AccountKeeperMutex {
	return &AccountKeeperMutex{
		ak:    ak,
		mutex: &sync.RWMutex{},
	}

}

func (a *AccountKeeperMutex) NewAccountWithAddress(ctx context.Context, address sdk.AccAddress) sdk.AccountI {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.ak.NewAccountWithAddress(ctx, address)
}

func (a *AccountKeeperMutex) HasAccount(ctx context.Context, address sdk.AccAddress) bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.ak.HasAccount(ctx, address)
}

func (a *AccountKeeperMutex) GetAccount(ctx context.Context, address sdk.AccAddress) sdk.AccountI {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.ak.GetAccount(ctx, address)
}

func (a *AccountKeeperMutex) SetAccount(ctx context.Context, i sdk.AccountI) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.ak.SetAccount(ctx, i)
}

func (a *AccountKeeperMutex) SetModuleAccount(ctx context.Context, macc sdk.ModuleAccountI) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.ak.SetModuleAccount(ctx, macc)
}

func (a *AccountKeeperMutex) GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.ak.GetModuleAccount(ctx, moduleName)
}

func (a *AccountKeeperMutex) GetModuleAddress(moduleName string) sdk.AccAddress {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.ak.GetModuleAddress(moduleName)
}
