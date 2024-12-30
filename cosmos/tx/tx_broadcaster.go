package tx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/tessellated-io/pickaxe/cosmos/rpc"
	"github.com/tessellated-io/pickaxe/crypto"
	"github.com/tessellated-io/pickaxe/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// Broadcaster wraps TxBroadcaster. You probably just want to use NewDefaultBroadcaster.

type Broadcaster struct {
	logger  *log.Logger
	wrapped TxBroadcaster
}

// Retryable broadcaster with polling and gas management.
func NewDefaultBroadcaster(
	chainName string,
	bech32Prefix string,
	signer crypto.BytesSigner,
	gasManager GasManager,
	logger *log.Logger,
	rpcClient rpc.RpcClient,
	signingMetadataProvider *SigningMetadataProvider,
	txProvider TxProvider,

	txPollAttempts uint,
	txPollDelay time.Duration,

	retryAttempts uint,
	retryDelay time.Duration,
) (*Broadcaster, error) {
	txb1, err := NewDefaultTxBroadcaster(chainName, bech32Prefix, signer, gasManager, logger, rpcClient, signingMetadataProvider, txProvider)
	if err != nil {
		return nil, err
	}

	txb2, err := NewPollingTxBroadcaster(txPollAttempts, txPollDelay, logger, txb1)
	if err != nil {
		return nil, err
	}

	txb3, err := NewGasTrackingTxBroadcaster(chainName, gasManager, logger, txb2)
	if err != nil {
		return nil, err
	}

	txb4, err := NewRetryableBroadcaster(retryAttempts, retryDelay, logger, txb3)
	if err != nil {
		return nil, err
	}

	broadcaster := &Broadcaster{
		logger:  logger,
		wrapped: txb4,
	}

	return broadcaster, nil
}

func (b *Broadcaster) SignAndBroadcast(ctx context.Context, msgs []sdk.Msg) (txHash string, err error) {
	for {
		// Ditch if context has timed out
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		// Attempt to sign and broadcast
		broadcastResult, broadcastErr := b.wrapped.signAndBroadcast(ctx, msgs)
		logger := b.logger.With("tx_hash", txHash, "error", broadcastErr, "broadcast_result", broadcastResult.String())

		logger.Debug("broadcaster::received from signAndBroadcast")
		if broadcastErr != nil {
			return "", broadcastErr
		}

		// Check if we failed
		isSuccess, isSuccessErr := IsSuccess(broadcastResult)
		if isSuccessErr != nil {
			panic("broadcaster::should never happen")
		}

		if !isSuccess {
			// If the broadcast result is a gas error, retry
			codespace := broadcastResult.TxResponse.Codespace
			code := broadcastResult.TxResponse.Code
			logs := broadcastResult.TxResponse.RawLog
			err := errors.New(logs)
			logger := logger.With("codespace", codespace, "code", code)

			if IsGasRelatedError(codespace, code) {
				logger.Error("failed to sign and broadcast due to gas, will retry", "error", err.Error())
				continue
			}

			// otherwise, we've failed.
			logger.Error("broadcasted, but got non-success response code", "error", err.Error())
			return "", err
		}

		// Otherwise, check for inclusion
		txHash := broadcastResult.TxResponse.TxHash
		txStatus, err := b.wrapped.checkTxStatus(ctx, txHash)
		if err != nil {
			// Something fundamentatlly bad happened, give up
			logger.Error("failed to get tx status", "error", err.Error)
			return "", err
		}

		if txStatus != nil && err == nil {
			// We got a tx status, so something is confirmed. Check if it was a gas error and retry if so.
			codespace := txStatus.TxResponse.Codespace
			code := txStatus.TxResponse.Code
			if IsGasRelatedError(codespace, code) {
				logger.Error("transaction landed on chain but failed due to gas, will retry", "error", fmt.Errorf("detected gas error in broadcast: %s", txStatus.TxResponse.RawLog))

				continue
			}

			// Otherwise, there is nothing we can do. Hot swap to an error though if needed.
			isSuccess := IsSuccessTxStatus(txStatus)
			if isSuccess {
				b.logger.Info("transaction sent and landed on chain, successfully.")
				return txHash, nil
			} else {
				err := errors.New(txStatus.TxResponse.RawLog)
				b.logger.Error("transaction sent and landed on chain but failed due to non-gas related error", "error", err.Error())
				return txHash, err
			}
		}

		if txStatus == nil && err == nil {
			// We didn't find the transaction, and we didn't get an error. Tough to say, but let's ditch since rebroadcasting could be dangerous
			// in case a bunch of txs settle in O(hours)
			err := fmt.Errorf("transaction status not found, consider increasing the gas fee")
			b.logger.Error("failed to get tx status", "error", err.Error())
			return "", err
		}

		panic("invalid status")
	}
}

// Broadcasts transactions reliably, and with retries
type TxBroadcaster interface {
	// Pass back a broadcast result, or error.
	signAndBroadcast(ctx context.Context, msgs []sdk.Msg) (broadcastResult *txtypes.BroadcastTxResponse, err error)

	// Pass back a tx status. If tx status is "not found" then pass back (nil, nil)
	checkTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error)
}

// default broadcaster simply broadcasts transactions
type defaultBroadcaster struct {
	// Parameters
	chainName    string
	bech32Prefix string
	signer       crypto.BytesSigner

	// Services
	gasManager              GasManager
	logger                  *log.Logger
	rpcClient               rpc.RpcClient
	signingMetadataProvider *SigningMetadataProvider
	txProvider              TxProvider
}

var _ TxBroadcaster = (*defaultBroadcaster)(nil)

func NewDefaultTxBroadcaster(
	chainName string,
	bech32Prefix string,
	signer crypto.BytesSigner,
	gasManager GasManager,
	logger *log.Logger,
	rpcClient rpc.RpcClient,
	signingMetadataProvider *SigningMetadataProvider,
	txProvider TxProvider,
) (TxBroadcaster, error) {
	broadcaster := &defaultBroadcaster{
		chainName:    chainName,
		bech32Prefix: bech32Prefix,
		signer:       signer,

		gasManager:              gasManager,
		logger:                  logger,
		rpcClient:               rpcClient,
		signingMetadataProvider: signingMetadataProvider,
		txProvider:              txProvider,
	}

	return broadcaster, nil
}

// Private helper, incorporating core functionality
func (b *defaultBroadcaster) signAndBroadcast(ctx context.Context, msgs []sdk.Msg) (broadcastResult *txtypes.BroadcastTxResponse, err error) {
	logger := b.logger.With("chain_name", b.chainName)

	// Get the gas price, which is needed to sign the message
	gasPrice, err := b.gasManager.GetGasPrice(b.chainName)
	if err != nil {
		return nil, err
	}
	logger.Debug("txbroadcaster received gas price")

	// Get the gas factor, which is needed to simulate the message
	gasFactor, err := b.gasManager.GetGasFactor(b.chainName)
	if err != nil {
		return nil, err
	}
	logger.Debug("txbroadcaster received gas factor")

	// Get the signer's metadata
	senderAddress := b.signer.GetAddress(b.bech32Prefix)
	signingMetadata, err := b.signingMetadataProvider.SigningMetadataForAccount(ctx, senderAddress)
	if err != nil {
		return nil, err
	}
	logger.Debug("txbroadcaster received signer metadata")

	// Formulate and sign the message
	signedMessage, gasWanted, err := b.txProvider.ProvideTx(ctx, gasPrice, gasFactor, msgs, signingMetadata)
	if err != nil {
		return nil, err
	}
	logger.Debug("tx broadcaster signed transaction")

	// Attempt to broadcast
	result, broadcastErr := b.rpcClient.Broadcast(ctx, signedMessage)

	// Log results, regardless of what happened
	if result != nil && result.TxResponse != nil {
		txHash := result.TxResponse.TxHash
		codespace := result.TxResponse.Codespace
		broadcastResponseCode := result.TxResponse.Code
		logs := result.TxResponse.RawLog
		b.logger.Info("📣 attempted to broadcast transaction", "logs", logs, "gas_price", gasPrice, "gas_factor", gasFactor, "tx_hash", txHash, "codespace", codespace, "code", broadcastResponseCode)
	}

	// Broadcast response helpfully sets `gasWanted` to zero if the transaction failed, which is a bit of a pain, especially if we want to get
	// gas data.
	// Swap it out here, in order to avoid returning spurious parameters.
	if result != nil && result.TxResponse != nil {
		// Sanity check
		responseGasWanted := result.TxResponse.GasWanted
		if responseGasWanted != 0 && responseGasWanted != gasWanted {
			panic(fmt.Sprintf("unexpected gas wanted in tx response. We calculated: %d, response had: %d", gasWanted, responseGasWanted))
		}

		result.TxResponse.GasWanted = gasWanted
	}

	return result, broadcastErr
}

func (b *defaultBroadcaster) checkTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	txStatus, err := b.rpcClient.GetTxStatus(ctx, txHash)
	codespace := txStatus.TxResponse.Codespace
	broadcastResponseCode := txStatus.TxResponse.Code
	logger := b.logger.With("chain_name", b.chainName, "tx_hash", txHash, "code", broadcastResponseCode, "codespace", codespace)
	if err == nil {
		logs := txStatus.TxResponse.RawLog
		logger.Info("got a settled tx status")
		logger.Debug("full tx logs", "logs", logs)

		return txStatus, nil
	}

	grpcErr, ok := status.FromError(err)
	if ok && grpcErr.Code() == codes.NotFound {

		// No error, but nothing was found
		logger.Debug("tx not included in chain")
		return nil, nil
	}

	logger.Debug("error querying tx status", "error", err.Error())
	return nil, err
}

// Polling broadcaster polls for tx inclusion
type pollingTxBroadcaster struct {
	// Parameters
	attempts uint
	delay    time.Duration

	// Services
	logger             *log.Logger
	wrappedBroadcaster TxBroadcaster
}

var _ TxBroadcaster = (*pollingTxBroadcaster)(nil)

func NewPollingTxBroadcaster(
	attempts uint,
	delay time.Duration,
	logger *log.Logger,
	wrappedBroadcaster TxBroadcaster,
) (TxBroadcaster, error) {
	broadcaster := &pollingTxBroadcaster{
		attempts: attempts,
		delay:    delay,

		logger:             logger,
		wrappedBroadcaster: wrappedBroadcaster,
	}

	return broadcaster, nil
}

func (b *pollingTxBroadcaster) signAndBroadcast(ctx context.Context, msgs []sdk.Msg) (broadcastResult *txtypes.BroadcastTxResponse, err error) {
	// Pass through, there's no polling to be done on initial broadcast.
	return b.wrappedBroadcaster.signAndBroadcast(ctx, msgs)
}

func (b *pollingTxBroadcaster) checkTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	logger := b.logger.With("tx_hash", txHash)
	logger.Info("polling for inclusion")

	var i uint
	for i = 0; i < b.attempts; i++ {
		// Ditch if context has timed out
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Initially sleep to give time to settle
		time.Sleep(b.delay)

		// Ask internal clients for results.
		txStatus, err := b.wrappedBroadcaster.checkTxStatus(ctx, txHash)

		if txStatus != nil {
			if err == nil {
				// We have a status, return it.
				return txStatus, err
			} else {
				// We should never have a non nil tx status and non nil error
				panic("unexpected state.")
			}
		} else {
			// precondition: txStatus is nil
			if err == nil {
				b.logger.Info("transaction still not included", "attempt", i+1, "max_attempts", b.attempts)
			} else {
				// something more fundamental has gone wrong.
				return nil, err
			}
		}
	}

	err := fmt.Errorf("transaction not included after exhausting all polling attempts: %s", txHash)
	logger.Error("polling finished", "error", err.Error())

	// Return nil here, since a not found error isn't really a nil error, and we may want to do things like track gas.
	return nil, nil
}

// gasTrackingTxBroadcaster tracks and updates gas prices
type gasTrackingTxBroadcaster struct {
	chainName string

	// Services
	gasManager         GasManager
	logger             *log.Logger
	wrappedBroadcaster TxBroadcaster
}

var _ TxBroadcaster = (*gasTrackingTxBroadcaster)(nil)

func NewGasTrackingTxBroadcaster(
	chainName string,
	gasManager GasManager,
	logger *log.Logger,
	wrappedBroadcaster TxBroadcaster,
) (TxBroadcaster, error) {
	broadcaster := &gasTrackingTxBroadcaster{
		chainName: chainName,

		gasManager:         gasManager,
		logger:             logger,
		wrappedBroadcaster: wrappedBroadcaster,
	}

	return broadcaster, nil
}

// NOTE: This function is just a pure pass through that does gas management
func (b *gasTrackingTxBroadcaster) signAndBroadcast(ctx context.Context, msgs []sdk.Msg) (*txtypes.BroadcastTxResponse, error) {
	result, err := b.wrappedBroadcaster.signAndBroadcast(ctx, msgs)
	if err != nil {
		return nil, err
	}

	// Check for success
	isSuccess, err := IsSuccess(result)
	if err != nil {
		panic("gas_tracking_tx_broadcaster::should never happen")
	}

	// Don't adjust on successful broadcasts, instead wait to see if it successfully lands on chain.
	if isSuccess {
		return result, err
	}

	// Otherwise, try to handle the result for gas adjustment
	gasManagementErr := b.gasManager.ManageFailingBroadcastResult(b.chainName, result)
	if gasManagementErr != nil {
		b.logger.Warn("failed to adjust gas due to broadcast result", "error", gasManagementErr)
	}

	return result, err
}

func (b *gasTrackingTxBroadcaster) checkTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	txStatus, err := b.wrappedBroadcaster.checkTxStatus(ctx, txHash)
	b.logger.Debug("gas_tracking_tx_broadcaster::got checktx result", "error", err, "tx_status", txStatus.String())

	// Errors indicate a fundamental problem, like network connectivity
	if err != nil {
		b.logger.Debug("gas_tracking_tx_broadcaster::received error from wrapped broadcaster, will not adjust gas")
		return txStatus, err
	}

	// If there's no error, but no txstatus reported, then the transaction is probably under-fee'd and we should report failure
	if err == nil && txStatus == nil {
		b.logger.Debug("gas_tracking_tx_broadcaster::did not find transaction, but did not get an error, adjusting gas")

		gasManagementErr := b.gasManager.ManageInclusionFailure(b.chainName)
		if gasManagementErr != nil {
			b.logger.Warn("failed to adjust gas due to missing tx inclusion", "error", gasManagementErr)
		}

		b.logger.Debug("gas_tracking_tx_broadcaster::adjusted gas")
		return txStatus, err
	}

	// If there is a tx status, try to manage it.
	b.logger.Debug("gas_tracking_tx_broadcaster::got a check tx result")
	gasManagementErr := b.gasManager.ManageIncludedTransactionStatus(b.chainName, txStatus)
	if gasManagementErr != nil {
		b.logger.Warn("failed to adjust gas due to tx status")
	}
	b.logger.Debug("gas_tracking_tx_broadcaster::adjusted gas due to check tx result")
	return txStatus, err
}

// Retrying broadcaster retries broadcasting. Attempts failing due to gas errors are retried
type retryableTxBroadcaster struct {
	// Parameters
	attempts uint
	delay    time.Duration

	// Services
	logger             *log.Logger
	wrappedBroadcaster TxBroadcaster
}

var _ TxBroadcaster = (*retryableTxBroadcaster)(nil)

func NewRetryableBroadcaster(
	attempts uint,
	delay time.Duration,
	logger *log.Logger,
	wrappedBroadcaster TxBroadcaster,
) (TxBroadcaster, error) {
	broadcaster := &retryableTxBroadcaster{
		attempts: attempts,
		delay:    delay,

		logger:             logger,
		wrappedBroadcaster: wrappedBroadcaster,
	}

	return broadcaster, nil
}

func (b *retryableTxBroadcaster) signAndBroadcast(ctx context.Context, msgs []sdk.Msg) (broadcastResult *txtypes.BroadcastTxResponse, err error) {
	logger := b.logger.With("max_attempts", b.attempts)

	var i uint
	for i = 0; i < b.attempts; i++ {
		// Ditch if context has timed out
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Attempt to sign and broadcast
		result, err := b.wrappedBroadcaster.signAndBroadcast(ctx, msgs)
		if err == nil {
			return result, err
		}
		logger := logger.With("attempt", i+1, "error", err.Error())

		// Give up if all attempts are exhausted.
		if i+1 == b.attempts {
			logger.Error("failed in all attempts to sign and broadcast")
			return result, err
		}

		// Otherwise, poll and wait.
		logger.Error("failed to sign and broadcast, will retry")
		time.Sleep(b.delay)
	}
	panic("retryable_tx_broadcaster::sign_and_broadcast::should never happen")
}

func (b *retryableTxBroadcaster) checkTxStatus(ctx context.Context, txHash string) (*txtypes.GetTxResponse, error) {
	logger := b.logger.With("max_attempts", b.attempts)

	var i uint
	for i = 0; i < b.attempts; i++ {
		// Ditch if context has timed out
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Attempt to sign and broadcast
		result, err := b.wrappedBroadcaster.checkTxStatus(ctx, txHash)
		if err == nil {
			return result, err
		}
		logger := logger.With("attempt", i+1, "error", err.Error())

		// Give up if all attempts are exhausted.
		if i+1 == b.attempts {
			logger.Error("failed in all attempts to check tx status")
			return result, err
		}

		// Otherwise, poll and wait.
		logger.Error("failed to check tx status, will retry")
		time.Sleep(b.delay)
	}
	panic("retryable_tx_broadcaster::check_tx_status::should never happen")
}

// Helpers

func IsSuccess(broadcastResult *txtypes.BroadcastTxResponse) (bool, error) {
	if broadcastResult == nil {
		return false, fmt.Errorf("received nil broadcast tx result")
	}
	if broadcastResult.TxResponse == nil {
		return false, fmt.Errorf("received nil tx response in broadcast tx result")
	}

	// Note: Zero codes do not have a codespace on them
	return broadcastResult.TxResponse.Code == 0, nil
}

func IsSuccessTxStatus(txStatus *txtypes.GetTxResponse) bool {
	code := txStatus.TxResponse.Code
	return code == 0
}
