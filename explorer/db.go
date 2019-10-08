package explorer

import (
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

	Pubkey                 []byte `gorm:"size:96"`
	WithdrawalCredentials  []byte `gorm:"size:32"`
	Status                 uint64
	LatestStatusChangeSlot uint64
	ExitCount              uint64
	ValidatorID            uint64
	ValidatorHash          []byte `gorm:"size:32"` // Hash(pubkey || validatorID)
}

// Attestation is a single attestation.
type Attestation struct {
	gorm.Model

	ParticipantHashes   []byte `gorm:"size:16384"`
	Signature           []byte `gorm:"size:48"`
	Slot                uint64
	Shard               uint64
	BeaconBlockHash     []byte `gorm:"size:32"`
	EpochBoundaryHash   []byte `gorm:"size:32"`
	ShardBlockHash      []byte `gorm:"size:32"`
	LatestCrosslinkHash []byte `gorm:"size:32"`
	JustifiedSlot       uint64
	JustifiedBlockHash  []byte `gorm:"size:32"`
	BlockID             uint
}

// TODO: add slashings, deposits, and exits

// Block is a single block in the chain.
type Block struct {
	gorm.Model

	Proposer        []byte `gorm:"size:32"`
	ParentBlockHash []byte `gorm:"size:32"`
	StateRoot       []byte `gorm:"size:32"`
	RandaoReveal    []byte `gorm:"size:48"`
	Signature       []byte `gorm:"size:48"`
	Hash            []byte `gorm:"size:32"`
	Height          uint64
	Slot            uint64
}

// Transaction is a slashing or reward on the beacon chain.
type Transaction struct {
	gorm.Model

	Amount        int64
	RecipientHash []byte `gorm:"size:32"`
	Type          uint8
	Slot          uint64
}

// Assignment is the assignment of a committee to shards.
type Assignment struct {
	gorm.Model

	Shard           uint64
	CommitteeHashes []byte `gorm:"size:16384"`
	Slot            uint64
}

// Epoch is a single epoch in the blockchain.
type Epoch struct {
	gorm.Model

	StartSlot  uint64
	Committees []Assignment
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
	if err := db.AutoMigrate(&Validator{}).Error; err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Block{}).Error; err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Attestation{}).Error; err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Transaction{}).Error; err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Assignment{}).Error; err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&Epoch{}).Error; err != nil {
		panic(err)
	}

	return &Database{db}
}
