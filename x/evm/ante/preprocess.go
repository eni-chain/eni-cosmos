package ante

import (
	"fmt"
	"math"
	"math/big"

	cosmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmtypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/utils/helpers"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core"

	sdkerrors "cosmossdk.io/errors"
	"github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	coserrors "github.com/cosmos/cosmos-sdk/types/errors"
	sdkTypeerr "github.com/cosmos/cosmos-sdk/types/errors"
	accountkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	//authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	//banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/utils"
	"github.com/cosmos/cosmos-sdk/utils/metrics"
	"github.com/cosmos/cosmos-sdk/x/evm/derived"
	evmkeeper "github.com/cosmos/cosmos-sdk/x/evm/keeper"
	evmtypes "github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types/ethtx"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

const EVMAssociatePriority = math.MaxInt64 - 101

// Accounts need to have at least 1Eni to force association. Note that account won't be charged.
const BalanceThreshold uint64 = 1000000

var BigBalanceThreshold *big.Int = new(big.Int).SetUint64(BalanceThreshold)
var BigBalanceThresholdMinus1 *big.Int = new(big.Int).SetUint64(BalanceThreshold - 1)

var SignerMap = map[derived.SignerVersion]func(*big.Int) ethtypes.Signer{
	derived.London: ethtypes.NewLondonSigner,
	derived.Cancun: ethtypes.NewCancunSigner,
}
var AllowedTxTypes = map[derived.SignerVersion][]uint8{
	derived.London: {ethtypes.LegacyTxType, ethtypes.AccessListTxType, ethtypes.DynamicFeeTxType},
	derived.Cancun: {ethtypes.LegacyTxType, ethtypes.AccessListTxType, ethtypes.DynamicFeeTxType, ethtypes.BlobTxType},
}

type EVMPreprocessDecorator struct {
	evmKeeper       *evmkeeper.Keeper
	accountKeeper   *accountkeeper.AccountKeeper
	latestCtxGetter func() sdk.Context // should be read-only
}

func NewEVMPreprocessDecorator(evmKeeper *evmkeeper.Keeper, accountKeeper *accountkeeper.AccountKeeper, latestCtxGetter func() sdk.Context) *EVMPreprocessDecorator {
	return &EVMPreprocessDecorator{evmKeeper: evmKeeper,
		accountKeeper:   accountKeeper,
		latestCtxGetter: latestCtxGetter}
}

//nolint:revive
func (p *EVMPreprocessDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	msg := evmtypes.MustGetEVMTransactionMessage(tx)

	txData, err := evmtypes.UnpackTxData(msg.Data)
	if err != nil {
		return ctx, err
	}

	if err := Preprocess(ctx, msg, txData); err != nil {
		return ctx, err
	}

	//// use infinite gas meter for EVM transaction because EVM handles gas checking from within
	//ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	//
	//derived := msg.Derived
	//eniAddr := derived.SenderEniAddr
	//evmAddr := derived.SenderEVMAddr
	//ctx.EventManager().EmitEvent(sdk.NewEvent(evmtypes.EventTypeSigner,
	//	sdk.NewAttribute(evmtypes.AttributeKeyEvmAddress, evmAddr.Hex()),
	//	sdk.NewAttribute(evmtypes.AttributeKeyEniAddress, eniAddr.String())))
	//pubkey := derived.PubKey
	//isAssociateTx := derived.IsAssociate
	//associateHelper := helpers.NewAssociationHelper(p.evmKeeper, p.evmKeeper.BankKeeper(), p.accountKeeper)
	////_, isAssociated := p.evmKeeper.GetEVMAddress(ctx, eniAddr)
	//
	//isAssociated := false
	//if isAssociateTx && isAssociated {
	//	return ctx, sdkerrors.Wrap(coserrors.ErrInvalidRequest, "account already has association set")
	//} else if isAssociateTx {
	//	// check if the account has enough balance (without charging)
	//	if !p.IsAccountBalancePositive(ctx, eniAddr, evmAddr) {
	//		metrics.IncrementAssociationError("associate_tx_insufficient_funds", evmtypes.NewAssociationMissingErr(eniAddr.String()))
	//		return ctx, sdkerrors.Wrap(coserrors.ErrInsufficientFunds, "account needs to have at least 1 wei to force association")
	//	}
	//	if err := associateHelper.AssociateAddresses(ctx, eniAddr, evmAddr, pubkey); err != nil {
	//		return ctx, err
	//	}
	//
	//	return ctx.WithPriority(EVMAssociatePriority), nil // short-circuit without calling next
	//} else if isAssociated {
	//	// noop; for readability
	//} else {
	//	// not associatedTx and not already associated
	//	// todo after completing the related functions, make detailed modifications. For now, comments will not affect the process.
	//	//if err := associateHelper.AssociateAddresses(ctx, eniAddr, evmAddr, pubkey); err != nil {
	//	//	return ctx, err
	//	//}
	//}

	ethTx := ethtypes.NewTx(txData.AsEthereumData())

	ctx, err = p.AnteHandleBasic(ctx, simulate, nil, msg, ethTx)
	if err != nil {
		return ctx, err
	}
	ctx, err = p.AnteHandleFee(ctx, simulate, txData, msg, ethTx)
	if err != nil {
		return ctx, err
	}
	ctx, err = p.AnteHandleSig(ctx, simulate, nil, msg, ethTx)
	if err != nil {
		return ctx, err
	}

	adjustedGasLimit := p.evmKeeper.GetPriorityNormalizer(ctx).MulInt64(int64(txData.GetGas()))
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(adjustedGasLimit.TruncateInt().Uint64()))

	return next(ctx, tx, simulate)
}

//nolint:revive
func (gl *EVMPreprocessDecorator) AnteHandleBasic(ctx sdk.Context, simulate bool, txData ethtx.TxData, msg *evmtypes.MsgEVMTransaction, etx *ethtypes.Transaction) (sdk.Context, error) {

	if msg.Derived != nil && !gl.evmKeeper.EthBlockTestConfig.Enabled {
		startingNonce := gl.evmKeeper.GetNonce(ctx, msg.Derived.SenderEVMAddr)
		txNonce := etx.Nonce()
		if !ctx.IsCheckTx() && !ctx.IsReCheckTx() && startingNonce == txNonce {
			ctx = ctx.WithDeliverTxCallback(func(callCtx sdk.Context) {
				// bump nonce if it is for some reason not incremented (e.g. ante failure)
				if gl.evmKeeper.GetNonce(callCtx, msg.Derived.SenderEVMAddr) == startingNonce {
					gl.evmKeeper.SetNonce(callCtx, msg.Derived.SenderEVMAddr, startingNonce+1)
				}
			})
		}
	}

	if etx.To() == nil && len(etx.Data()) > params.MaxInitCodeSize {
		return ctx, fmt.Errorf("%w: code size %v, limit %v", core.ErrMaxInitCodeSizeExceeded, len(etx.Data()), params.MaxInitCodeSize)
	}

	if etx.Value().Sign() < 0 {
		return ctx, coserrors.ErrInvalidCoins
	}

	intrGas, err := core.IntrinsicGas(etx.Data(), etx.AccessList(), nil, etx.To() == nil, true, true, true)
	if err != nil {
		return ctx, err
	}
	if etx.Gas() < intrGas {
		return ctx, coserrors.ErrOutOfGas
	}

	if etx.Type() == ethtypes.BlobTxType {
		return ctx, coserrors.ErrUnsupportedTxType
	}

	// Check if gas exceed the limit
	if cp := ctx.ConsensusParams(); cp.Block != nil {
		// If there exists a maximum block gas limit, we must ensure that the tx
		// does not exceed it.
		if cp.Block.MaxGas > 0 && etx.Gas() > uint64(cp.Block.MaxGas) {
			return ctx, sdkerrors.Wrapf(coserrors.ErrOutOfGas, "tx gas limit %d exceeds block max gas %d", etx.Gas(), cp.Block.MaxGas)
		}
	}
	return ctx, nil
}

func (fc *EVMPreprocessDecorator) AnteHandleFee(ctx sdk.Context, simulate bool, txData ethtx.TxData, msg *evmtypes.MsgEVMTransaction, etx *ethtypes.Transaction) (sdk.Context, error) {
	if simulate {
		return ctx, nil
	}

	ver := msg.Derived.Version

	if txData.GetGasFeeCap().Cmp(fc.getBaseFee(ctx)) < 0 {
		return ctx, coserrors.ErrInsufficientFee
	}
	if txData.GetGasFeeCap().Cmp(fc.getMinimumFee(ctx)) < 0 {
		return ctx, coserrors.ErrInsufficientFee
	}
	if txData.GetGasTipCap().Sign() < 0 {
		return ctx, sdkerrors.Wrapf(coserrors.ErrInvalidRequest, "gas fee cap cannot be negative")
	}

	// if EVM version is Cancun or later, and the transaction contains at least one blob, we need to
	// make sure the transaction carries a non-zero blob fee cap.
	if ver >= derived.Cancun && len(txData.GetBlobHashes()) > 0 {
		// For now we are simply assuming excessive blob gas is 0. In the future we might change it to be
		// dynamic based on prior block usage.
		zero := uint64(0)
		if txData.GetBlobFeeCap().Cmp(eip4844.CalcBlobFee(&params.ChainConfig{CancunTime: &zero}, &ethtypes.Header{})) < 0 {
			return ctx, coserrors.ErrInsufficientFee
		}
	}

	balance := fc.evmKeeper.BankKeeper().GetBalance(ctx, sdk.AccAddress(msg.Derived.SenderEVMAddr[:]), fc.evmKeeper.GetBaseDenom(ctx))
	if balance.Amount.IsZero() {
		return ctx, sdkerrors.Wrap(coserrors.ErrInsufficientFunds, "account needs to have enough balance to cover the transaction fees")
	}
	mgval := new(big.Int).SetUint64(etx.Gas())
	mgval.Mul(mgval, etx.GasPrice())
	if balance.Amount.LT(cosmath.NewIntFromBigInt(mgval)) {
		return ctx, sdkerrors.Wrap(coserrors.ErrInsufficientFunds, "account needs to have enough balance to cover the transaction fees")
	}
	return ctx, nil

	// check if the sender has enough balance to cover fees
	//etx, _ := msg.AsTransaction()

	//emsg := fc.evmKeeper.GetEVMMessage(ctx, etx, msg.Derived.SenderEVMAddr)
	//stateDB := state.NewDBImpl(ctx, fc.evmKeeper, false)
	//gp := fc.evmKeeper.GetGasPool()
	//
	//blockCtx, err := fc.evmKeeper.GetVMBlockContext(ctx, gp)
	//if err != nil {
	//	return ctx, err
	//}
	//
	//cfg := evmtypes.DefaultChainConfig().EthereumConfig(fc.evmKeeper.ChainID(ctx))
	//txCtx := core.NewEVMTxContext(emsg)
	//evmInstance := vm.NewEVM(*blockCtx, stateDB, cfg, vm.Config{})
	//evmInstance.SetTxContext(txCtx)
	//
	////st, err := core.ApplyMessage(evmInstance, emsg, &gp)
	//st := core.NewStateTransition(evmInstance, emsg, &gp, true)
	//// run stateless checks before charging gas (mimicking Geth behavior)
	//if !ctx.IsCheckTx() && !ctx.IsReCheckTx() {
	//	// we don't want to run nonce check here for CheckTx because we have special
	//	// logic for pending nonce during CheckTx in sig.go
	//	if err := st.StatelessChecks(); err != nil {
	//		return ctx, sdkerrors.Wrap(coserrors.ErrWrongSequence, err.Error())
	//	}
	//}
	//if err := st.BuyGas(); err != nil {
	//	return ctx, sdkerrors.Wrap(coserrors.ErrInsufficientFunds, err.Error())
	//}
	//if !ctx.IsCheckTx() && !ctx.IsReCheckTx() {
	//	surplus, err := stateDB.Finalize()
	//	if err != nil {
	//		return ctx, err
	//	}
	//	if err := fc.evmKeeper.AddAnteSurplus(ctx, etx.Hash(), surplus); err != nil {
	//		return ctx, err
	//	}
	//}
	//
	//// calculate the priority by dividing the total fee with the native gas limit (i.e. the effective native gas price)
	//priority := fc.CalculatePriority(ctx, txData)
	//ctx = ctx.WithPriority(priority.Int64())
	//
	//return ctx, nil
}

func (svd *EVMPreprocessDecorator) AnteHandleSig(ctx sdk.Context, simulate bool, txData ethtx.TxData, msg *evmtypes.MsgEVMTransaction, ethTx *ethtypes.Transaction) (sdk.Context, error) {

	evmAddr := msg.Derived.SenderEVMAddr
	nextNonce := svd.evmKeeper.GetNonce(ctx, evmAddr)
	txNonce := ethTx.Nonce()

	// set EVM properties
	ctx = ctx.WithIsEVM(true)
	ctx = ctx.WithEVMNonce(txNonce)
	ctx = ctx.WithEVMSenderAddress(evmAddr.Hex())
	ctx = ctx.WithEVMTxHash(ethTx.Hash().Hex())

	chainID := svd.evmKeeper.ChainID(ctx)
	txChainID := ethTx.ChainId()

	// validate chain ID on the transaction
	switch ethTx.Type() {
	case ethtypes.LegacyTxType:
		// legacy either can have a zero or correct chain ID
		if txChainID.Cmp(big.NewInt(0)) != 0 && txChainID.Cmp(chainID) != 0 {
			ctx.Logger().Debug("chainID mismatch", "txChainID", ethTx.ChainId(), "chainID", chainID)
			return ctx, sdkTypeerr.ErrInvalidChainID
		}
	default:
		// after legacy, all transactions must have the correct chain ID
		if txChainID.Cmp(chainID) != 0 {
			ctx.Logger().Debug("chainID mismatch", "txChainID", ethTx.ChainId(), "chainID", chainID)
			return ctx, sdkTypeerr.ErrInvalidChainID
		}
	}

	if ctx.IsCheckTx() {
		if txNonce < nextNonce {
			return ctx, sdkTypeerr.ErrWrongSequence
		}
		ctx = ctx.WithCheckTxCallback(func(thenCtx sdk.Context, e error) {
			if e != nil {
				return
			}
			txKey := tmtypes.Tx(ctx.TxBytes()).Key()
			svd.evmKeeper.AddPendingNonce(txKey, evmAddr, txNonce, thenCtx.Priority())
		})

		// if the mempool expires a transaction, this handler is invoked
		ctx = ctx.WithExpireTxHandler(func() {
			txKey := tmtypes.Tx(ctx.TxBytes()).Key()
			svd.evmKeeper.RemovePendingNonce(txKey)
		})

		if txNonce > nextNonce {
			// transaction shall be added to mempool as a pending transaction
			ctx = ctx.WithPendingTxChecker(func() abci.PendingTxCheckerResponse {
				latestCtx := svd.latestCtxGetter()

				// nextNonceToBeMined is the next nonce that will be mined
				// geth calls SetNonce(n+1) after a transaction is mined
				nextNonceToBeMined := svd.evmKeeper.GetNonce(latestCtx, evmAddr)

				// nextPendingNonce is the minimum nonce a user may send without stomping on an already-sent
				// nonce, including non-mined or pending transactions
				// If a user skips a nonce [1,2,4], then this will be the value of that hole (e.g., 3)
				nextPendingNonce := svd.evmKeeper.CalculateNextNonce(latestCtx, evmAddr, true)

				if txNonce < nextNonceToBeMined {
					// this nonce has already been mined, we cannot accept it again
					return abci.Rejected
				} else if txNonce < nextPendingNonce {
					// this nonce is allowed to process as it is part of the
					// consecutive nonces from nextNonceToBeMined to nextPendingNonce
					// This logic allows multiple nonces from an account to be processed in a block.
					return abci.Accepted
				}
				return abci.Pending
			})
		}
	} else if txNonce != nextNonce {
		return ctx, sdkTypeerr.ErrWrongSequence
	}

	return ctx, nil
}

//// Called at the end of the ante chain to set gas limit properly
//func (gl EVMPreprocessDecorator) AnteHandleGas(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
//
//	return next(ctx, tx, simulate)
//}

// minimum fee per gas required for a tx to be processed
func (fc EVMPreprocessDecorator) getBaseFee(ctx sdk.Context) *big.Int {
	return fc.evmKeeper.GetCurrBaseFeePerGas(ctx).TruncateInt().BigInt()
}

// lowest allowed fee per gas, base fee will not be lower than this
func (fc EVMPreprocessDecorator) getMinimumFee(ctx sdk.Context) *big.Int {
	return fc.evmKeeper.GetMinimumFeePerGas(ctx).TruncateInt().BigInt()
}

// CalculatePriority returns a priority based on the effective gas price of the transaction
func (fc EVMPreprocessDecorator) CalculatePriority(ctx sdk.Context, txData ethtx.TxData) *big.Int {
	gp := txData.EffectiveGasPrice(utils.Big0)
	if !ctx.IsCheckTx() && !ctx.IsReCheckTx() {
		metrics.HistogramEvmEffectiveGasPrice(gp)
	}
	priority := cosmath.LegacyNewDecFromBigInt(gp).Quo(fc.evmKeeper.GetPriorityNormalizer(ctx)).TruncateInt().BigInt()
	if priority.Cmp(big.NewInt(MaxPriority)) > 0 {
		priority = big.NewInt(MaxPriority)
	}
	return priority
}

func (p *EVMPreprocessDecorator) IsAccountBalancePositive(ctx sdk.Context, eniAddr sdk.AccAddress, evmAddr common.Address) bool {
	baseDenom := p.evmKeeper.GetBaseDenom(ctx)
	if amt := p.evmKeeper.BankKeeper().GetBalance(ctx, eniAddr, baseDenom).Amount; amt.IsPositive() {
		return true
	}
	if amt := p.evmKeeper.BankKeeper().GetBalance(ctx, sdk.AccAddress(evmAddr[:]), baseDenom).Amount; amt.IsPositive() {
		return true
	}
	return false
	//if amt := p.evmKeeper.BankKeeper().GetWeiBalance(ctx, eniAddr); amt.IsPositive() {
	//	return true
	//}
	//return p.evmKeeper.BankKeeper().GetWeiBalance(ctx, sdk.AccAddress(evmAddr[:])).IsPositive()
}

// stateless
func Preprocess(ctx sdk.Context, msgEVMTransaction *evmtypes.MsgEVMTransaction, txData ethtx.TxData) error {
	if msgEVMTransaction.Derived != nil {
		if msgEVMTransaction.Derived.PubKey == nil {
			// this means the message has `Derived` set from the outside, in which case we should reject
			return coserrors.ErrInvalidPubKey
		}
		// already preprocessed
		return nil
	}
	//txData, err := evmtypes.UnpackTxData(msgEVMTransaction.Data)
	//if err != nil {
	//	return err
	//}

	if atx, ok := txData.(*ethtx.AssociateTx); ok {
		V, R, S := atx.GetRawSignatureValues()
		V = new(big.Int).Add(V, utils.Big27)
		// Hash custom message passed in
		customMessageHash := crypto.Keccak256Hash([]byte(atx.CustomMessage))
		evmAddr, eniAddr, pubkey, err := helpers.GetAddresses(V, R, S, customMessageHash)
		if err != nil {
			return err
		}
		msgEVMTransaction.Derived = &derived.Derived{
			SenderEVMAddr: evmAddr,
			SenderEniAddr: eniAddr,
			PubKey:        &secp256k1.PubKey{Key: pubkey.Bytes()},
			Version:       derived.Cancun,
			IsAssociate:   true,
		}
		return nil
	}

	ethTx := ethtypes.NewTx(txData.AsEthereumData())
	chainID := ethTx.ChainId()
	chainCfg := evmtypes.DefaultChainConfig()
	ethCfg := chainCfg.EthereumConfig(chainID)
	version := GetVersion(ctx, ethCfg)
	signer := SignerMap[version](chainID)
	if !isTxTypeAllowed(version, ethTx.Type()) {
		return ethtypes.ErrInvalidChainId
	}

	var txHash common.Hash
	V, R, S := ethTx.RawSignatureValues()
	if ethTx.Protected() {
		V = AdjustV(V, ethTx.Type(), ethCfg.ChainID)
		txHash = signer.Hash(ethTx)
	} else {
		txHash = ethtypes.FrontierSigner{}.Hash(ethTx)
	}
	evmAddr, eniAddr, eniPubkey, err := helpers.GetAddresses(V, R, S, txHash)
	if err != nil {
		return err
	}
	msgEVMTransaction.Sender = eniAddr.String()
	msgEVMTransaction.Derived = &derived.Derived{
		SenderEVMAddr: evmAddr,
		SenderEniAddr: eniAddr,
		PubKey:        &secp256k1.PubKey{Key: eniPubkey.Bytes()},
		Version:       version,
		IsAssociate:   false,
	}
	return nil
}

func PreprocessMsgSender(msgEVMTransaction *evmtypes.MsgEVMTransaction) error {
	if msgEVMTransaction.Derived != nil {
		if msgEVMTransaction.Derived.PubKey == nil {
			// this means the message has `Derived` set from the outside, in which case we should reject
			return coserrors.ErrInvalidPubKey
		}
		// already preprocessed
		return nil
	}
	txData, err := evmtypes.UnpackTxData(msgEVMTransaction.Data)
	if err != nil {
		return err
	}

	if atx, ok := txData.(*ethtx.AssociateTx); ok {
		V, R, S := atx.GetRawSignatureValues()
		V = new(big.Int).Add(V, utils.Big27)
		// Hash custom message passed in
		customMessageHash := crypto.Keccak256Hash([]byte(atx.CustomMessage))
		_, eniAddr, _, err := helpers.GetAddresses(V, R, S, customMessageHash)
		if err != nil {
			return err
		}
		//msgEVMTransaction.Derived = &derived.Derived{
		//	SenderEVMAddr: evmAddr,
		//	SenderEniAddr: eniAddr,
		//	PubKey:        &secp256k1.PubKey{Key: pubkey.Bytes()},
		//	Version:       derived.Cancun,
		//	IsAssociate:   true,
		//}
		msgEVMTransaction.Sender = eniAddr.String()
		return nil
	}

	ethTx := ethtypes.NewTx(txData.AsEthereumData())
	chainID := ethTx.ChainId()
	chainCfg := evmtypes.DefaultChainConfig()
	ethCfg := chainCfg.EthereumConfig(chainID)
	signer := SignerMap[derived.Cancun](chainID)
	if !isTxTypeAllowed(derived.Cancun, ethTx.Type()) {
		return ethtypes.ErrInvalidChainId
	}

	var txHash common.Hash
	V, R, S := ethTx.RawSignatureValues()
	if ethTx.Protected() {
		V = AdjustV(V, ethTx.Type(), ethCfg.ChainID)
		txHash = signer.Hash(ethTx)
	} else {
		txHash = ethtypes.FrontierSigner{}.Hash(ethTx)
	}
	_, eniAddr, _, err := helpers.GetAddresses(V, R, S, txHash)
	if err != nil {
		return err
	}
	msgEVMTransaction.Sender = eniAddr.String()
	return nil
}

//func (p *EVMPreprocessDecorator) AnteDeps(txDeps []sdkacltypes.AccessOperation, tx sdk.Tx, txIndex int, next sdk.AnteDepGenerator) (newTxDeps []sdkacltypes.AccessOperation, err error) {
//	msg := evmtypes.MustGetEVMTransactionMessage(tx)
//	return next(append(txDeps, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_READ,
//		ResourceType:       sdkacltypes.ResourceType_KV_EVM_S2E,
//		IdentifierTemplate: hex.EncodeToString(evmtypes.EniAddressToEVMAddressKey(msg.Derived.SenderEniAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_WRITE,
//		ResourceType:       sdkacltypes.ResourceType_KV_EVM_S2E,
//		IdentifierTemplate: hex.EncodeToString(evmtypes.EniAddressToEVMAddressKey(msg.Derived.SenderEniAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_WRITE,
//		ResourceType:       sdkacltypes.ResourceType_KV_EVM_E2S,
//		IdentifierTemplate: hex.EncodeToString(evmtypes.EVMAddressToEniAddressKey(msg.Derived.SenderEVMAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_READ,
//		ResourceType:       sdkacltypes.ResourceType_KV_BANK_BALANCES,
//		IdentifierTemplate: hex.EncodeToString(banktypes.CreateAccountBalancesPrefix(msg.Derived.SenderEniAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_WRITE,
//		ResourceType:       sdkacltypes.ResourceType_KV_BANK_BALANCES,
//		IdentifierTemplate: hex.EncodeToString(banktypes.CreateAccountBalancesPrefix(msg.Derived.SenderEniAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_READ,
//		ResourceType:       sdkacltypes.ResourceType_KV_BANK_BALANCES,
//		IdentifierTemplate: hex.EncodeToString(banktypes.CreateAccountBalancesPrefix(msg.Derived.SenderEVMAddr[:])),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_WRITE,
//		ResourceType:       sdkacltypes.ResourceType_KV_BANK_BALANCES,
//		IdentifierTemplate: hex.EncodeToString(banktypes.CreateAccountBalancesPrefix(msg.Derived.SenderEVMAddr[:])),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_READ,
//		ResourceType:       sdkacltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
//		IdentifierTemplate: hex.EncodeToString(authtypes.AddressStoreKey(msg.Derived.SenderEniAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_WRITE,
//		ResourceType:       sdkacltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
//		IdentifierTemplate: hex.EncodeToString(authtypes.AddressStoreKey(msg.Derived.SenderEniAddr)),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_READ,
//		ResourceType:       sdkacltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
//		IdentifierTemplate: hex.EncodeToString(authtypes.AddressStoreKey(msg.Derived.SenderEVMAddr[:])),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_WRITE,
//		ResourceType:       sdkacltypes.ResourceType_KV_AUTH_ADDRESS_STORE,
//		IdentifierTemplate: hex.EncodeToString(authtypes.AddressStoreKey(msg.Derived.SenderEVMAddr[:])),
//	}, sdkacltypes.AccessOperation{
//		AccessType:         sdkacltypes.AccessType_READ,
//		ResourceType:       sdkacltypes.ResourceType_KV_EVM_NONCE,
//		IdentifierTemplate: hex.EncodeToString(append(evmtypes.NonceKeyPrefix, msg.Derived.SenderEVMAddr[:]...)),
//	}), tx, txIndex)
//}

func isTxTypeAllowed(version derived.SignerVersion, txType uint8) bool {
	for _, t := range AllowedTxTypes[version] {
		if t == txType {
			return true
		}
	}
	return false
}

func AdjustV(V *big.Int, txType uint8, chainID *big.Int) *big.Int {
	// Non-legacy TX always needs to be bumped by 27
	if txType != ethtypes.LegacyTxType {
		return new(big.Int).Add(V, utils.Big27)
	}

	// legacy TX needs to be adjusted based on chainID
	V = new(big.Int).Sub(V, new(big.Int).Mul(chainID, utils.Big2))
	return V.Sub(V, utils.Big8)
}

func GetVersion(ctx sdk.Context, ethCfg *params.ChainConfig) derived.SignerVersion {
	blockNum := big.NewInt(ctx.BlockHeight())
	ts := uint64(ctx.BlockTime().Unix())
	switch {
	case ethCfg.IsCancun(blockNum, ts):
		return derived.Cancun
	default:
		return derived.London
	}
}

type EVMAddressDecorator struct {
	evmKeeper     *evmkeeper.Keeper
	accountKeeper *accountkeeper.AccountKeeper
}

func NewEVMAddressDecorator(evmKeeper *evmkeeper.Keeper, accountKeeper *accountkeeper.AccountKeeper) *EVMAddressDecorator {
	return &EVMAddressDecorator{evmKeeper: evmKeeper, accountKeeper: accountKeeper}
}

//nolint:revive
func (p *EVMAddressDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.Wrap(coserrors.ErrTxDecode, "invalid tx type")
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return ctx, sdkerrors.Wrap(coserrors.ErrInvalidAddress, "failed to get signers")
	}
	for _, signer := range signers {
		acc := p.accountKeeper.GetAccount(ctx, signer)
		if evmAddr, associated := p.evmKeeper.GetEVMAddress(ctx, signer); associated {
			ctx.EventManager().EmitEvent(sdk.NewEvent(evmtypes.EventTypeSigner,
				sdk.NewAttribute(evmtypes.AttributeKeyEvmAddress, evmAddr.Hex()),
				sdk.NewAttribute(evmtypes.AttributeKeyEniAddress, acc.String())))
			continue
		}

		if acc.GetPubKey() == nil {
			ctx.Logger().Error(fmt.Sprintf("missing pubkey for %s", acc.String()))
			ctx.EventManager().EmitEvent(sdk.NewEvent(evmtypes.EventTypeSigner,
				sdk.NewAttribute(evmtypes.AttributeKeyEniAddress, acc.String())))
			continue
		}
		pk, err := btcec.ParsePubKey(acc.GetPubKey().Bytes(), btcec.S256())
		if err != nil {
			ctx.Logger().Debug(fmt.Sprintf("failed to parse pubkey for %s, likely due to the fact that it isn't on secp256k1 curve", acc.GetPubKey()), "err", err)
			ctx.EventManager().EmitEvent(sdk.NewEvent(evmtypes.EventTypeSigner,
				sdk.NewAttribute(evmtypes.AttributeKeyEniAddress, acc.String())))
			continue
		}
		evmAddr, err := helpers.PubkeyToEVMAddress(pk.SerializeUncompressed())
		if err != nil {
			ctx.Logger().Error(fmt.Sprintf("failed to get EVM address from pubkey due to %s", err))
			ctx.EventManager().EmitEvent(sdk.NewEvent(evmtypes.EventTypeSigner,
				sdk.NewAttribute(evmtypes.AttributeKeyEniAddress, acc.String())))
			continue
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(evmtypes.EventTypeSigner,
			sdk.NewAttribute(evmtypes.AttributeKeyEvmAddress, evmAddr.Hex()),
			sdk.NewAttribute(evmtypes.AttributeKeyEniAddress, acc.String())))
		p.evmKeeper.SetAddressMapping(ctx, signer, evmAddr)
		associationHelper := helpers.NewAssociationHelper(p.evmKeeper, p.evmKeeper.BankKeeper(), p.accountKeeper)
		if err := associationHelper.MigrateBalance(ctx, evmAddr, signer); err != nil {
			ctx.Logger().Error(fmt.Sprintf("failed to migrate EVM address balance (%s) %s", evmAddr.Hex(), err))
			return ctx, err
		}
		if evmtypes.IsTxMsgAssociate(tx) {
			// check if there is non-zero balance
			if !p.evmKeeper.BankKeeper().GetBalance(ctx, signer, p.evmKeeper.GetBaseDenom(ctx)).IsPositive() { //&& !p.evmKeeper.BankKeeper().GetWeiBalance(ctx, signer).IsPositive()
				return ctx, sdkerrors.Wrap(coserrors.ErrInsufficientFunds, "account needs to have at least 1 wei to force association")
			}
		}
	}
	return next(ctx, tx, simulate)
}
