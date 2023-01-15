package explorer

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/pb"
	"github.com/phoreproject/synapse/primitives"
	"github.com/prysmaticlabs/go-ssz"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jinzhu/gorm"
	"github.com/mitchellh/go-homedir"
	"github.com/phoreproject/synapse/beacon/config"

	"github.com/sirupsen/logrus"

	"github.com/labstack/echo"
)

// Explorer is a blockchain explorer.
// The explorer streams blocks from the beacon chain as they are received
// and then keeps track of its own blockchain so that it can access more
// info like forking.
type Explorer struct {
	config Config

	database   *Database
	beaconRPC  pb.BlockchainRPCClient
	blockchain *beacon.Blockchain

	ctx context.Context

	tipHash chainhash.Hash
}

// NewExplorer creates a new block explorer
func NewExplorer(c Config, gormDB *gorm.DB, beaconRPC pb.BlockchainRPCClient, blockchain *beacon.Blockchain) (*Explorer, error) {
	// todo: replace in memory DB

	return &Explorer{
		database:   NewDatabase(gormDB),
		config:     c,
		beaconRPC:  beaconRPC,
		blockchain: blockchain,
		tipHash:    blockchain.GenesisHash(),
		ctx:        context.TODO(),
	}, nil
}

func (ex *Explorer) loadDatabase() error {
	var dir string
	if ex.config.DataDirectory == "" {
		dataDir, err := config.GetBaseDirectory(true)
		if err != nil {
			panic(err)
		}
		dir = dataDir
	} else {
		d, err := homedir.Expand(ex.config.DataDirectory)
		if err != nil {
			panic(err)
		}
		dir = d
	}

	err := os.MkdirAll(dir, 0777)
	if err != nil {
		panic(err)
	}

	logrus.Info("initializing client")

	return nil
}

// Template is the template engine used by the explorer.
type Template struct {
	templates *template.Template
}

// Render renders the template.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
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

func (ex *Explorer) processBlock(block *primitives.Block) error {
	blockHash, _ := ssz.HashTreeRoot(block)
	logrus.Infof("explorer processing block %s", chainhash.Hash(blockHash))

	epochTransition, state, err := ex.blockchain.ProcessBlock(block, false, true)
	if err != nil {
		return err
	}

	ex.tipHash = blockHash

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

	for _, r := range epochTransition.Receipts {
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

	epochStart := state.EpochIndex * ex.config.NetworkConfig.EpochLength

	ex.database.database.Model(&Epoch{}).Where(&Epoch{StartSlot: epochStart}).Count(&epochCount)

	fmt.Println(epochStart)

	if epochCount == 0 {
		var assignments []Assignment

		for i := epochStart; i < epochStart+ex.config.NetworkConfig.EpochLength; i++ {
			assignmentForSlot, err := state.GetShardCommitteesAtSlot(i, ex.config.NetworkConfig)
			if err != nil {
				panic(err)
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

	proposerIdx, err := state.GetBeaconProposerIndex(block.BlockHeader.SlotNumber-1, ex.config.NetworkConfig)
	if err != nil {
		panic(err)
	}

	var idBytes [4]byte
	binary.BigEndian.PutUint32(idBytes[:], proposerIdx)
	pubAndID := append(state.ValidatorRegistry[proposerIdx].Pubkey[:], idBytes[:]...)
	proposerHash := chainhash.HashH(pubAndID)

	blockDB := &Block{
		ParentBlockHash: block.BlockHeader.ParentRoot[:],
		StateRoot:       block.BlockHeader.StateRoot[:],
		RandaoReveal:    block.BlockHeader.RandaoReveal[:],
		Signature:       block.BlockHeader.Signature[:],
		Hash:            blockHash[:],
		Slot:            block.BlockHeader.SlotNumber,
		Proposer:        proposerHash[:],
	}

	ex.database.database.Create(blockDB)

	// Update attestations
	for _, att := range block.BlockBody.Attestations {
		participants, err := state.GetAttestationParticipants(att.Data, att.ParticipationBitfield, ex.config.NetworkConfig)
		if err != nil {
			panic(err)
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

	return nil
}

func (ex *Explorer) exit() {
	ex.database.database.Close()

	os.Exit(0)
}

const minPollTime = 5 * time.Second

func (ex *Explorer) pollForBlocks() {
	lastUpdate := time.Unix(0, 0)

	for {
		timeSinceUpdate := time.Since(lastUpdate)

		if timeSinceUpdate < minPollTime {
			time.Sleep(minPollTime - timeSinceUpdate)
		}

		select {
		case <-ex.ctx.Done():

		default:
		}

		latestBlockHash, err := ex.beaconRPC.GetLastBlockHash(ex.ctx, &emptypb.Empty{})
		if err != nil {
			logrus.Error(err)
			continue
		}

		if bytes.Equal(latestBlockHash.Hash, ex.tipHash[:]) {
			continue
		}

		blocksToProcess := make([]*primitives.Block, 0)
		protoBlock, err := ex.beaconRPC.GetBlock(ex.ctx, &pb.GetBlockRequest{Hash: latestBlockHash.Hash[:]})
		if err != nil {
			logrus.Error(err)
			continue
		}

		currentBlock, err := primitives.BlockFromProto(protoBlock.Block)
		if err != nil {
			logrus.Error(err)
			continue
		}

		blocksToProcess = append(blocksToProcess, currentBlock)

		for currentBlock.BlockHeader.ParentRoot != ex.tipHash {
			protoBlock, err := ex.beaconRPC.GetBlock(ex.ctx, &pb.GetBlockRequest{Hash: currentBlock.BlockHeader.ParentRoot[:]})
			if err != nil {
				logrus.Error(err)
				continue
			}

			block, err := primitives.BlockFromProto(protoBlock.Block)
			if err != nil {
				logrus.Error(err)
				continue
			}

			currentBlock = block
			blocksToProcess = append(blocksToProcess, currentBlock)
		}

		for i := len(blocksToProcess) - 1; i >= 0; i-- {
			err := ex.processBlock(blocksToProcess[i])
			if err != nil {
				logrus.Error(err)
			}
		}
	}
}

// StartExplorer starts the block explorer
func (ex *Explorer) StartExplorer() error {
	err := ex.loadDatabase()
	if err != nil {
		return err
	}

	signalHandler := make(chan os.Signal, 1)
	signal.Notify(signalHandler, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalHandler

		ex.exit()
	}()

	go ex.pollForBlocks()

	t := &Template{
		templates: template.Must(template.ParseGlob("explorer/templates/*.html")),
	}

	e := echo.New()
	e.Renderer = t

	e.Static("/static", "assets")
	e.GET("/", ex.renderIndex)
	e.GET("/b/:blockHash", ex.renderBlock)
	e.GET("/v/:validatorHash", ex.renderValidator)

	e.Logger.Fatal(e.Start(":1323"))

	return nil
}
