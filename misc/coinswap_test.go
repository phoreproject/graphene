package misc

import (
	"os"
	"testing"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	logrus.SetLevel(logrus.DebugLevel)
	retCode := m.Run()
	os.Exit(retCode)
}

func TestComputeProofRootHash(t *testing.T) {
	proofText := "L1b74f997236425e2e42e6abcde553b2cabe64c67aebc297be921dc97624a9323Rdd42a5cf029522116657f38ba7abc88f5b38fa1c274aeedb889c7719955a47c2R90d0d50efde71fce94e8d213ce62ecb75e2555fa448958d2f6434e8b3ee4740fR5501e10d7b90e7ec87eb956d21a9fde86a6b15cf99d16750c40e0a09e5d463a8L3da0910c3d543c6984a1efc5b22988ee14d719022896940aa9e21bffe4b33cc1Rb823d0827e2b3bd28afafb4e7503e428a20be4690d519d13facd20dd922d1a44R9469b28a94af53b032500d0c930d431fb5fa8e0a64eee6f1e366581873434fd5Le5c217fd53ef06c71502e9f4691ada50a59f191d596c54c6007428161b0007dbL0f7d3ca8672bf5a5bd53b62f4a476e55663b5fd6db365352ae38b4e96bae1a6dLce462ab56f8f9df89c9ef268c2df18ab7cf2b25d77aa84ff1ebcde543ed1fa30Rb3accc225d38eac59aa0a8e8c73fef4adb84478fe57b6a6bd388786dda03ec7dRb9c09b68c9207c58945c32b36c58b899bb69dc8e7b403aa447bb049778abba57R8081ebb0d1c2ba7265852ef6b5350fa1034104eb11b7ed3d0e693c82d1f7664cR99e5eff4721d2d89170f60cd0e355dc74926324aa94559969e4d4448692917e0L78209bd8d8674a60dfc3ad33dfa9d59add69c89d2b0c3da1682f21a01f5d806dRee70f9382a46211d2312de52f75e2bfaf5d90b11a8b9c37e866713734c0fb61aR9611cb8847e702f3bdb5139aacdc540290f3e53b368b79f4511719b99db44e88R8965b2071c637ad7788689b930a083d3d4f572314fba408b45550efc12e4bfa9"
	proofList := textToProofList(proofText)
	hashText := "1b7509f59479bc2b048fc8a5d6f5b1b239b3d3d5e818d663817f250bcac7ff47"
	hash, err := chainhash.NewHashFromStr(hashText)
	if err != nil {
		t.Fatal(err)
	}
	rootHashText := "09a550c89002802b1d5ae473f839ced904439c03bcc19d399869d095312b3d52"
	rootHash, err := chainhash.NewHashFromStr(rootHashText)
	if err != nil {
		t.Fatal(err)
	}

	proofHash := computeProofRootHash(hash, proofList)

	if proofHash != *rootHash {
		t.Fatalf("Unmatched hash: root=%s computed=%s", rootHashText, proofHash.String())
	}
}
