package swap

import (
	"os"
	"testing"
	"fmt"
	"encoding/hex"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	btcdhash "github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestComputeProofRootHash(t *testing.T) {
	proofText := `[{"left":1,"hash":"1b74f997236425e2e42e6abcde553b2cabe64c67aebc297be921dc97624a9323"},{"left":1,"hash":"99e97ac121039b12ccae98b020e65be2e87fa76b152e1981edfd17ad9236ab9b"},{"left":1,"hash":"cdb63da025d64fcef8502360d4ff92ce73585f933097e4b8c9d6d4f42764e1a6"},{"left":0,"hash":"637a3ea530856c587067107829bf57fd487910b1f1688d6ef7b0eb2b6c5435f2"},{"left":0,"hash":"65567e55025fb9bd036a5421ee6ed948341c2db5c51ce8ad2dd84f4a6b254b49"},{"left":1,"hash":"b50c0cdef32a7091867c84c6aa1bdd106789d555e312c9e8fd676932ace570e8"},{"left":1,"hash":"747c93561ca44e0b3fcddeef4843e4c3e7332ae2aa273e2b17e126c37b531d3d"},{"left":1,"hash":"499bd10e09ca60d076fd335be8c3604ec46be3dcdadbcaf0f78b94ceb03f72f9"},{"left":1,"hash":"2b1a34ae4ce9f2e0ea7f7d6f9d1625c5eea5328ff5ecf738821103350aaa7804"},{"left":1,"hash":"774ec950bdf3310e4e00eea1fae3036ff40c387642d41619d27190e4e6026099"},{"left":0,"hash":"b1f90056ad3f8c857af752e9072ba20087d6c9934dd8264a61631ed0eb4fe5d1"},{"left":0,"hash":"51c1edc44b03532892322d49b2ae7c92da003d173731572d45664f408f33375e"},{"left":0,"hash":"64537e844116161caf73b89331131e22cb17874ada168070878537a577e767e7"},{"left":0,"hash":"7fc7d789a5a146c9b01c88f0871ed52740703d55468f0d3c0cb30e43a3207442"},{"left":1,"hash":"dd7ec62e2efd794cf52b357fda34a71e09e70e90882fe55d8af07cbfc02a8ecc"},{"left":0,"hash":"84741e23e7932f4eb9a87c407005748fcbe7f2639b00738510f7b6c3a1a310c5"},{"left":0,"hash":"9c1db4c5f3758f3593fe8ad40623ac07419785e6b25a2169acc779770c5b80b0"},{"left":0,"hash":"25bca9cf47142fe4cc0632dfe099f969b38cc8991d323df6644cb600e90bbf60"}]`
	proofList := textToProofList(proofText)
	hashText := "1b7509f59479bc2b048fc8a5d6f5b1b239b3d3d5e818d663817f250bcac7ff47"
	hash, err := chainhash.NewHashFromStr(hashText)
	if err != nil {
		t.Fatal(err)
	}
	rootHashText := "40ed5ea75f6a5ab79ae109198703df1153b1cd272b122609d1bb3dc1b4ca376e"
	rootHash, err := chainhash.NewHashFromStr(rootHashText)
	if err != nil {
		t.Fatal(err)
	}

	proofHash := computeProofRootHash(hash, proofList)

	if proofHash != *rootHash {
		t.Fatalf("Unmatched hash: root=%s computed=%s", rootHashText, proofHash.String())
	}

	btcec.ParsePubKey(nil, btcec.S256())
}

func TestUnlockItem(t * testing.T) {
	unlockItemText := `{"txid":"03e73524b5bba21c4d2114900f6b8649e7ddc4480a8024c1ed09b883b5dbac45","out":0,"scriptPubKey":"210293704de00b9e47eee3848fc146fe659411ba839cda39534aa98a12611056939eac","amount":250000000000,"redeemScript":"483045022100c9ef24aa76cbb3b2341875c51af16aa0e1b17ac326203ab339c41c7578062f3b02203f5425c215872ee4a185f620a58baf986a9e614e7bac27b0df63412158af9d7401"}`
	unlockItem := textToUnlockItem(unlockItemText)

	err := verifyUnlockItem(&unlockItem)
	if err != nil {
		t.Fatal(err)
	}
}

// This is from BTCD txscript sample code
func TestBtcdExample(t * testing.T) {

	// Ordinarily the private key would come from whatever storage mechanism
	// is being used, but for this example just hard code it.
	privKeyBytes, err := hex.DecodeString("22a47fa09a223f2aa079edf85a7c2" +
		"d4f8720ee63e502ee2869afab7de234b80c")
	if err != nil {
		t.Fatal(err)
		fmt.Println(err)
		return
	}
	privKey, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), privKeyBytes)
	pubKeyHash := btcutil.Hash160(pubKey.SerializeCompressed())
	addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash,
		&chaincfg.MainNetParams)
	if err != nil {
		t.Fatal(err)
		fmt.Println(err)
		return
	}

	// For this example, create a fake transaction that represents what
	// would ordinarily be the real transaction that is being spent.  It
	// contains a single output that pays to address in the amount of 1 BTC.
	originTx := wire.NewMsgTx(wire.TxVersion)
	prevOut := wire.NewOutPoint(&btcdhash.Hash{}, ^uint32(0))
	txIn := wire.NewTxIn(prevOut, []byte{txscript.OP_0, txscript.OP_0}, nil)
	originTx.AddTxIn(txIn)
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		t.Fatal(err)
		fmt.Println(err)
		return
	}
	txOut := wire.NewTxOut(1, pkScript)
	originTx.AddTxOut(txOut)
	originTxHash := originTx.TxHash()

	// Create the transaction to redeem the fake transaction.
	redeemTx := wire.NewMsgTx(wire.TxVersion)

	// Add the input(s) the redeeming transaction will spend.  There is no
	// signature script at this point since it hasn't been created or signed
	// yet, hence nil is provided for it.
	prevOut = wire.NewOutPoint(&originTxHash, 0)
	txIn = wire.NewTxIn(prevOut, nil, nil)
	redeemTx.AddTxIn(txIn)

	// Ordinarily this would contain that actual destination of the funds,
	// but for this example don't bother.
	txOut = wire.NewTxOut(0, nil)
	redeemTx.AddTxOut(txOut)

	// Sign the redeeming transaction.
	lookupKey := func(a btcutil.Address) (*btcec.PrivateKey, bool, error) {
		// Ordinarily this function would involve looking up the private
		// key for the provided address, but since the only thing being
		// signed in this example uses the address associated with the
		// private key from above, simply return it with the compressed
		// flag set since the address is using the associated compressed
		// public key.
		//
		// NOTE: If you want to prove the code is actually signing the
		// transaction properly, uncomment the following line which
		// intentionally returns an invalid key to sign with, which in
		// turn will result in a failure during the script execution
		// when verifying the signature.
		//
		// privKey.D.SetInt64(12345)
		//
		return privKey, true, nil
	}
	// Notice that the script database parameter is nil here since it isn't
	// used.  It must be specified when pay-to-script-hash transactions are
	// being signed.
	sigScript, err := txscript.SignTxOutput(&chaincfg.MainNetParams,
		redeemTx, 0, originTx.TxOut[0].PkScript, txscript.SigHashAll,
		txscript.KeyClosure(lookupKey), nil, nil)
	if err != nil {
		t.Fatal(err)
		fmt.Println(err)
		return
	}
	redeemTx.TxIn[0].SignatureScript = sigScript

	// Prove that the transaction has been validly signed by executing the
	// script pair.
	flags := txscript.ScriptBip16 | txscript.ScriptVerifyDERSignatures |
		txscript.ScriptStrictMultiSig |
		txscript.ScriptDiscourageUpgradableNops
	vm, err := txscript.NewEngine(originTx.TxOut[0].PkScript, redeemTx, 0,
		flags, nil, nil, -1)
	if err != nil {
		t.Fatal(err)
		fmt.Println(err)
		return
	}
	if err := vm.Execute(); err != nil {
		t.Fatal(err)
		fmt.Println(err)
		return
	}
	//fmt.Println("Transaction successfully signed")

}
