package vote

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

var votesManagerCounter = metrics.NewRegisteredCounter("votesManager/local", nil)

// Backend wraps all methods required for voting.
type Backend interface {
	IsMining() bool
	EventMux() *event.TypeMux
}

// VoteManager will handle the vote produced by self.
type VoteManager struct {
	eth Backend

	chain *core.BlockChain

	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription

	// used for backup validators to sync votes from corresponding mining validator
	syncVoteCh  chan core.NewVoteEvent
	syncVoteSub event.Subscription

	pool    *VotePool
	signer  *VoteSigner
	journal *VoteJournal

	engine consensus.PoSA
}

func NewVoteManager(eth Backend, chain *core.BlockChain, pool *VotePool, journalPath, blsPasswordPath, blsWalletPath string, engine consensus.PoSA) (*VoteManager, error) {
	voteManager := &VoteManager{
		eth:         eth,
		chain:       chain,
		chainHeadCh: make(chan core.ChainHeadEvent, chainHeadChanSize),
		syncVoteCh:  make(chan core.NewVoteEvent, voteBufferForPut),
		pool:        pool,
		engine:      engine,
	}

	// Create voteSigner.
	voteSigner, err := NewVoteSigner(blsPasswordPath, blsWalletPath)
	if err != nil {
		return nil, err
	}
	log.Info("Create voteSigner successfully")
	voteManager.signer = voteSigner

	// Create voteJournal
	voteJournal, err := NewVoteJournal(journalPath)
	if err != nil {
		return nil, err
	}
	log.Info("Create voteJournal successfully")
	voteManager.journal = voteJournal

	// Subscribe to chain head event.
	voteManager.chainHeadSub = voteManager.chain.SubscribeChainHeadEvent(voteManager.chainHeadCh)
	voteManager.syncVoteSub = voteManager.pool.SubscribeNewVoteEvent(voteManager.syncVoteCh)

	return voteManager, nil
}

// UnderRules checks if the produced header under the following rules:
// A validator must not publish two distinct votes for the same height. (Rule 1)
// A validator must not vote within the span of its other votes . (Rule 2)
// Validators always vote for their canonical chain’s latest block. (Rule 3)
func (voteManager *VoteManager) UnderRules(header *types.Header) (bool, uint64, common.Hash) {
	sourceNumber, sourceHash, err := voteManager.engine.GetJustifiedNumberAndHash(voteManager.chain, header)
	if err != nil {
		log.Error("failed to get the highest justified number and hash at cur header", "curHeader's BlockNumber", header.Number, "curHeader's BlockHash", header.Hash())
		return false, 0, common.Hash{}
	}

	targetNumber := header.Number.Uint64()

	voteDataBuffer := voteManager.journal.voteDataBuffer
	//Rule 1:  A validator must not publish two distinct votes for the same height.
	if voteDataBuffer.Contains(targetNumber) {
		log.Debug("err: A validator must not publish two distinct votes for the same height.")
		return false, 0, common.Hash{}
	}

	//Rule 2: A validator must not vote within the span of its other votes.
	blockNumber := sourceNumber + 1
	if blockNumber+maliciousVoteSlashScope < targetNumber {
		blockNumber = targetNumber - maliciousVoteSlashScope
	}
	for ; blockNumber < targetNumber; blockNumber++ {
		if voteDataBuffer.Contains(blockNumber) {
			voteData, ok := voteDataBuffer.Get(blockNumber)
			if !ok {
				log.Error("Failed to get voteData info from LRU cache.")
				continue
			}
			if voteData.(*types.VoteData).SourceNumber > sourceNumber {
				log.Debug(fmt.Sprintf("error: cur vote %d-->%d is across the span of other votes %d-->%d",
					sourceNumber, targetNumber, voteData.(*types.VoteData).SourceNumber, voteData.(*types.VoteData).TargetNumber))
				return false, 0, common.Hash{}
			}
		}
	}
	for blockNumber := targetNumber + 1; blockNumber <= targetNumber+upperLimitOfVoteBlockNumber; blockNumber++ {
		if voteDataBuffer.Contains(blockNumber) {
			voteData, ok := voteDataBuffer.Get(blockNumber)
			if !ok {
				log.Error("Failed to get voteData info from LRU cache.")
				continue
			}
			if voteData.(*types.VoteData).SourceNumber < sourceNumber {
				log.Debug(fmt.Sprintf("error: cur vote %d-->%d is within the span of other votes %d-->%d",
					sourceNumber, targetNumber, voteData.(*types.VoteData).SourceNumber, voteData.(*types.VoteData).TargetNumber))
				return false, 0, common.Hash{}
			}
		}
	}

	// Rule 3: Validators always vote for their canonical chain’s latest block.
	// Since the header subscribed to is the canonical chain, so this rule is satisfied by default.
	log.Debug("All three rules check passed")
	return true, sourceNumber, sourceHash
}
