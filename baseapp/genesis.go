package baseapp

import (
	"errors"

	"github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/core/genesis"
)

var _ genesis.TxHandler = (*BaseApp)(nil)

// ExecuteGenesisTx implements genesis.GenesisState from
// cosmossdk.io/core/genesis to set initial state in genesis
func (app BaseApp) ExecuteGenesisTx(tx []byte) error {
	ctx := app.getContextForTx(execModeFinalize, tx)
	res := app.deliverTx(ctx, tx)

	if res.Code != types.CodeTypeOK {
		return errors.New(res.Log)
	}

	return nil
}
