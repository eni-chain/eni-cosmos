package types

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/gogoproto/proto"

	"cosmossdk.io/core/comet"
	"cosmossdk.io/core/header"
	"cosmossdk.io/log"
	"cosmossdk.io/store/gaskv"
	storetypes "cosmossdk.io/store/types"
)

// ExecMode defines the execution mode which can be set on a Context.
type ExecMode uint8

// All possible execution modes.
const (
	ExecModeCheck ExecMode = iota
	ExecModeReCheck
	ExecModeSimulate
	ExecModePrepareProposal
	ExecModeProcessProposal
	ExecModeVoteExtension
	ExecModeVerifyVoteExtension
	ExecModeFinalize
)

/*
Context is an immutable object contains all information needed to
process a request.

It contains a context.Context object inside if you want to use that,
but please do not over-use it. We try to keep all data structured
and standard additions here would be better just to add to the Context struct
*/
type Context struct {
	ctx     context.Context
	baseCtx context.Context
	ms      storetypes.MultiStore
	// Deprecated: Use HeaderService for height, time, and chainID and CometService for the rest
	header cmtproto.Header
	// Deprecated: Use HeaderService for hash
	headerHash []byte
	// Deprecated: Use HeaderService for chainID and CometService for the rest
	chainID              string
	txBytes              []byte
	logger               log.Logger
	voteInfo             []abci.VoteInfo
	gasMeter             storetypes.GasMeter
	blockGasMeter        storetypes.GasMeter
	checkTx              bool
	recheckTx            bool // if recheckTx == true, then checkTx must also be true
	sigverifyTx          bool // when run simulation, because the private key corresponding to the account in the genesis.json randomly generated, we must skip the sigverify.
	execMode             ExecMode
	minGasPrice          DecCoins
	consParams           cmtproto.ConsensusParams
	eventManager         EventManagerI
	priority             int64 // The tx priority, only relevant in CheckTx
	kvGasConfig          storetypes.GasConfig
	transientKVGasConfig storetypes.GasConfig
	streamingManager     storetypes.StreamingManager
	cometInfo            comet.BlockInfo
	headerInfo           header.Info

	txSum             [32]byte
	occEnabled        bool
	evmEventManager   *EVMEventManager
	pendingTxChecker  PendingTxChecker     // Checker for pending transaction, only relevant in CheckTx
	checkTxCallback   func(Context, error) // callback to make at the end of CheckTx. Input param is the error (nil-able) of `runMsgs`
	deliverTxCallback func(Context)        // callback to make at the end of DeliverTx.
	expireTxHandler   func()               // callback that the mempool invokes when a tx is expired

	// EVM properties
	evm                                 bool   // EVM transaction flag
	evmNonce                            uint64 // EVM Transaction nonce
	evmSenderAddress                    string // EVM Sender address
	evmTxHash                           string // EVM TX hash
	evmVmError                          string // EVM VM error during execution
	evmPrecompileCalledFromDelegateCall bool   // EVM precompile is called from a delegate call

	messageIndex int // Used to track current message being processed
	txIndex      int

	traceSpanContext context.Context
}

// Proposed rename, not done to avoid API breakage
type Request = Context

// Read-only accessors
func (c Context) Context() context.Context                      { return c.baseCtx }
func (c Context) MultiStore() storetypes.MultiStore             { return c.ms }
func (c Context) BlockHeight() int64                            { return c.header.Height }
func (c Context) BlockTime() time.Time                          { return c.header.Time }
func (c Context) ChainID() string                               { return c.chainID }
func (c Context) TxBytes() []byte                               { return c.txBytes }
func (c Context) Logger() log.Logger                            { return c.logger }
func (c Context) VoteInfos() []abci.VoteInfo                    { return c.voteInfo }
func (c Context) GasMeter() storetypes.GasMeter                 { return c.gasMeter }
func (c Context) BlockGasMeter() storetypes.GasMeter            { return c.blockGasMeter }
func (c Context) IsCheckTx() bool                               { return c.checkTx }
func (c Context) IsReCheckTx() bool                             { return c.recheckTx }
func (c Context) IsSigverifyTx() bool                           { return c.sigverifyTx }
func (c Context) ExecMode() ExecMode                            { return c.execMode }
func (c Context) MinGasPrices() DecCoins                        { return c.minGasPrice }
func (c Context) EventManager() EventManagerI                   { return c.eventManager }
func (c Context) Priority() int64                               { return c.priority }
func (c Context) KVGasConfig() storetypes.GasConfig             { return c.kvGasConfig }
func (c Context) TransientKVGasConfig() storetypes.GasConfig    { return c.transientKVGasConfig }
func (c Context) StreamingManager() storetypes.StreamingManager { return c.streamingManager }
func (c Context) CometInfo() comet.BlockInfo                    { return c.cometInfo }
func (c Context) HeaderInfo() header.Info                       { return c.headerInfo }

// clone the header before returning
func (c Context) BlockHeader() cmtproto.Header {
	msg := proto.Clone(&c.header).(*cmtproto.Header)
	return *msg
}

// HeaderHash returns a copy of the header hash obtained during abci.RequestBeginBlock
func (c Context) HeaderHash() []byte {
	hash := make([]byte, len(c.headerHash))
	copy(hash, c.headerHash)
	return hash
}

func (c Context) ConsensusParams() cmtproto.ConsensusParams {
	return c.consParams
}

func (c Context) Deadline() (deadline time.Time, ok bool) {
	return c.baseCtx.Deadline()
}

func (c Context) Done() <-chan struct{} {
	return c.baseCtx.Done()
}

func (c Context) Err() error {
	return c.baseCtx.Err()
}

func (c Context) TxSum() [32]byte {
	return c.txSum
}

// OccEnabled returns the occEnabled value.
func (c Context) OccEnabled() bool {
	return c.occEnabled
}

// EvmEventManager returns the evmEventManager value.
func (c Context) EvmEventManager() *EVMEventManager {
	return c.evmEventManager
}

// PendingTxChecker returns the pendingTxChecker value.
func (c Context) PendingTxChecker() PendingTxChecker {
	return c.pendingTxChecker
}

// CheckTxCallback returns the checkTxCallback value.
func (c Context) CheckTxCallback() func(Context, error) {
	return c.checkTxCallback
}

// DeliverTxCallback returns the deliverTxCallback value.
func (c Context) DeliverTxCallback() func(Context) {
	return c.deliverTxCallback
}

// ExpireTxHandler returns the expireTxHandler value.
func (c Context) ExpireTxHandler() func() {
	return c.expireTxHandler
}

// Evm returns the evm flag value.
func (c Context) Evm() bool {
	return c.evm
}

// EvmNonce returns the evmNonce value.
func (c Context) EvmNonce() uint64 {
	return c.evmNonce
}

// EvmSenderAddress returns the evmSenderAddress value.
func (c Context) EvmSenderAddress() string {
	return c.evmSenderAddress
}

// EvmTxHash returns the evmTxHash value.
func (c Context) EvmTxHash() string {
	return c.evmTxHash
}

// EvmVmError returns the evmVmError value.
func (c Context) EvmVmError() string {
	return c.evmVmError
}

// EvmPrecompileCalledFromDelegateCall returns the evmPrecompileCalledFromDelegateCall value.
func (c Context) EvmPrecompileCalledFromDelegateCall() bool {
	return c.evmPrecompileCalledFromDelegateCall
}

// MessageIndex returns the messageIndex value.
func (c Context) MessageIndex() int {
	return c.messageIndex
}

// TxIndex returns the txIndex value.
func (c Context) TxIndex() int {
	return c.txIndex
}

// TraceSpanContext returns the traceSpanContext value.
func (c Context) TraceSpanContext() context.Context {
	return c.traceSpanContext
}

// create a new context
func NewContext(ms storetypes.MultiStore, header cmtproto.Header, isCheckTx bool, logger log.Logger) Context {
	// https://github.com/gogo/protobuf/issues/519
	header.Time = header.Time.UTC()
	return Context{
		baseCtx:         context.Background(),
		ms:              ms,
		header:          header,
		chainID:         header.ChainID,
		checkTx:         isCheckTx,
		sigverifyTx:     true,
		logger:          logger,
		gasMeter:        storetypes.NewInfiniteGasMeter(),
		minGasPrice:     DecCoins{},
		eventManager:    NewEventManager(),
		evmEventManager: NewEVMEventManager(),

		kvGasConfig:          storetypes.KVGasConfig(),
		transientKVGasConfig: storetypes.TransientGasConfig(),
	}
}

// WithContext returns a Context with an updated context.Context.
func (c Context) WithContext(ctx context.Context) Context {
	c.baseCtx = ctx
	return c
}

// WithMultiStore returns a Context with an updated MultiStore.
func (c Context) WithMultiStore(ms storetypes.MultiStore) Context {
	c.ms = ms
	return c
}

// WithBlockHeader returns a Context with an updated CometBFT block header in UTC time.
func (c Context) WithBlockHeader(header cmtproto.Header) Context {
	// https://github.com/gogo/protobuf/issues/519
	header.Time = header.Time.UTC()
	c.header = header
	return c
}

// WithHeaderHash returns a Context with an updated CometBFT block header hash.
func (c Context) WithHeaderHash(hash []byte) Context {
	temp := make([]byte, len(hash))
	copy(temp, hash)

	c.headerHash = temp
	return c
}

// WithBlockTime returns a Context with an updated CometBFT block header time in UTC with no monotonic component.
// Stripping the monotonic component is for time equality.
func (c Context) WithBlockTime(newTime time.Time) Context {
	newHeader := c.BlockHeader()
	// https://github.com/gogo/protobuf/issues/519
	newHeader.Time = newTime.Round(0).UTC()
	return c.WithBlockHeader(newHeader)
}

// WithProposer returns a Context with an updated proposer consensus address.
func (c Context) WithProposer(addr ConsAddress) Context {
	newHeader := c.BlockHeader()
	newHeader.ProposerAddress = addr.Bytes()
	return c.WithBlockHeader(newHeader)
}

// WithBlockHeight returns a Context with an updated block height.
func (c Context) WithBlockHeight(height int64) Context {
	newHeader := c.BlockHeader()
	newHeader.Height = height
	return c.WithBlockHeader(newHeader)
}

// WithChainID returns a Context with an updated chain identifier.
func (c Context) WithChainID(chainID string) Context {
	c.chainID = chainID
	return c
}

// WithTxBytes returns a Context with an updated txBytes.
func (c Context) WithTxBytes(txBytes []byte) Context {
	c.txBytes = txBytes
	return c
}

// WithLogger returns a Context with an updated logger.
func (c Context) WithLogger(logger log.Logger) Context {
	c.logger = logger
	return c
}

// WithVoteInfos returns a Context with an updated consensus VoteInfo.
func (c Context) WithVoteInfos(voteInfo []abci.VoteInfo) Context {
	c.voteInfo = voteInfo
	return c
}

// WithGasMeter returns a Context with an updated transaction GasMeter.
func (c Context) WithGasMeter(meter storetypes.GasMeter) Context {
	c.gasMeter = meter
	return c
}

// WithBlockGasMeter returns a Context with an updated block GasMeter
func (c Context) WithBlockGasMeter(meter storetypes.GasMeter) Context {
	c.blockGasMeter = meter
	return c
}

// WithKVGasConfig returns a Context with an updated gas configuration for
// the KVStore
func (c Context) WithKVGasConfig(gasConfig storetypes.GasConfig) Context {
	c.kvGasConfig = gasConfig
	return c
}

// WithTransientKVGasConfig returns a Context with an updated gas configuration for
// the transient KVStore
func (c Context) WithTransientKVGasConfig(gasConfig storetypes.GasConfig) Context {
	c.transientKVGasConfig = gasConfig
	return c
}

// WithIsCheckTx enables or disables CheckTx value for verifying transactions and returns an updated Context
func (c Context) WithIsCheckTx(isCheckTx bool) Context {
	c.checkTx = isCheckTx
	c.execMode = ExecModeCheck
	return c
}

// WithIsRecheckTx called with true will also set true on checkTx in order to
// enforce the invariant that if recheckTx = true then checkTx = true as well.
func (c Context) WithIsReCheckTx(isRecheckTx bool) Context {
	if isRecheckTx {
		c.checkTx = true
	}
	c.recheckTx = isRecheckTx
	c.execMode = ExecModeReCheck
	return c
}

// WithIsSigverifyTx called with true will sigverify in auth module
func (c Context) WithIsSigverifyTx(isSigverifyTx bool) Context {
	c.sigverifyTx = isSigverifyTx
	return c
}

// WithExecMode returns a Context with an updated ExecMode.
func (c Context) WithExecMode(m ExecMode) Context {
	c.execMode = m
	return c
}

// WithMinGasPrices returns a Context with an updated minimum gas price value
func (c Context) WithMinGasPrices(gasPrices DecCoins) Context {
	c.minGasPrice = gasPrices
	return c
}

// WithConsensusParams returns a Context with an updated consensus params
func (c Context) WithConsensusParams(params cmtproto.ConsensusParams) Context {
	c.consParams = params
	return c
}

// WithEventManager returns a Context with an updated event manager
func (c Context) WithEventManager(em EventManagerI) Context {
	c.eventManager = em
	return c
}

// WithPriority returns a Context with an updated tx priority
func (c Context) WithPriority(p int64) Context {
	c.priority = p
	return c
}

// WithStreamingManager returns a Context with an updated streaming manager
func (c Context) WithStreamingManager(sm storetypes.StreamingManager) Context {
	c.streamingManager = sm
	return c
}

// WithCometInfo returns a Context with an updated comet info
func (c Context) WithCometInfo(cometInfo comet.BlockInfo) Context {
	c.cometInfo = cometInfo
	return c
}

// WithHeaderInfo returns a Context with an updated header info
func (c Context) WithHeaderInfo(headerInfo header.Info) Context {
	// Settime to UTC
	headerInfo.Time = headerInfo.Time.UTC()
	c.headerInfo = headerInfo
	return c
}

func (c Context) WithTxSum(txSum [32]byte) Context {
	c.txSum = txSum
	return c
}

// WithOccEnabled returns a Context with an updated occEnabled value.
func (c Context) WithOccEnabled(occEnabled bool) Context {
	c.occEnabled = occEnabled
	return c
}

// WithEvmEventManager returns a Context with an updated evmEventManager value.
func (c Context) WithEvmEventManager(evmEventManager *EVMEventManager) Context {
	c.evmEventManager = evmEventManager
	return c
}

// WithPendingTxChecker returns a Context with an updated pendingTxChecker value.
func (c Context) WithPendingTxChecker(pendingTxChecker PendingTxChecker) Context {
	c.pendingTxChecker = pendingTxChecker
	return c
}

// WithCheckTxCallback returns a Context with an updated checkTxCallback value.
func (c Context) WithCheckTxCallback(callback func(Context, error)) Context {
	c.checkTxCallback = callback
	return c
}

// WithDeliverTxCallback returns a Context with an updated deliverTxCallback value.
func (c Context) WithDeliverTxCallback(callback func(Context)) Context {
	c.deliverTxCallback = callback
	return c
}

// WithExpireTxHandler returns a Context with an updated expireTxHandler value.
func (c Context) WithExpireTxHandler(handler func()) Context {
	c.expireTxHandler = handler
	return c
}

// WithEvm returns a Context with an updated evm flag value.
func (c Context) WithEvm(evm bool) Context {
	c.evm = evm
	return c
}

// WithEvmNonce returns a Context with an updated evmNonce value.
func (c Context) WithEvmNonce(nonce uint64) Context {
	c.evmNonce = nonce
	return c
}

// WithEvmSenderAddress returns a Context with an updated evmSenderAddress value.
func (c Context) WithEvmSenderAddress(senderAddress string) Context {
	c.evmSenderAddress = senderAddress
	return c
}

// WithEvmTxHash returns a Context with an updated evmTxHash value.
func (c Context) WithEvmTxHash(txHash string) Context {
	c.evmTxHash = txHash
	return c
}

// WithEvmVmError returns a Context with an updated evmVmError value.
func (c Context) WithEvmVmError(vmError string) Context {
	c.evmVmError = vmError
	return c
}

// WithEvmPrecompileCalledFromDelegateCall returns a Context with an updated evmPrecompileCalledFromDelegateCall value.
func (c Context) WithEvmPrecompileCalledFromDelegateCall(called bool) Context {
	c.evmPrecompileCalledFromDelegateCall = called
	return c
}

// WithMessageIndex returns a Context with an updated messageIndex value.
func (c Context) WithMessageIndex(index int) Context {
	c.messageIndex = index
	return c
}

// WithTxIndex returns a Context with an updated txIndex value.
func (c Context) WithTxIndex(index int) Context {
	c.txIndex = index
	return c
}

// WithTraceSpanContext returns a Context with an updated traceSpanContext value.
func (c Context) WithTraceSpanContext(ctx context.Context) Context {
	c.traceSpanContext = ctx
	return c
}

// TODO: remove???
func (c Context) IsZero() bool {
	return c.ms == nil
}

func (c Context) WithValue(key, value interface{}) Context {
	c.baseCtx = context.WithValue(c.baseCtx, key, value)
	return c
}

func (c Context) Value(key interface{}) interface{} {
	if key == SdkContextKey {
		return c
	}

	return c.baseCtx.Value(key)
}

// ----------------------------------------------------------------------------
// Store / Caching
// ----------------------------------------------------------------------------

// KVStore fetches a KVStore from the MultiStore.
func (c Context) KVStore(key storetypes.StoreKey) storetypes.KVStore {
	return gaskv.NewStore(c.ms.GetKVStore(key), c.gasMeter, c.kvGasConfig)
}

// TransientStore fetches a TransientStore from the MultiStore.
func (c Context) TransientStore(key storetypes.StoreKey) storetypes.KVStore {
	return gaskv.NewStore(c.ms.GetKVStore(key), c.gasMeter, c.transientKVGasConfig)
}

// CacheContext returns a new Context with the multi-store cached and a new
// EventManager. The cached context is written to the context when writeCache
// is called. Note, events are automatically emitted on the parent context's
// EventManager when the caller executes the write.
func (c Context) CacheContext() (cc Context, writeCache func()) {
	cms := c.ms.CacheMultiStore()
	cc = c.WithMultiStore(cms).WithEventManager(NewEventManager())

	writeCache = func() {
		c.EventManager().EmitEvents(cc.EventManager().Events())
		cms.Write()
	}

	return cc, writeCache
}

var (
	_ context.Context    = Context{}
	_ storetypes.Context = Context{}
)

// ContextKey defines a type alias for a stdlib Context key.
type ContextKey string

// SdkContextKey is the key in the context.Context which holds the sdk.Context.
const SdkContextKey ContextKey = "sdk-context"

// WrapSDKContext returns a stdlib context.Context with the provided sdk.Context's internal
// context as a value. It is useful for passing an sdk.Context  through methods that take a
// stdlib context.Context parameter such as generated gRPC methods. To get the original
// sdk.Context back, call UnwrapSDKContext.
//
// Deprecated: there is no need to wrap anymore as the Cosmos SDK context implements context.Context.
func WrapSDKContext(ctx Context) context.Context {
	return ctx
}

// UnwrapSDKContext retrieves a Context from a context.Context instance
// attached with WrapSDKContext. It panics if a Context was not properly
// attached
func UnwrapSDKContext(ctx context.Context) Context {
	if sdkCtx, ok := ctx.(Context); ok {
		return sdkCtx
	}
	return ctx.Value(SdkContextKey).(Context)
}

// todo should be cometbft abci.PendingTxChecker
type PendingTxCheckerResponse int

const (
	Accepted PendingTxCheckerResponse = iota
	Rejected
	Pending
)

type PendingTxChecker func() PendingTxCheckerResponse
