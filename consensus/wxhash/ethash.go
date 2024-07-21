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

// Package ethash implements the ethash proof-of-work consensus engine.
package wxhash

import (
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// Ethash is a consensus engine based on proof-of-work implementing the ethash
// algorithm.
type Wxhash struct {
}

// NewFaker creates an ethash consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the Ethereum
// consensus rules.
func NewFaker() *Wxhash {
	return new(Wxhash)
}

// Close closes the exit channel to notify all backend threads exiting.
func (ethash *Wxhash) Close() error {
	return nil
}

// APIs implements consensus.Engine, returning no APIs as ethash is an empty
// shell in the post-merge world.
func (ethash *Wxhash) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{}
}

// Seal generates a new sealing request for the given input block and pushes
// the result into the given channel. For the ethash engine, this method will
// just panic as sealing is not supported anymore.
func (ethash *Wxhash) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	panic("ethash (pow) sealing not supported any more")
}
