package explorer

import (
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo"
)

// ValidatorData is the data of a validator passed to explorer templates.
type ValidatorData struct {
	Pubkey                 string
	WithdrawalCredentials  string
	Status                 uint64
	LatestStatusChangeSlot uint64
	ExitCount              uint64
	ValidatorID            uint64
	ValidatorHash          string
	Attestations           []AttestationData
}

func (ex *Explorer) renderValidator(c echo.Context) error {
	var v Validator

	validatorHashStr := c.Param("validatorHash")

	validatorHashHex, err := hex.DecodeString(validatorHashStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Block hash is not valid hex")
	}

	if len(validatorHashHex) != 32 {
		return echo.NewHTTPError(http.StatusBadRequest, "Please provide a valid block hash.")
	}

	ex.database.database.Where(&Validator{ValidatorHash: validatorHashHex}).First(&v)

	//var attestations []Attestation
	//ex.database.database.Where("participant_hashes LIKE ?", validatorHashHex).Find(&attestations)
	//
	//attestationsData := make([]AttestationData, len(attestations))
	//for i := range attestations {
	//	participantHashesSeparate := splitHashes(attestations[i].ParticipantHashes)
	//
	//	participantHashes := make([]string, len(participantHashesSeparate[i]))
	//
	//	for j := range participantHashesSeparate {
	//		participantHashes[j] = chainhash.Hash(participantHashesSeparate[j]).String()
	//	}
	//
	//	attestationsData[i] = AttestationData{
	//		ParticipantHashes:   participantHashes,
	//		Signature:           hex.EncodeToString(attestations[i].Signature),
	//		Slot:                attestations[i].Slot,
	//		Shard:               attestations[i].Shard,
	//		BeaconBlockHash:     hex.EncodeToString(attestations[i].BeaconBlockHash),
	//		EpochBoundaryHash:   hex.EncodeToString(attestations[i].EpochBoundaryHash),
	//		ShardBlockHash:      hex.EncodeToString(attestations[i].ShardBlockHash),
	//		LatestCrosslinkHash: hex.EncodeToString(attestations[i].LatestCrosslinkHash),
	//		JustifiedBlockHash:  hex.EncodeToString(attestations[i].JustifiedBlockHash),
	//		JustifiedSlot:       attestations[i].JustifiedSlot,
	//	}
	//}

	validator := ValidatorData{
		Pubkey:                 hex.EncodeToString(v.Pubkey),
		WithdrawalCredentials:  hex.EncodeToString(v.WithdrawalCredentials),
		Status:                 v.Status,
		LatestStatusChangeSlot: v.LatestStatusChangeSlot,
		ExitCount:              v.ExitCount,
		ValidatorID:            v.ValidatorID,
		ValidatorHash:          hex.EncodeToString(validatorHashHex),
		//Attestations:           attestationsData,
	}

	err = c.JSON(http.StatusOK, validator)

	if err != nil {
		return err
	}
	return err
}
