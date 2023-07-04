// (c) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package evm

import (
	"container/heap"
	"math/big"
	"sync"
	"time"

	"github.com/lasthyphen/dijetalgo/cache"
	"github.com/lasthyphen/dijetalgo/ids"
	"github.com/lasthyphen/dijetalgo/snow"

	commonEng "github.com/lasthyphen/dijetalgo/snow/engine/common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/lasthyphen/subnet-evm/core"
	"github.com/lasthyphen/subnet-evm/core/state"
	"github.com/lasthyphen/subnet-evm/core/types"
	"github.com/lasthyphen/subnet-evm/plugin/evm/message"

	subnetEVM "github.com/lasthyphen/subnet-evm/chain"
)

const (
	// We allow [recentCacheSize] to be fairly large because we only store hashes
	// in the cache, not entire transactions.
	recentCacheSize = 512

	// [ethTxsGossipInterval] is how often we attempt to gossip newly seen
	// transactions to other nodes.
	ethTxsGossipInterval = 1 * time.Second
)

type Network interface {
	// Message handling
	AppRequestFailed(nodeID ids.ShortID, requestID uint32) error
	AppRequest(nodeID ids.ShortID, requestID uint32, deadline time.Time, msgBytes []byte) error
	AppResponse(nodeID ids.ShortID, requestID uint32, msgBytes []byte) error
	AppGossip(nodeID ids.ShortID, msgBytes []byte) error

	// Gossip entrypoints
	GossipEthTxs(txs []*types.Transaction) error
}

func (vm *VM) AppRequest(nodeID ids.ShortID, requestID uint32, deadline time.Time, request []byte) error {
	return vm.network.AppRequest(nodeID, requestID, deadline, request)
}

func (vm *VM) AppResponse(nodeID ids.ShortID, requestID uint32, response []byte) error {
	return vm.network.AppResponse(nodeID, requestID, response)
}

func (vm *VM) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	return vm.network.AppRequestFailed(nodeID, requestID)
}

func (vm *VM) AppGossip(nodeID ids.ShortID, msg []byte) error {
	return vm.network.AppGossip(nodeID, msg)
}

// NewNetwork creates a new Network based on the [vm.chainConfig].
func (vm *VM) NewNetwork(appSender commonEng.AppSender) Network {
	if vm.chainConfig.SubnetEVMTimestamp != nil {
		return vm.newPushNetwork(
			time.Unix(vm.chainConfig.SubnetEVMTimestamp.Int64(), 0),
			vm.config,
			appSender,
			vm.chain,
		)
	}

	return &noopNetwork{}
}

type pushNetwork struct {
	ctx                  *snow.Context
	gossipActivationTime time.Time
	config               Config

	appSender commonEng.AppSender
	chain     *subnetEVM.ETHChain

	gossipHandler message.Handler

	// We attempt to batch transactions we need to gossip to avoid runaway
	// amplification of mempol chatter.
	ethTxsToGossipChan chan []*types.Transaction
	ethTxsToGossip     map[common.Hash]*types.Transaction
	lastGossiped       time.Time
	shutdownChan       chan struct{}
	shutdownWg         *sync.WaitGroup

	// [recentEthTxs] prevent us from over-gossiping the
	// same transaction in a short period of time.
	recentEthTxs *cache.LRU
}

func (vm *VM) newPushNetwork(
	activationTime time.Time,
	config Config,
	appSender commonEng.AppSender,
	chain *subnetEVM.ETHChain,
) Network {
	net := &pushNetwork{
		ctx:                  vm.ctx,
		gossipActivationTime: activationTime,
		config:               config,
		appSender:            appSender,
		chain:                chain,
		ethTxsToGossipChan:   make(chan []*types.Transaction),
		ethTxsToGossip:       make(map[common.Hash]*types.Transaction),
		shutdownChan:         vm.shutdownChan,
		shutdownWg:           &vm.shutdownWg,
		recentEthTxs:         &cache.LRU{Size: recentCacheSize},
	}
	net.gossipHandler = &GossipHandler{
		vm:  vm,
		net: net,
	}
	net.awaitEthTxGossip()
	return net
}

// queueExecutableTxs attempts to select up to [maxTxs] from the tx pool for
// regossiping.
//
// We assume that [txs] contains an array of nonce-ordered transactions for a given
// account. This array of transactions can have gaps and start at a nonce lower
// than the current state of an account.
func (n *pushNetwork) queueExecutableTxs(state *state.StateDB, baseFee *big.Int, txs map[common.Address]types.Transactions, maxTxs int) types.Transactions {
	// Setup heap for transactions
	heads := make(types.TxByPriceAndTime, 0, len(txs))
	for addr, accountTxs := range txs {
		// Short-circuit here to avoid performing an unnecessary state lookup
		if len(accountTxs) == 0 {
			continue
		}

		// Ensure any transactions regossiped are immediately executable
		var (
			currentNonce = state.GetNonce(addr)
			tx           *types.Transaction
		)
		for _, accountTx := range accountTxs {
			// The tx pool may be out of sync with current state, so we iterate
			// through the account transactions until we get to one that is
			// executable.
			if accountTx.Nonce() == currentNonce {
				tx = accountTx
				break
			}
			// There may be gaps in the tx pool and we could jump past the nonce we'd
			// like to execute.
			if accountTx.Nonce() > currentNonce {
				break
			}
		}
		if tx == nil {
			continue
		}

		// Don't try to regossip a transaction too frequently
		if time.Since(tx.FirstSeen()) < n.config.TxRegossipFrequency.Duration {
			continue
		}

		// Ensure the fee the transaction pays is valid at tip
		wrapped, err := types.NewTxWithMinerFee(tx, baseFee)
		if err != nil {
			log.Debug(
				"not queuing tx for regossip",
				"tx", tx.Hash(),
				"err", err,
			)
			continue
		}

		heads = append(heads, wrapped)
	}
	heap.Init(&heads)

	// Add up to [maxTxs] transactions to be gossiped
	queued := make([]*types.Transaction, 0, maxTxs)
	for len(heads) > 0 && len(queued) < maxTxs {
		tx := heads[0].Tx
		queued = append(queued, tx)
		heap.Pop(&heads)
	}

	return queued
}

// queueRegossipTxs finds the best transactions in the mempool and adds up to
// [TxRegossipMaxSize] of them to [ethTxsToGossip].
func (n *pushNetwork) queueRegossipTxs() types.Transactions {
	txPool := n.chain.GetTxPool()

	// Fetch all pending transactions
	pending := txPool.Pending(true)

	// Split the pending transactions into locals and remotes
	localTxs := make(map[common.Address]types.Transactions)
	remoteTxs := pending
	for _, account := range txPool.Locals() {
		if txs := remoteTxs[account]; len(txs) > 0 {
			delete(remoteTxs, account)
			localTxs[account] = txs
		}
	}

	// Add best transactions to be gossiped (preferring local txs)
	tip := n.chain.BlockChain().CurrentBlock()
	state, err := n.chain.BlockChain().StateAt(tip.Root())
	if err != nil || state == nil {
		log.Debug(
			"could not get state at tip",
			"tip", tip.Hash(),
			"err", err,
		)
		return nil
	}
	localQueued := n.queueExecutableTxs(state, tip.BaseFee(), localTxs, n.config.TxRegossipMaxSize)
	localCount := len(localQueued)
	if localCount >= n.config.TxRegossipMaxSize {
		return localQueued
	}
	remoteQueued := n.queueExecutableTxs(state, tip.BaseFee(), remoteTxs, n.config.TxRegossipMaxSize-localCount)
	return append(localQueued, remoteQueued...)
}

// awaitEthTxGossip periodically gossips transactions that have been queued for
// gossip at least once every [ethTxsGossipInterval].
func (n *pushNetwork) awaitEthTxGossip() {
	n.shutdownWg.Add(1)
	go n.ctx.Log.RecoverAndPanic(func() {
		defer n.shutdownWg.Done()

		var (
			gossipTicker   = time.NewTicker(ethTxsGossipInterval)
			regossipTicker = time.NewTicker(n.config.TxRegossipFrequency.Duration)
		)

		for {
			select {
			case <-gossipTicker.C:
				if attempted, err := n.gossipEthTxs(false); err != nil {
					log.Warn(
						"failed to send eth transactions",
						"len(txs)", attempted,
						"err", err,
					)
				}
			case <-regossipTicker.C:
				for _, tx := range n.queueRegossipTxs() {
					n.ethTxsToGossip[tx.Hash()] = tx
				}
				if attempted, err := n.gossipEthTxs(true); err != nil {
					log.Warn(
						"failed to send eth transactions",
						"len(txs)", attempted,
						"err", err,
					)
				}
			case txs := <-n.ethTxsToGossipChan:
				for _, tx := range txs {
					n.ethTxsToGossip[tx.Hash()] = tx
				}
				if attempted, err := n.gossipEthTxs(false); err != nil {
					log.Warn(
						"failed to send eth transactions",
						"len(txs)", attempted,
						"err", err,
					)
				}
			case <-n.shutdownChan:
				return
			}
		}
	})
}

func (n *pushNetwork) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	return nil
}

func (n *pushNetwork) AppRequest(nodeID ids.ShortID, requestID uint32, deadline time.Time, msgBytes []byte) error {
	return nil
}

func (n *pushNetwork) AppResponse(nodeID ids.ShortID, requestID uint32, msgBytes []byte) error {
	return nil
}

func (n *pushNetwork) AppGossip(nodeID ids.ShortID, msgBytes []byte) error {
	return n.handle(
		n.gossipHandler,
		"Gossip",
		nodeID,
		0,
		msgBytes,
	)
}

func (n *pushNetwork) sendEthTxs(txs []*types.Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	txBytes, err := rlp.EncodeToBytes(txs)
	if err != nil {
		return err
	}
	msg := message.Txs{
		Txs: txBytes,
	}
	msgBytes, err := message.Build(&msg)
	if err != nil {
		return err
	}

	log.Trace(
		"gossiping eth txs",
		"len(txs)", len(txs),
		"size(txs)", len(msg.Txs),
	)
	return n.appSender.SendAppGossip(msgBytes)
}

func (n *pushNetwork) gossipEthTxs(force bool) (int, error) {
	if (!force && time.Since(n.lastGossiped) < ethTxsGossipInterval) || len(n.ethTxsToGossip) == 0 {
		return 0, nil
	}
	n.lastGossiped = time.Now()
	txs := make([]*types.Transaction, 0, len(n.ethTxsToGossip))
	for _, tx := range n.ethTxsToGossip {
		txs = append(txs, tx)
		delete(n.ethTxsToGossip, tx.Hash())
	}

	pool := n.chain.GetTxPool()
	selectedTxs := make([]*types.Transaction, 0)
	for _, tx := range txs {
		txHash := tx.Hash()
		txStatus := pool.Status([]common.Hash{txHash})[0]
		if txStatus != core.TxStatusPending {
			continue
		}

		if n.config.RemoteTxGossipOnlyEnabled && pool.HasLocal(txHash) {
			continue
		}

		// We check [force] outside of the if statement to avoid an unnecessary
		// cache lookup.
		if !force {
			if _, has := n.recentEthTxs.Get(txHash); has {
				continue
			}
		}
		n.recentEthTxs.Put(txHash, nil)

		selectedTxs = append(selectedTxs, tx)
	}

	if len(selectedTxs) == 0 {
		return 0, nil
	}

	// Attempt to gossip [selectedTxs]
	msgTxs := make([]*types.Transaction, 0)
	msgTxsSize := common.StorageSize(0)
	for _, tx := range selectedTxs {
		size := tx.Size()
		if msgTxsSize+size > message.EthMsgSoftCapSize {
			if err := n.sendEthTxs(msgTxs); err != nil {
				return len(selectedTxs), err
			}
			msgTxs = msgTxs[:0]
			msgTxsSize = 0
		}
		msgTxs = append(msgTxs, tx)
		msgTxsSize += size
	}

	// Send any remaining [msgTxs]
	return len(selectedTxs), n.sendEthTxs(msgTxs)
}

// GossipEthTxs enqueues the provided [txs] for gossiping. At some point, the
// [pushNetwork] will attempt to gossip the provided txs to other nodes
// (usually right away if not under load).
//
// NOTE: We never return a non-nil error from this function but retain the
// option to do so in case it becomes useful.
func (n *pushNetwork) GossipEthTxs(txs []*types.Transaction) error {
	if time.Now().Before(n.gossipActivationTime) {
		log.Trace(
			"not gossiping eth txs before the gossiping activation time",
			"len(txs)", len(txs),
		)
		return nil
	}

	select {
	case n.ethTxsToGossipChan <- txs:
	case <-n.shutdownChan:
	}
	return nil
}

func (n *pushNetwork) handle(
	handler message.Handler,
	handlerName string,
	nodeID ids.ShortID,
	requestID uint32,
	msgBytes []byte,
) error {
	log.Trace(
		"App message handler called",
		"handler", handlerName,
		"peerID", nodeID,
		"requestID", requestID,
		"len(msg)", len(msgBytes),
	)

	if time.Now().Before(n.gossipActivationTime) {
		log.Trace("App message called before activation time")
		return nil
	}

	msg, err := message.Parse(msgBytes)
	if err != nil {
		log.Trace(
			"dropping App message due to failing to parse message",
			"err", err,
		)
		return nil
	}

	return msg.Handle(handler, nodeID, requestID)
}

type GossipHandler struct {
	message.NoopHandler

	vm  *VM
	net *pushNetwork
}

func (h *GossipHandler) HandleTxs(nodeID ids.ShortID, _ uint32, msg *message.Txs) error {
	log.Trace(
		"AppGossip called with EthTxs",
		"peerID", nodeID,
		"size(txs)", len(msg.Txs),
	)

	if len(msg.Txs) == 0 {
		log.Trace(
			"AppGossip received empty EthTxs Message",
			"peerID", nodeID,
		)
		return nil
	}

	// The maximum size of this encoded object is enforced by the codec.
	txs := make([]*types.Transaction, 0)
	if err := rlp.DecodeBytes(msg.Txs, &txs); err != nil {
		log.Trace(
			"AppGossip provided invalid txs",
			"peerID", nodeID,
			"err", err,
		)
		return nil
	}
	errs := h.net.chain.GetTxPool().AddRemotes(txs)
	for i, err := range errs {
		if err != nil {
			log.Trace(
				"AppGossip failed to add to mempool",
				"err", err,
				"tx", txs[i].Hash(),
			)
		}
	}
	return nil
}

// noopNetwork should be used when gossip communication is not supported
type noopNetwork struct{}

func (n *noopNetwork) AppRequestFailed(nodeID ids.ShortID, requestID uint32) error {
	return nil
}

func (n *noopNetwork) AppRequest(nodeID ids.ShortID, requestID uint32, deadline time.Time, msgBytes []byte) error {
	return nil
}

func (n *noopNetwork) AppResponse(nodeID ids.ShortID, requestID uint32, msgBytes []byte) error {
	return nil
}

func (n *noopNetwork) AppGossip(nodeID ids.ShortID, msgBytes []byte) error {
	return nil
}

func (n *noopNetwork) GossipEthTxs(txs []*types.Transaction) error {
	return nil
}
