package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func decryptKey(jsonPath string, passphrase string) (common.Address, *ecdsa.PrivateKey, error) {
	keyjson, err := os.ReadFile(jsonPath)
	if err != nil {
		return common.Address{}, nil, err
	}

	key, err := keystore.DecryptKey(keyjson, passphrase)
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(key.PrivateKey.PublicKey), key.PrivateKey, nil
}

func getBalance(ctx context.Context, ec *ethclient.Client, addr common.Address) error {
	balance, err := ec.BalanceAt(ctx, addr, nil)
	if err != nil {
		return err
	}
	fmt.Println("addr", addr, "balance", balance)
	return nil
}

func geteAddressFromEncryptedKey(jsonPath string) (common.Address, error) {
	keyjson, err := os.ReadFile(jsonPath)
	if err != nil {
		return common.Address{}, err
	}
	key := map[string]interface{}{}
	if err := json.Unmarshal(keyjson, &key); err != nil {
		return common.Address{}, err
	}
	return common.HexToAddress(key["address"].(string)), nil
}

func makeTxData(ctx context.Context, ec *ethclient.Client, from, to common.Address, value *big.Int) (*types.DynamicFeeTx, types.Signer, error) {
	// chainID
	chainID, err := ec.ChainID(ctx)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("chainID", chainID)

	// nonce
	nonce, err := ec.NonceAt(ctx, from, nil)
	if err != nil {
		return nil, nil, err
	}
	fmt.Println("nonce", nonce)

	// latest header
	header, err := ec.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	// gasTipCap
	gasTipCap, err := ec.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, err
	}
	// gasFeeCap
	gasFeeCap := new(big.Int).Add(gasTipCap, new(big.Int).Mul(header.BaseFee, common.Big2))

	// gasLimit
	gasLimit := uint64(21000)
	fmt.Println("header.Number", header.Number, "header.BaseFee", header.BaseFee, "gasTipCap", gasTipCap, "gasFeeCap", gasFeeCap, "gasLimit", gasLimit)

	return &types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     value,
	}, types.NewLondonSigner(chainID), nil
}

func main() {
	if err := func() error {
		url := "http://127.0.0.1:22001"
		ctx := context.Background()
		var ec *ethclient.Client

		// connect to node
		if client, err := rpc.DialContext(ctx, url); err != nil {
			return err
		} else {
			ec = ethclient.NewClient(client)
			fmt.Println("URL", url)
		}

		// decrypt from's pk
		from, fromPK, err := decryptKey("./alloc.json", "")
		if err != nil {
			return err
		}
		fmt.Println("from", from)

		// to
		to, err := geteAddressFromEncryptedKey("./recipient.json")
		if err != nil {
			return err
		}
		fmt.Println("to", to)

		fmt.Println("balance before transfer")
		getBalance(ctx, ec, from)
		getBalance(ctx, ec, to)

		// tx data and london-signer
		txData, signer, err := makeTxData(ctx, ec, from, to, big.NewInt(1e18))
		if err != nil {
			return err
		}

		tx, err := types.SignTx(types.NewTx(txData), signer, fromPK)
		if err != nil {
			return err
		}

		// send tx
		if err = ec.SendTransaction(ctx, tx); err != nil {
			return err
		}

		start := time.Now()
		fmt.Println("waits for tx to be mined....")

		// get receipt
		if recepit, err := bind.WaitMined(ctx, ec, tx); err != nil {
			return err
		} else {
			fmt.Println("tx", tx.Hash(), "status", recepit.Status, "elapse", time.Since(start))
			getBalance(ctx, ec, from)
			getBalance(ctx, ec, to)
		}
		return nil
	}(); err != nil {
		panic(err)
	}
}
