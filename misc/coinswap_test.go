package misc

import (
	"os"
	"testing"

	"github.com/btcsuite/btcd/btcec"
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
