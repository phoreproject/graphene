package transaction

import (
	"github.com/phoreproject/synapse/bls"
)

// CasperSlashingTransaction is used for reporting fraud in an attestation.
type CasperSlashingTransaction struct {
	SourceValidators         []uint32
	SourceDataSigned         []byte
	SourceAggregateSignature bls.Signature
	DestinationValidators    []uint32
	DestinationDataSigned    []byte
	DestinationSignature     bls.Signature
}
