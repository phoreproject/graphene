package explorer

import (
	"crypto/rand"
	"encoding/binary"
	"io"
	"os"
	"text/template"
	"time"

	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/phoreproject/prysm/shared/ssz"
	"github.com/phoreproject/synapse/chainhash"
	"github.com/phoreproject/synapse/primitives"

	"github.com/jinzhu/gorm"
	homedir "github.com/mitchellh/go-homedir"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/phoreproject/synapse/beacon"
	"github.com/phoreproject/synapse/beacon/config"
	"github.com/phoreproject/synapse/beacon/db"
	"github.com/phoreproject/synapse/p2p"

	logger "github.com/sirupsen/logrus"

	"github.com/labstack/echo"
)

// Config is the explorer app config.
type Config struct {
	GenesisTime          uint64
	DataDirectory        string
	InitialValidatorList []beacon.InitialValidatorEntry
	NetworkConfig        *config.Config
	Resync               bool
	ListeningAddress     string
	DiscoveryOptions     p2p.DiscoveryOptions
}

// Explorer is a blockchain explorer.
// The explorer streams blocks from the beacon chain as they are received
// and then keeps track of its own blockchain so that it can access more
// info like forking.
type Explorer struct {
	blockchain *beacon.Blockchain

	// P2P
	hostNode    *p2p.HostNode
	syncManager beacon.SyncManager

	config Config

	database *Database
	chainDB  db.Database
}

// NewExplorer creates a new block explorer
func NewExplorer(c Config, gormDB *gorm.DB) (*Explorer, error) {
	return &Explorer{
		database: NewDatabase(gormDB),
		config:   c,
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

	logger.Info("initializing client")

	logger.Info("initializing database")
	database := db.NewBadgerDB(dir)

	if ex.config.Resync {
		logger.Info("dropping all keys in database to resync")
		err := database.Flush()
		if err != nil {
			return err
		}
	}

	ex.chainDB = database

	return nil
}

func (ex *Explorer) loadP2P() error {
	logger.Info("loading P2P")
	addr, err := ma.NewMultiaddr(ex.config.ListeningAddress)
	if err != nil {
		panic(err)
	}

	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}

	hostNode, err := p2p.NewHostNode(addr, pub, priv, ex.config.DiscoveryOptions, 16*time.Second)
	if err != nil {
		panic(err)
	}
	ex.hostNode = hostNode

	logger.Debug("starting peer discovery")
	err = ex.hostNode.StartDiscovery()
	if err != nil {
		panic(err)
	}

	return nil
}

func (ex *Explorer) loadBlockchain() error {
	var genesisTime uint64
	if t, err := ex.chainDB.GetGenesisTime(); err == nil {
		logger.WithField("genesisTime", t).Info("using time from database")
		genesisTime = t
	} else {
		logger.WithField("genesisTime", ex.config.GenesisTime).Info("using time from config")
		err := ex.chainDB.SetGenesisTime(ex.config.GenesisTime)
		if err != nil {
			return err
		}
		genesisTime = ex.config.GenesisTime
	}

	blockchain, err := beacon.NewBlockchainWithInitialValidators(ex.chainDB, ex.config.NetworkConfig, ex.config.InitialValidatorList, true, genesisTime)
	if err != nil {
		panic(err)
	}

	ex.blockchain = blockchain

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

// WaitForConnections waits until beacon app is connected
func (ex *Explorer) WaitForConnections(numConnections int) {
	for {
		if ex.hostNode.PeersConnected() >= numConnections {
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

	epochStart := state.Slot - (state.Slot % ex.config.NetworkConfig.EpochLength)

	ex.database.database.Model(&Epoch{}).Where(&Epoch{StartSlot: epochStart}).Count(&epochCount)

	if epochCount == 0 {
		var assignments []Assignment

		for i := epochStart; i < epochStart+ex.config.NetworkConfig.EpochLength; i++ {
			assignmentForSlot, err := state.GetShardCommitteesAtSlot(state.Slot, i, ex.config.NetworkConfig)
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

	blockHash, err := ssz.TreeHash(block)
	if err != nil {
		panic(err)
	}

	proposerIdx, err := state.GetBeaconProposerIndex(state.Slot, block.BlockHeader.SlotNumber, ex.blockchain.GetConfig())
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

	var attestations []Attestation

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

		attestation := &Attestation{
			ParticipantHashes:   combineHashes(participantHashes),
			Signature:           att.AggregateSig[:],
			Slot:                att.Data.Slot,
			Shard:               att.Data.Shard,
			BeaconBlockHash:     att.Data.BeaconBlockHash[:],
			EpochBoundaryHash:   att.Data.EpochBoundaryHash[:],
			ShardBlockHash:      att.Data.ShardBlockHash[:],
			LatestCrosslinkHash: att.Data.LatestCrosslinkHash[:],
			JustifiedBlockHash:  att.Data.JustifiedBlockHash[:],
			JustifiedSlot:       att.Data.JustifiedSlot,
			BlockID:             blockDB.ID,
		}

		ex.database.database.Create(attestation)

		attestations = append(attestations, *attestation)
	}
}

// StartExplorer starts the block explorer
func (ex *Explorer) StartExplorer() error {
	err := ex.loadDatabase()
	if err != nil {
		return err
	}

	err = ex.loadP2P()
	if err != nil {
		return err
	}

	err = ex.loadBlockchain()
	if err != nil {
		return err
	}

	ex.syncManager = beacon.NewSyncManager(ex.hostNode, time.Second*5, ex.blockchain)

	ex.syncManager.RegisterPostProcessHook(ex.postProcessHook)

	ex.syncManager.Start()

	ex.WaitForConnections(1)

	go ex.syncManager.TryInitialSync()

	go ex.syncManager.ListenForBlocks()

	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	e := echo.New()
	e.Renderer = t

	e.GET("/", ex.renderIndex)
	e.GET("/b/:blockHash", ex.renderBlock)
	e.GET("/v/:validatorHash", ex.renderValidator)

	defer ex.chainDB.Close()

	e.Logger.Fatal(e.Start(":1323"))

	return nil
}
