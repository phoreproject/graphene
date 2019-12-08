package explorer

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"

	"github.com/jinzhu/gorm"
	// blank import
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/phoreproject/synapse/beacon/config"
	beaconModule "github.com/phoreproject/synapse/beacon/module"
	shardChain "github.com/phoreproject/synapse/shard/chain"
	shardModule "github.com/phoreproject/synapse/shard/module"
	logger "github.com/sirupsen/logrus"
)

// Explorer is a blockchain explorer.
// The explorer streams blocks from the beacon chain as they are received
// and then keeps track of its own blockchain so that it can access more
// info like forking.
type Explorer struct {
	beaconApp *beaconModule.BeaconApp
	shardApp  *shardModule.ShardApp

	config *Config

	db *gorm.DB

	database *Database
}

func createDb(c *Config) *gorm.DB {
	var db *gorm.DB
	var err error
	switch c.DbDriver {
	case "sqlite":
		db, err = gorm.Open("sqlite3", c.DbDatabase)
		break

	case "mysql":
		passwordText := ""
		if c.DbPassword != "" {
			passwordText = fmt.Sprintf(":%s", c.DbPassword)
		}
		dbText := fmt.Sprintf(
			"%s%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.DbUser,
			passwordText,
			c.DbHost,
			c.DbDatabase)
		db, err = gorm.Open("mysql", dbText)
		break

	case "postgres":
		db, err = gorm.Open(
			"postgres",
			fmt.Sprintf(
				"host=%s port=5432 user=%s dbname=%s password=%s",
				c.DbHost,
				c.DbUser,
				c.DbDatabase,
				c.DbPassword))
		break
	}
	if err != nil {
		panic(err)
	}
	return db
}

// NewExplorer creates a new block explorer
func NewExplorer(c *Config) (*Explorer, error) {
	db := createDb(c)

	beaconConfig := config.Options{}
	beaconConfig.Resync = c.Resync
	beaconConfig.ChainCFG = c.ChainConfig
	beaconConfig.DataDir = c.DataDir
	beaconConfig.GenesisTime = strconv.FormatUint(c.beaconConfig.GenesisTime, 10)
	beaconConfig.InitialConnections = strings.Split(c.Connect, ",")
	beaconConfig.P2PListen = c.Listen
	beaconConfig.RPCListen = c.Listen
	beaconApp, err := beaconModule.NewBeaconApp(beaconConfig)
	if err != nil {
		panic(err)
	}

	shardApp, err := shardModule.NewShardApp(*c.shardConfig)
	if err != nil {
		panic(err)
	}

	ex := &Explorer{
		beaconApp: beaconApp,
		shardApp:  shardApp,
		db:        db,
		database:  NewDatabase(db),
		config:    c,
	}
	lvl, err := logger.ParseLevel(c.Level)
	if err != nil {
		panic(err)
	}
	logger.SetLevel(lvl)
	return ex, nil
}

// WaitForConnections waits until beacon app is connected
func (ex *Explorer) WaitForConnections(numConnections int) {
	for {
		if ex.beaconApp.GetHostNode().PeersConnected() >= numConnections {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func combineHashes(in [][32]byte) []byte {
	out := make([]byte, 32*len(in))

	for i, h := range in {
		copy(out[32*i:32*(i+1)], h[:])
	}

	return out
}

func splitHashes(in []byte) [][32]byte {
	out := make([][32]byte, len(in)/32)

	for i := range out {
		copy(out[i][:], in[32*i:32*(i+1)])
	}

	return out
}

func (ex *Explorer) postProcessHook(block *primitives.Block, state *primitives.State, receipts []primitives.Receipt) {
	validators := make(map[int]Validator)

	// Update Validators
	for id, v := range state.ValidatorRegistry {
		var idBytes [4]byte
		binary.BigEndian.PutUint32(idBytes[:], uint32(id))
		pubAndID := append(v.Pubkey[:], idBytes[:]...)
		validatorHash := chainhash.HashH(pubAndID)

		var newV Validator

		ex.database.database.Where(Validator{ValidatorHash: validatorHash[:]}).FirstOrCreate(&newV)

		newV.Pubkey = v.Pubkey[:]
		newV.WithdrawalCredentials = v.WithdrawalCredentials[:]
		newV.Status = v.Status
		newV.LatestStatusChangeSlot = v.LatestStatusChangeSlot
		newV.ExitCount = v.ExitCount
		newV.ValidatorID = uint64(id)

		ex.database.database.Save(&newV)

		validators[id] = newV
	}

	for _, r := range receipts {
		var idBytes [4]byte
		binary.BigEndian.PutUint32(idBytes[:], r.Index)
		pubAndID := append(state.ValidatorRegistry[r.Index].Pubkey[:], idBytes[:]...)
		validatorHash := chainhash.HashH(pubAndID)

		receipt := &Transaction{
			Amount:        r.Amount,
			RecipientHash: validatorHash[:],
			Type:          r.Type,
			Slot:          r.Slot,
		}

		if receipt.Amount > 0 {
			ex.database.database.Create(receipt)
		}
	}

	var epochCount int

	epochLength := ex.config.beaconConfig.NetworkConfig.EpochLength
	//epochStart := state.Slot - (state.Slot % epochLength)
	epochStart := state.EpochIndex * epochLength

	ex.database.database.Model(&Epoch{}).Where(&Epoch{StartSlot: epochStart}).Count(&epochCount)

	if epochCount == 0 {
		var assignments []Assignment

		for i := epochStart; i < epochStart+epochLength; i++ {
			assignmentForSlot, err := state.GetShardCommitteesAtSlot(i, ex.config.beaconConfig.NetworkConfig)
			if err != nil {
				logger.Errorf("%v epochStart=%d", err, epochStart)
				continue
			}

			for _, as := range assignmentForSlot {
				committeeHashes := make([][32]byte, len(as.Committee))
				for i, member := range as.Committee {
					var idBytes [4]byte
					binary.BigEndian.PutUint32(idBytes[:], member)
					pubAndID := append(state.ValidatorRegistry[member].Pubkey[:], idBytes[:]...)
					committeeHashes[i] = chainhash.HashH(pubAndID)
				}

				assignment := &Assignment{
					Shard:           as.Shard,
					Slot:            i,
					CommitteeHashes: combineHashes(committeeHashes),
				}

				ex.database.database.Create(assignment)

				assignments = append(assignments, *assignment)
			}
		}

		ex.database.database.Create(&Epoch{
			StartSlot:  epochStart,
			Committees: assignments,
		})
	}

	blockHash, err := ssz.HashTreeRoot(block)
	if err != nil {
		logger.Errorf("%v", err)
	}

	proposerIdx, err := state.GetBeaconProposerIndex(block.BlockHeader.SlotNumber-1, ex.config.beaconConfig.NetworkConfig)
	if err != nil {
		logger.Errorf("%v", err)
	}

	var idBytes [4]byte
	binary.BigEndian.PutUint32(idBytes[:], proposerIdx)
	pubAndID := append(state.ValidatorRegistry[proposerIdx].Pubkey[:], idBytes[:]...)
	proposerHash := chainhash.HashH(pubAndID)

	blockDB := &Block{
		ParentBlockHash:   block.BlockHeader.ParentRoot[:],
		StateRoot:         block.BlockHeader.StateRoot[:],
		RandaoReveal:      block.BlockHeader.RandaoReveal[:],
		Signature:         block.BlockHeader.Signature[:],
		Hash:              blockHash[:],
		Slot:              block.BlockHeader.SlotNumber,
		Proposer:          proposerHash[:],
		Attestations:      uint32(len(block.BlockBody.Attestations)),
		ProposerSlashings: uint32(len(block.BlockBody.ProposerSlashings)),
		CasperSlashings:   uint32(len(block.BlockBody.CasperSlashings)),
		Deposits:          uint32(len(block.BlockBody.Deposits)),
		Exits:             uint32(len(block.BlockBody.Exits)),
		Votes:             uint32(len(block.BlockBody.Votes)),
	}

	ex.database.database.Create(blockDB)

	// Update attestations
	for _, att := range block.BlockBody.Attestations {
		participants, err := state.GetAttestationParticipants(att.Data, att.ParticipationBitfield, ex.config.beaconConfig.NetworkConfig)
		if err != nil {
			logger.Errorf("%v", err)
			continue
		}

		participantHashes := make([][32]byte, len(participants))

		for i, p := range participants {
			var idBytes [4]byte
			binary.BigEndian.PutUint32(idBytes[:], p)
			pubAndID := append(state.ValidatorRegistry[p].Pubkey[:], idBytes[:]...)
			validatorHash := chainhash.HashH(pubAndID)

			participantHashes[i] = validatorHash
		}

		// TODO: fixme

		attestation := &Attestation{
			ParticipantHashes:   combineHashes(participantHashes),
			Signature:           att.AggregateSig[:],
			Slot:                att.Data.Slot,
			Shard:               att.Data.Shard,
			BeaconBlockHash:     att.Data.BeaconBlockHash[:],
			ShardBlockHash:      att.Data.ShardBlockHash[:],
			LatestCrosslinkHash: att.Data.LatestCrosslinkHash[:],
			BlockID:             blockDB.ID,
		}

		ex.database.database.Create(attestation)
	}
}

func (ex *Explorer) doProcessShardBlocks(shardManager *shardChain.ShardManager) {
	chain := shardManager.Chain
	tip, _ := chain.Tip()
	if tip == nil {
		return
	}
	currentSlot := tip.Slot
	for {
		var foundSlotCount int
		ex.database.database.Model(&ShardBlock{}).Where(&ShardBlock{ShardID: shardManager.ShardID, Slot: currentSlot}).Count(&foundSlotCount)
		if foundSlotCount != 0 {
			break
		}

		block, _ := chain.GetNodeBySlot(currentSlot)
		if block == nil {
			break
		}
		ex.database.database.Create(&ShardBlock{
			ShardID:       shardManager.ShardID,
			BlockHash:     block.BlockHash[:],
			StateRootHash: block.StateRoot[:],
			Slot:          block.Slot,
			Height:        block.Height,
		})

		currentSlot--
	}
}

func (ex *Explorer) doProcessSingleShard(shardManager *shardChain.ShardManager) {
	var shardCount int
	ex.database.database.Model(&Shard{}).Where(&Shard{ShardID: shardManager.ShardID}).Count(&shardCount)
	if shardCount == 0 {
		ex.database.database.Create(&Shard{
			ShardID:       shardManager.ShardID,
			RootBlockHash: shardManager.InitializationParameters.RootBlockHash.CloneBytes(),
			RootSlot:      shardManager.InitializationParameters.RootSlot,
			GenesisTime:   shardManager.InitializationParameters.GenesisTime,
		})
	}

	ex.doProcessShardBlocks(shardManager)
}

func (ex *Explorer) doProcessShards() {
	mux := ex.shardApp.Mux
	if mux == nil {
		return
	}

	shardIDList := mux.GetShardIDList()
	for _, shardID := range shardIDList {
		shardManager, _ := mux.GetManager(shardID)
		if shardManager == nil {
			continue
		}
		ex.doProcessSingleShard(shardManager)
	}
}

func (ex *Explorer) processShards() {
	for {
		ex.doProcessShards()
		time.Sleep(5)
	}
}

func (ex *Explorer) exit() {
	ex.beaconApp.Exit()

	os.Exit(0)
}

// StartExplorer starts the block explorer
func (ex *Explorer) StartExplorer() error {
	logger.Info("StartExplorer 1")

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalHandler

		ex.exit()
	}()

	ex.beaconApp.GetSyncManager().RegisterPostProcessHook(ex.postProcessHook)

	go ex.processShards()

	logger.Info("Start shard chains.")
	ex.shardApp.Run()

	// temp
	rootHash, _ := chainhash.NewHashFromStr("4b511a3448fd23a25f81eecfde8d0ef9747e3f4d183cba6a16ec4bd893930d60")
	ex.shardApp.Mux.StartManaging(1, shardChain.ShardChainInitializationParameters{
		RootBlockHash: *rootHash,
		RootSlot:      1,
	})

	logger.Info("Start Beacon chain.")
	ex.beaconApp.Run()

	return nil
}
