package primitives

import (
	"errors"
	"sync"

	"github.com/phoreproject/synapse/bls"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)

const (
	// Active is a status for a validator that is active.
	Active = iota
	// ActivePendingExit is a status for a validator that is active but pending exit.
	ActivePendingExit
	// PendingActivation is a status for a newly added validator
	PendingActivation
	// ExitedWithoutPenalty is a validator that gracefully exited
	ExitedWithoutPenalty
	// ExitedWithPenalty is a validator that exited not-so-gracefully
	ExitedWithPenalty
)

var pubkeyCache = make(map[[96]byte]bls.PublicKey)
var pubkeyCacheLock = new(sync.RWMutex)

func lookupPubkey(pk [96]byte) *bls.PublicKey {
	pubkeyCacheLock.RLock()
	out, found := pubkeyCache[pk]
	pubkeyCacheLock.RUnlock()
	if found {
		return &out
	}
	return nil
}

func setPubkey(pkSer [96]byte, pub *bls.PublicKey) {
	pubkeyCacheLock.Lock()
	pubkeyCache[pkSer] = *pub
	pubkeyCacheLock.Unlock()
}

// Validator is a single validator session (logging in and out)
type Validator struct {
	// BLS public key
	Pubkey [96]byte
	// Withdrawal credentials
	WithdrawalCredentials chainhash.Hash
	// Status code
	Status uint64
	// Slot when validator last changed status (or 0)
	LatestStatusChangeSlot uint64
	// Sequence number when validator exited (or 0)
	ExitCount uint64
	// LastPoCChangeSlot is the last time the PoC was changed
	LastPoCChangeSlot uint64
	// SecondLastPoCChangeSlot is the second to last time the PoC was changed
	SecondLastPoCChangeSlot uint64
}

// GetPublicKey gets the cached validator pubkey.
func (v *Validator) GetPublicKey() (*bls.PublicKey, error) {
	if pub := lookupPubkey(v.Pubkey); pub != nil {
		return pub, nil
	}

	pub, err := bls.DeserializePublicKey(v.Pubkey)
	if err != nil {
		return nil, err
	}

	setPubkey(v.Pubkey, pub)

	return pub, nil
}

// Copy copies a validator instance.
func (v *Validator) Copy() Validator {
	return *v
}

// IsActive checks if the validator is active.
func (v Validator) IsActive() bool {
	return v.Status == Active || v.Status == ActivePendingExit
}

// ValidatorFromProto gets the validator for the protobuf representation
func ValidatorFromProto(validator *pb.Validator) (*Validator, error) {
	if len(validator.Pubkey) != 96 {
		return nil, errors.New("validator pubkey should be 96 bytes")
	}
	v := &Validator{
		Status:                  validator.Status,
		LatestStatusChangeSlot:  validator.LatestStatusChangeSlot,
		ExitCount:               validator.LatestStatusChangeSlot,
		LastPoCChangeSlot:       validator.LastPoCChangeSlot,
		SecondLastPoCChangeSlot: validator.SecondLastPoCChangeSlot,
	}
	err := v.WithdrawalCredentials.SetBytes(validator.WithdrawalCredentials)
	if err != nil {
		return nil, err
	}
	copy(v.Pubkey[:], validator.Pubkey)

	return v, nil
}

// GetActiveValidatorIndices gets validator indices that are active.
func GetActiveValidatorIndices(validators []Validator) []uint32 {
	var active []uint32
	for i, v := range validators {
		if v.IsActive() {
			active = append(active, uint32(i))
		}
	}
	return active
}

// ToProto creates a ProtoBuf ValidatorResponse from a Validator
func (v *Validator) ToProto() *pb.Validator {
	return &pb.Validator{
		Pubkey:                  v.Pubkey[:],
		WithdrawalCredentials:   v.WithdrawalCredentials[:],
		LastPoCChangeSlot:       v.LastPoCChangeSlot,
		SecondLastPoCChangeSlot: v.SecondLastPoCChangeSlot,
		Status:                  v.Status,
		LatestStatusChangeSlot:  v.LatestStatusChangeSlot,
		ExitCount:               v.ExitCount}
}
