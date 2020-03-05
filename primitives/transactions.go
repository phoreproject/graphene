package primitives

import (
	"fmt"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
)


// CombineHashes combines two branches of the tree.
func CombineHashes(left *chainhash.Hash, right *chainhash.Hash) chainhash.Hash {
	return chainhash.HashH(append(left[:], right[:]...))
}

// EmptyTrees are empty trees for each level.
var EmptyTrees [256]chainhash.Hash

// EmptyTree is the hash of an empty tree.
var EmptyTree = chainhash.Hash{}

func init() {
	EmptyTrees[0] = zeroHash
	for i := range EmptyTrees[1:] {
		EmptyTrees[i+1] = CombineHashes(&EmptyTrees[i], &EmptyTrees[i])
	}

	EmptyTree = EmptyTrees[255]
}


// UpdateWitness allows an executor to securely update the tree root so that only a single key is changed.
type UpdateWitness struct {
	Key             chainhash.Hash
	OldValue        chainhash.Hash
	NewValue        chainhash.Hash
	WitnessBitfield chainhash.Hash
	LastLevel       uint8
	Witnesses       []chainhash.Hash
}

// UpdateWitnessFromProto converts a protobuf representation of an update witness into an update witness.
func UpdateWitnessFromProto(uw *pb.UpdateWitness) (*UpdateWitness, error) {
	if uw.LastLevel > 255 {
		return nil, fmt.Errorf("expected LastLevel to be under 255")
	}

	witnesses := make([]chainhash.Hash, len(uw.WitnessHashes))
	for i := range witnesses {
		err := witnesses[i].SetBytes(uw.WitnessHashes[i])
		if err != nil {
			return nil, err
		}
	}

	out := &UpdateWitness{
		Witnesses: witnesses,
		LastLevel: uint8(uw.LastLevel),
	}

	err := out.Key.SetBytes(uw.Key)
	if err != nil {
		return nil, err
	}

	err = out.OldValue.SetBytes(uw.OldValue)
	if err != nil {
		return nil, err
	}

	err = out.NewValue.SetBytes(uw.NewValue)
	if err != nil {
		return nil, err
	}

	err = out.WitnessBitfield.SetBytes(uw.WitnessBitfield)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// ToProto converts the UpdateWitness into protobuf form.
func (uw *UpdateWitness) ToProto() *pb.UpdateWitness {
	out := &pb.UpdateWitness{
		Key: uw.Key[:],
		OldValue: uw.OldValue[:],
		NewValue: uw.NewValue[:],
		WitnessBitfield: uw.WitnessBitfield[:],
		LastLevel: uint32(uw.LastLevel),
		WitnessHashes: make([][]byte, len(uw.Witnesses)),
	}

	for i := range out.WitnessHashes {
		out.WitnessHashes[i] = uw.Witnesses[i][:]
	}

	return out
}

// Copy returns a copy of the update witness.
func (uw *UpdateWitness) Copy() UpdateWitness {
	newUw := *uw

	newUw.Witnesses = make([]chainhash.Hash, len(uw.Witnesses))
	for i := range newUw.Witnesses {
		copy(newUw.Witnesses[i][:], uw.Witnesses[i][:])
	}

	return newUw
}


// VerificationWitness allows an executor to verify a specific node in the tree.
type VerificationWitness struct {
	Key             chainhash.Hash
	Value           chainhash.Hash
	WitnessBitfield chainhash.Hash
	Witnesses       []chainhash.Hash
	LastLevel       uint8
}

// VerificationWitnessFromProto converts a protobuf representation of a verification witness.
func VerificationWitnessFromProto(vw *pb.VerificationWitness) (*VerificationWitness, error) {
	if vw.LastLevel > 255 {
		return nil, fmt.Errorf("LastLevel should be under 255")
	}

	witnesses := make([]chainhash.Hash, len(vw.WitnessHashes))
	for i := range witnesses {
		err := witnesses[i].SetBytes(vw.WitnessHashes[i])
		if err != nil {
			return nil, err
		}
	}

	out := &VerificationWitness{
		LastLevel: uint8(vw.LastLevel),
		Witnesses: witnesses,
	}

	err := out.Key.SetBytes(vw.Key)
	if err != nil {
		return nil, err
	}

	err = out.Value.SetBytes(vw.Value)
	if err != nil {
		return nil, err
	}

	err = out.WitnessBitfield.SetBytes(vw.WitnessBitfield)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// ToProto converts the verification witness into protobuf form.
func (vw *VerificationWitness) ToProto() *pb.VerificationWitness {
	out := &pb.VerificationWitness{
		Key: vw.Key[:],
		Value: vw.Value[:],
		WitnessBitfield: vw.WitnessBitfield[:],
		LastLevel: uint32(vw.LastLevel),
		WitnessHashes: make([][]byte, len(vw.Witnesses)),
	}

	for i := range out.WitnessHashes {
		out.WitnessHashes[i] = vw.Witnesses[i][:]
	}

	return out
}

// Copy returns a copy of the update witness.
func (vw *VerificationWitness) Copy() VerificationWitness {
	newVw := *vw

	newVw.Witnesses = make([]chainhash.Hash, len(vw.Witnesses))
	for i := range newVw.Witnesses {
		copy(newVw.Witnesses[i][:], vw.Witnesses[i][:])
	}

	return newVw
}

// TransactionPackage is a way to update the state root without having the entire state.
type TransactionPackage struct {
	StartRoot chainhash.Hash
	EndRoot chainhash.Hash
	Updates []UpdateWitness
	Verifications []VerificationWitness
	Transactions []ShardTransaction
}

// Copy copies the transaction package.
func (tp *TransactionPackage) Copy() TransactionPackage {
	newTp := *tp

	newTp.Updates = make([]UpdateWitness, len(tp.Updates))
	newTp.Verifications = make([]VerificationWitness, len(tp.Verifications))
	newTp.Transactions = make([]ShardTransaction, len(tp.Transactions))

	for i := range newTp.Updates {
		newTp.Updates[i] = tp.Updates[i].Copy()
	}

	for i := range newTp.Verifications {
		newTp.Verifications[i] = tp.Verifications[i].Copy()
	}

	for i := range newTp.Transactions {
		newTp.Transactions[i] = tp.Transactions[i].Copy()
	}

	return newTp
}

// ToProto converts a transaction package into the protobuf representation.
func (tp *TransactionPackage) ToProto() *pb.TransactionPackage {
	out := &pb.TransactionPackage{
		StartRoot: tp.StartRoot[:],
		EndRoot: tp.EndRoot[:],
		UpdateWitnesses: make([]*pb.UpdateWitness, len(tp.Updates)),
		VerificationWitnesses: make([]*pb.VerificationWitness, len(tp.Verifications)),
		Transactions: make([]*pb.ShardTransaction, len(tp.Transactions)),
	}

	for i := range out.UpdateWitnesses {
		out.UpdateWitnesses[i] = tp.Updates[i].ToProto()
	}

	for i := range out.VerificationWitnesses {
		out.VerificationWitnesses[i] = tp.Verifications[i].ToProto()
	}

	for i := range out.Transactions {
		out.Transactions[i] = tp.Transactions[i].ToProto()
	}

	return out
}

// TransactionPackageFromProto converts a protobuf representation of a transaction package to normal form.
func TransactionPackageFromProto(tp *pb.TransactionPackage) (*TransactionPackage, error) {
	outTp := &TransactionPackage{
		Updates: make([]UpdateWitness, len(tp.UpdateWitnesses)),
		Verifications: make([]VerificationWitness, len(tp.VerificationWitnesses)),
		Transactions: make([]ShardTransaction, len(tp.Transactions)),
	}

	err := outTp.StartRoot.SetBytes(tp.StartRoot)
	if err != nil {
		return nil, err
	}

	err = outTp.EndRoot.SetBytes(tp.EndRoot)
	if err != nil {
		return nil, err
	}

	for i := range outTp.Updates {
		u, err := UpdateWitnessFromProto(tp.UpdateWitnesses[i])
		if err != nil {
			return nil, err
		}
		outTp.Updates[i] = *u
	}

	for i := range outTp.Verifications {
		v, err := VerificationWitnessFromProto(tp.VerificationWitnesses[i])
		if err != nil {
			return nil, err
		}
		outTp.Verifications[i] = *v
	}

	for i := range outTp.Transactions {
		t, err := ShardTransactionFromProto(tp.Transactions[i])
		if err != nil {
			return nil, err
		}
		outTp.Transactions[i] = *t
	}

	return outTp, nil
}