// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package wxhash

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"golang.org/x/crypto/sha3"
)

var (
	WxhashDifficulty = big.NewInt(1)
	//	DefaultDifficulty = big.NewInt(1)
	//beaconDifficulty = common.Big0
	allowedFutureBlockTimeSeconds = int64(15) // Max seconds from current time allowed for blocks, before they're considered future blocks
)

var (
	errOlderBlockTime = errors.New("timestamp older than parent")
)

func (w *Wxhash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (wxhash *Wxhash) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return wxhash.verifyHeader(chain, header, parent, false, time.Now().Unix())
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (wxhash *Wxhash) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}
	abort := make(chan struct{})
	results := make(chan error, len(headers))
	unixNow := time.Now().Unix()

	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
			} else if headers[i-1].Hash() == headers[i].ParentHash {
				parent = headers[i-1]
			}
			var err error
			if parent == nil {
				err = consensus.ErrUnknownAncestor
			} else {
				err = wxhash.verifyHeader(chain, header, parent, false, unixNow)
			}
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (wxhash *Wxhash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
// See YP section 4.3.4. "Block Header Validity"
func (wxhash *Wxhash) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, uncle bool, unixNow int64) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if !uncle {
		if header.Time > uint64(unixNow+allowedFutureBlockTimeSeconds) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	// Verify the block's difficulty based on its timestamp and parent's difficulty
	expected := wxhash.CalcDifficulty(chain, header.Time, parent)
	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	if header.GasLimit > params.MaxGasLimit {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, params.MaxGasLimit)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (wxhash *Wxhash) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return WxhashDifficulty
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (wxhash *Wxhash) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = wxhash.CalcDifficulty(chain, header.Time, parent)
	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards.
func (wxhash *Wxhash) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
	// Accumulate any block and uncle rewards
	accumulateRewards(chain.Config(), state, header, uncles)
}

// FinalizeAndAssemble implements consensus.Engine, accumulating the block and
// uncle rewards, setting the final state and assembling the block.
func (wxhash *Wxhash) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt, withdrawals []*types.Withdrawal) (*types.Block, error) {
	if len(withdrawals) > 0 {
		return nil, errors.New("wxhash does not support withdrawals")
	}
	// Finalize block
	wxhash.Finalize(chain, header, state, txs, uncles, nil)

	// Assign the final state root to header.
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, uncles, receipts, trie.NewStackTrie(nil)), nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (wxhash *Wxhash) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if header.WithdrawalsHash != nil {
		panic("withdrawal hash set on wxhash")
	}
	if header.ExcessBlobGas != nil {
		panic("excess blob gas set on wxhash")
	}
	if header.BlobGasUsed != nil {
		panic("blob gas used set on wxhash")
	}
	if header.ParentBeaconRoot != nil {
		panic("parent beacon root set on wxhash")
	}
	rlp.Encode(hasher, enc)
	hasher.Sum(hash[:0])
	return hash
}

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(_ *params.ChainConfig, state *state.StateDB, header *types.Header, _ []*types.Header) {
	// todo
	type reward struct {
		Addr   common.Address `json:"addr"`
		Reward *big.Int       `json:"reward"`
	}

	rewards := []reward{}
	if err := json.Unmarshal(header.Rewards, &rewards); err != nil {
		panic("failed to unmarshal rewards")
	}

	for _, r := range rewards {
		state.AddBalance(r.Addr, uint256.MustFromBig(r.Reward))
	}
}
