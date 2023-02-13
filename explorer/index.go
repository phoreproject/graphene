package explorer

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/phoreproject/synapse/chainhash"
)

// TransactionData is the data of a transaction that gets passed to templates.
type TransactionData struct {
	Recipient string
	Amount    int64
	Slot      uint64
}

// ProposerData is the data of a proposer that gets passed to templates.
type ProposerData struct {
	Slot      uint64
	Validator string
}

// IndexData is the data that gets sent to the index page.
type IndexData struct {
	Blocks       []BlockData
	Transactions []TransactionData
}

func (ex *Explorer) renderIndex(c echo.Context) error {
	blocks := make([]BlockData, 0)

	var dbBlocks []Block

	ex.database.database.Order("slot desc").Limit(30).Preload("Attestations").Find(&dbBlocks)

	blockHashes := make([]string, 0)
	for _, b := range dbBlocks {
		blockHashes = append(blockHashes, hex.EncodeToString(b.Hash))
	}

	for _, b := range dbBlocks {
		var blockHash chainhash.Hash
		copy(blockHash[:], b.Hash)

		var proposerHash chainhash.Hash
		copy(proposerHash[:], b.Proposer)

		var parentBlockHash chainhash.Hash
		copy(parentBlockHash[:], b.ParentBlockHash)

		var stateRoot chainhash.Hash
		copy(stateRoot[:], b.StateRoot)

		var attestationData []AttestationData
		for _, a := range b.Attestations {
			var attestationHash chainhash.Hash
			copy(attestationHash[:], a.Hash)

			participantHashes := strings.Split(a.ParticipantHashes, ",")

			attestationData = append(attestationData, AttestationData{
				Hash:               attestationHash.String(),
				ParticipantHashes:  participantHashes,
				Signature:          fmt.Sprintf("%x", a.Signature),
				Slot:               a.Slot,
				Shard:              a.Shard,
				BeaconBlockHash:    hex.EncodeToString(a.BeaconBlockHash),
				EpochBoundaryHash:  hex.EncodeToString(a.EpochBoundaryHash),
				ShardBlockHash:     hex.EncodeToString(a.ShardBlockHash),
				JustifiedSlot:      a.JustifiedSlot,
				JustifiedBlockHash: hex.EncodeToString(a.JustifiedBlockHash),
			})
		}

		blocks = append(blocks, BlockData{
			Slot:         b.Slot,
			BlockHash:    blockHash.String(),
			ProposerHash: proposerHash.String(),
			ParentBlock:  parentBlockHash.String(),
			StateRoot:    stateRoot.String(),
			RandaoReveal: fmt.Sprintf("%x", b.RandaoReveal),
			Signature:    fmt.Sprintf("%x", b.Signature),
			Timestamp:    b.Timestamp,
			Height:       b.Height,
			Attestations: b.Attestations,
		})
	}

	transactions := make([]TransactionData, 0)

	var dbTransactions []Transaction

	ex.database.database.Select("recipient_hash, amount, slot").Order("slot desc").Limit(30).Find(&dbTransactions)

	for _, t := range dbTransactions {
		var recipientHash chainhash.Hash
		copy(recipientHash[:], t.RecipientHash)

		transactions = append(transactions, TransactionData{
			Slot:      t.Slot,
			Amount:    t.Amount,
			Recipient: recipientHash.String(),
		})
	}

	err := c.JSON(http.StatusOK, IndexData{
		Blocks:       blocks,
		Transactions: transactions,
	})

	if err != nil {
		return err
	}
	return err
}
