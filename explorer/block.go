package explorer

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/phoreproject/synapse/chainhash"
)

// BlockData is the data of a block that gets passed to templates.
type BlockData struct {
	Slot         uint64
	BlockHash    string
	ProposerHash string
	ParentBlock  string
	StateRoot    string
	RandaoReveal string
	Signature    string
}

func (ex *Explorer) renderBlock(c echo.Context) error {
	var b Block

	blockHashStr := c.Param("blockHash")

	blockHashHex, err := hex.DecodeString(blockHashStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Block hash is not valid hex")
	}

	if len(blockHashHex) != 32 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid block hash.")
	}

	ex.database.database.Order("slot desc").Limit(30).Where("hash = ?", blockHashHex).Find(&b)

	var blockHash chainhash.Hash
	copy(blockHash[:], blockHashHex)

	var proposerHash chainhash.Hash
	copy(proposerHash[:], b.Proposer)

	var parentBlockHash chainhash.Hash
	copy(parentBlockHash[:], b.ParentBlockHash)

	var stateRoot chainhash.Hash
	copy(stateRoot[:], b.StateRoot)

	block := BlockData{
		Slot:         b.Slot,
		BlockHash:    blockHash.String(),
		ProposerHash: proposerHash.String(),
		ParentBlock:  parentBlockHash.String(),
		StateRoot:    stateRoot.String(),
		RandaoReveal: fmt.Sprintf("%x", b.RandaoReveal),
		Signature:    fmt.Sprintf("%x", b.Signature),
	}

	err = c.Render(http.StatusOK, "block.html", block)

	if err != nil {
		return err
	}
	return err
}
