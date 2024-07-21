package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/google/uuid"
)

func TestExtra(t *testing.T) {
	expectedIstExtra := &types.QBFTExtra{
		VanityData: []byte("steve"),
		Validators: []common.Address{
			common.BytesToAddress(hexutil.MustDecode("0x415b1312a4adc370eb791fd0db6086d5059b746a")),
			common.BytesToAddress(hexutil.MustDecode("0x8c04752f2b5b3a541b5709a095887ecb2a815f85")),
			common.BytesToAddress(hexutil.MustDecode("0x17afdd710ecd39435efc693c8fadc9b8411b8a23")),
			common.BytesToAddress(hexutil.MustDecode("0x9400e547db5c0ad78e0f166623cfdecab144b6f6")),
		},
		CommittedSeal: [][]byte{},
		Round:         0,
		Vote:          nil,
	}

	b, err := rlp.EncodeToBytes(expectedIstExtra)
	if err != nil {
		t.Error(err)
	}
	t.Log("extra", common.Bytes2Hex(b))
	//f85f857374657665f85494415b1312a4adc370eb791fd0db6086d5059b746a948c04752f2b5b3a541b5709a095887ecb2a815f859417afdd710ecd39435efc693c8fadc9b8411b8a23949400e547db5c0ad78e0f166623cfdecab144b6f6c080c0

	qbftExtra := new(types.QBFTExtra)
	err = rlp.DecodeBytes(b, qbftExtra)
	if err != nil {
		t.Error(err)
	}
	t.Log("VanityData", string(qbftExtra.VanityData))
	t.Log("Validators", qbftExtra.Validators)
}

func TestMakeKeyJson(t *testing.T) {
	// Create the keyfile object with a random UUID.
	UUID, err := uuid.NewRandom()
	if err != nil {
		utils.Fatalf("Failed to generate random uuid: %v", err)
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		utils.Fatalf("Failed to generate key: %v", err)
	}

	key := &keystore.Key{
		Id:         UUID,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	passphrase := ""
	scryptN, scryptP := keystore.LightScryptN, keystore.LightScryptP

	keyjson, err := keystore.EncryptKey(key, passphrase, scryptN, scryptP)
	if err != nil {
		utils.Fatalf("Error encrypting key: %v", err)
	}

	fmt.Println(string(keyjson))

	m := new(big.Int).Mul(big.NewInt(1000000000), big.NewInt(1e18))
	fmt.Println(m)

	fmt.Println(common.Bytes2Hex(m.Bytes()))
}

func TestMakeGenesis(t *testing.T) {
	file, err := os.Open("./genesis.json")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()

	genesis := new(core.Genesis)

	if err := json.NewDecoder(file).Decode(&genesis); err != nil {
		t.Errorf("invalid genesis file: %v", err)
	}
	t.Log(genesis)

	b, err := json.Marshal(genesis)
	if err != nil {
		t.Error(err)
	}
	decoded := new(core.Genesis)
	err = json.Unmarshal(b, &decoded)
	if err != nil {
		t.Error(err)
	}

	b, err = json.MarshalIndent(&decoded, "", "  ")
	if err != nil {
		t.Error(err)
	}

	t.Log(string(b))
}
