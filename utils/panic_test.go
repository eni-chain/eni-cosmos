package utils_test

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/utils/helpers"
	"github.com/ethereum/go-ethereum/crypto"
	"testing"

	"github.com/cosmos/cosmos-sdk/utils"
	"github.com/stretchr/testify/require"
)

func TestHardFail(t *testing.T) {
	hardFailer := func() {
		panic(utils.DecorateHardFailError(errors.New("some error")))
	}
	panicHandlingFn := func() {
		defer utils.PanicHandler(func(_ any) {})()
		hardFailer()
	}
	require.Panics(t, panicHandlingFn)
}

func TestName(t *testing.T) {
	//priv := secp256k1.PrivKey(privB)
	////println(priv.PubKey().Address().String())
	//address, err := helpers.PubkeyToEVMAddress(priv.PubKey().Bytes())
	//if err != nil {
	//	panic(err)
	//}
	//println(address.String())
	// 示例私钥（16 进制格式）
	privateKeyHex := "08609b373a64e358cee1dfbf1740dc8a6c98cdb025ae7de73a1429316b9906c9"

	// 解析 16 进制私钥
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		t.Fatal(err)
	}

	// 转换为 ECDSA 私钥
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		t.Fatal("Failed to convert to ECDSA private key:", err)
	}

	// 获取公钥（ECDSA 格式）
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("Failed to cast public key to ECDSA")
	}

	// 计算公钥（65 字节未压缩格式，0x04 开头）
	publicKeyBytes := crypto.FromECDSAPub(publicKeyECDSA)
	fmt.Println("Public Key (Uncompressed):", hex.EncodeToString(publicKeyBytes))

	// 计算以太坊地址（Keccak-256 哈希后取后 20 字节）
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	fmt.Println("Address:", address.Hex())

	address, err = helpers.PubkeyToEVMAddress(publicKeyBytes)
	if err != nil {
		t.Fatal("Failed to PubkeyToEVMAddress:", err)
	}
	fmt.Println("Address2:", address.Hex())
}
