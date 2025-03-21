package baseapp

import (
	"errors"

	"github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/core/genesis"
)

var _ genesis.TxHandler = (*BaseApp)(nil)

// ExecuteGenesisTx implements genesis.GenesisState from
// cosmossdk.io/core/genesis to set initial state in genesis
func (ba BaseApp) ExecuteGenesisTx(tx []byte) error {
	ctx := ba.getContextForTx(execModeFinalize, tx)
	res := ba.deliverTx(ctx, tx)

	if res.Code != types.CodeTypeOK {
		return errors.New(res.Log)
	}

	return nil
}
