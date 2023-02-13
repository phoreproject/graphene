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
	Attestations []AttestationData
	Timestamp    uint64
	Height       uint64
}

// AttestationData is the data of an attestation that gets passed to templates.
type AttestationData struct {
	ParticipantHashes   []string
	Signature           string
	Slot                uint64
	Shard               uint64
	BeaconBlockHash     string
	EpochBoundaryHash   string
	ShardBlockHash      string
	LatestCrosslinkHash string
	JustifiedSlot       uint64
	JustifiedBlockHash  string
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

	ex.database.database.Order("slot desc").Limit(30).Where(&Block{Hash: blockHashHex}).First(&b)

	var attestations []Attestation
	ex.database.database.Where(&Attestation{BlockID: b.ID}).Find(&attestations)

	attestationsData := make([]AttestationData, len(attestations))
	for i := range attestations {
		participantHashesSeparate := splitHashes(attestations[i].ParticipantHashes)

		participantHashes := make([]string, len(participantHashesSeparate[i]))

		for j := range participantHashesSeparate {
			participantHashes[j] = chainhash.Hash(participantHashesSeparate[j]).String()
		}

		attestationsData[i] = AttestationData{
			ParticipantHashes:   participantHashes,
			Signature:           hex.EncodeToString(attestations[i].Signature),
			Slot:                attestations[i].Slot,
			Shard:               attestations[i].Shard,
			BeaconBlockHash:     hex.EncodeToString(attestations[i].BeaconBlockHash),
			EpochBoundaryHash:   hex.EncodeToString(attestations[i].EpochBoundaryHash),
			ShardBlockHash:      hex.EncodeToString(attestations[i].ShardBlockHash),
			LatestCrosslinkHash: hex.EncodeToString(attestations[i].LatestCrosslinkHash),
			JustifiedBlockHash:  hex.EncodeToString(attestations[i].JustifiedBlockHash),
			JustifiedSlot:       attestations[i].JustifiedSlot,
		}
	}

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
		Timestamp:    b.Timestamp,
		StateRoot:    stateRoot.String(),
		RandaoReveal: fmt.Sprintf("%x", b.RandaoReveal),
		Signature:    fmt.Sprintf("%x", b.Signature),
		Attestations: attestationsData,
		Height:       b.Height,
	}

	err = c.JSON(http.StatusOK, block)

	if err != nil {
		return err
	}
	return err
}
