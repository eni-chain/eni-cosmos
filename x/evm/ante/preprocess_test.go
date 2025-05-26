package ante_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	"cosmossdk.io/math"
	"google.golang.org/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//sdkacltypes "github.com/cosmos/cosmos-sdk/types/accesscontrol"
	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/ante"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestPreprocessAnteHandler(t *testing.T) {
	app, ctx := testkeeper.NewMockApp(t, false)
	k := app.GetEVMKeeper()
	handler := ante.NewEVMPreprocessDecorator(k, k.AccountKeeper(), func() sdk.Context {
		return app.NewContext(false)
	})
	privKey := testkeeper.MockPrivateKey()
	eniAddr, evmAddr := testkeeper.PrivateKeyToAddresses(privKey)
	require.Nil(t, k.BankKeeper().AddCoins(ctx, evmAddr[:], sdk.NewCoins(sdk.NewCoin("ueni", math.NewInt(1000000000*30001)))))

	testPrivHex := hex.EncodeToString(privKey.Bytes())
	key, _ := crypto.HexToECDSA(testPrivHex)
	to := new(common.Address)
	copy(to[:], "0x1234567890abcdef1234567890abcdef12345678")
	txData := ethtypes.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(1000000000),
		Gas:      30000,
		To:       to,
		Value:    big.NewInt(1000),
		Data:     []byte("abc"),
	}
	chainID := k.ChainID(ctx)
	chainCfg := types.DefaultChainConfig()
	ethCfg := chainCfg.EthereumConfig(chainID)
	blockNum := big.NewInt(ctx.BlockHeight())
	signer := ethtypes.MakeSigner(ethCfg, blockNum, uint64(ctx.BlockTime().Unix()))
	tx, err := ethtypes.SignTx(ethtypes.NewTx(&txData), signer, key)
	require.Nil(t, err)
	typedTx, err := ethtx.NewLegacyTx(tx)
	require.Nil(t, err)
	msg, err := types.NewMsgEVMTransaction(typedTx)
	require.Nil(t, err)
	ctx, err = handler.AnteHandle(ctx, mockTx{msgs: []sdk.Msg{msg}}, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.Nil(t, err)

	require.Equal(t, sdk.AccAddress(privKey.PubKey().Address()), msg.Derived.SenderEniAddr)
	require.Equal(t, math.NewInt(1000000000).String(), k.BankKeeper().GetBalance(ctx, eniAddr, "ueni").Amount.String())
	// consume gas fee
	require.Equal(t, math.NewInt(1000000000).String(), k.BankKeeper().GetBalance(ctx, evmAddr[:], "ueni").Amount.String())
}

type mockTx struct {
	msgs    []sdk.Msg
	signers []sdk.AccAddress
	msgsV2  []proto.Message
}

func (tx mockTx) GetMsgsV2() ([]proto.Message, error) {
	return tx.msgsV2, nil
}

func (tx mockTx) GetMsgs() []sdk.Msg   { return tx.msgs }
func (tx mockTx) ValidateBasic() error { return nil }
