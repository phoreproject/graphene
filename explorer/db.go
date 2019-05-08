package explorer

import (
	"time"

	"github.com/jinzhu/gorm"
)

// Database models:
// - Transaction
// - Block
// - Epoch
// - Attestation
// - Validator
// - Slot

// Validator is a single validator
type Validator struct {
	gorm.Model

	Pubkey                 [96]byte
	WithdrawalCredentials  [32]byte
	Status                 uint64
	LatestStatusChangeSlot uint64
	ExitCount              uint64
	EntrySlot              uint64
	ValidatorID            uint64
	ValidatorHash          [32]byte // Hash(entrySlot || validatorID)
}

// Attestation is a single attestation.
type Attestation struct {
	gorm.Model

	Participants        []Validator
	Signature           [48]byte
	Slot                uint64
	Shard               uint64
	BeaconBlockHash     [32]byte
	EpochBoundaryHash   [32]byte
	ShardBlockHash      [32]byte
	LatestCrosslinkHash [32]byte
	JustifiedSlot       uint64
	JustifiedBlockHash  [32]byte
}

// TODO: add slashings, deposits, and exits

// Block is a single block in the chain.
type Block struct {
	gorm.Model

	Attestations []Attestation
	ParentBlock  *Block
	StateRoot    [32]byte
	RandaoReveal [48]byte
	Signature    [48]byte
	Hash         [32]byte
	Height       uint64
	Slot         uint64
}

// Slot is a slot in the chain.
type Slot struct {
	gorm.Model

	SlotStart time.Time
	Proposer  Validator
	Block     *Block
}

// Transaction is a slashing or reward on the beacon chain.
type Transaction struct {
	gorm.Model

	Amount    uint64
	Recipient Validator
	Type      uint8
	Slot      Slot
}

// Assignment is the assignment of a committee to shards.
type Assignment struct {
	gorm.Model

	Shard               uint64
	Committee           []Validator
	TotalValidatorCount uint64
}

// Epoch is a single epoch in the blockchain.
type Epoch struct {
	gorm.Model

	Blocks                    []Block
	StartSlot                 Slot
	ShardAndCommitteeForSlots [][]Assignment
}

// Database is the database used to store information about the blockchain.
type Database struct {
	database *gorm.DB
}

// GetLatestBlocks gets the latest blocks in the chain.
func (db *Database) GetLatestBlocks(n int) []Block {
	var blocks []Block
	db.database.Limit(n).Order("height desc").Find(&blocks)
	return blocks
}

// NewDatabase creates a new database given a gorm DB.
func NewDatabase(db *gorm.DB) *Database {
	// Migrate the schema
	db.AutoMigrate(&Validator{})
	db.AutoMigrate(&Attestation{})
	db.AutoMigrate(&Block{})
	db.AutoMigrate(&Slot{})
	db.AutoMigrate(&Transaction{})
	db.AutoMigrate(&Assignment{})
	db.AutoMigrate(&Epoch{})

	return &Database{db}
}
