package transaction

// CasperSlashingTransaction is used for reporting fraud in an attestation.
type CasperSlashingTransaction struct {
	SourceValidators         []uint32
	SourceDataSigned         []byte
	SourceAggregateSignature []byte
	DestinationValidators    []uint32
	DestinationDataSigned    []byte
	DestinationSignature     []byte
}
