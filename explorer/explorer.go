package explorer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/phoreproject/synapse/primitives"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/phoreproject/synapse/pb"
	"google.golang.org/grpc"

	"github.com/labstack/echo"
)

// AttestationInfo is the info of an attestation used by the explorer.
type AttestationInfo struct {
	ShardBlockHash string
	Shard          uint64
	Slot           uint64
	JustifiedSlot  uint64
}

// SlotInfo is the info of a slot used by the explorer.
type SlotInfo struct {
	Slot         uint64
	Proposer     uint64
	BlockHash    string
	Attestations []AttestationInfo
}

// Explorer is a blockchain explorer.
type Explorer struct {
	slotData          []IndexSlotData
	slotInfo          map[uint64]SlotInfo
	currentSlotNumber int64
	blockchainRPC     pb.BlockchainRPCClient
}

// NewExplorer creates a new block explorer
func NewExplorer(blockchainConn *grpc.ClientConn) *Explorer {
	return &Explorer{
		slotData:          make([]IndexSlotData, 0),
		slotInfo:          make(map[uint64]SlotInfo),
		currentSlotNumber: -1,
		blockchainRPC:     pb.NewBlockchainRPCClient(blockchainConn),
	}
}

// Template is the template engine used by the explorer.
type Template struct {
	templates *template.Template
}

// Render renders the template.
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// IndexSlotData is the slot data used by the index page.
type IndexSlotData struct {
	Slot      uint64
	Block     bool
	BlockHash string
	Proposer  uint32
	Rewards   int64
}

// IndexData is the data used by the index page.
type IndexData struct {
	Slots []IndexSlotData
}

// UpdateData updates the data if needed.
func (ex *Explorer) UpdateData() error {
	slotNumber, err := ex.blockchainRPC.GetSlotNumber(context.Background(), &empty.Empty{})
	if err != nil {
		return err
	}

	if int64(slotNumber.SlotNumber) != ex.currentSlotNumber {
		ex.currentSlotNumber = int64(slotNumber.SlotNumber)

		blockHash, err := ex.blockchainRPC.GetBlockHash(context.Background(), &pb.GetBlockHashRequest{
			SlotNumber: slotNumber.SlotNumber,
		})
		if err != nil {
			return err
		}

		proposer, err := ex.blockchainRPC.GetProposerForSlot(context.Background(), &pb.GetProposerForSlotRequest{
			Slot: slotNumber.SlotNumber,
		})
		if err != nil {
			return err
		}

		ex.slotData = append([]IndexSlotData{
			{
				Slot:      slotNumber.SlotNumber,
				Block:     true,
				BlockHash: fmt.Sprintf("%x", blockHash.Hash),
				Proposer:  proposer.Proposer,
				Rewards:   0,
			},
		}, ex.slotData...)

		blockData, err := ex.blockchainRPC.GetBlock(context.Background(), &pb.GetBlockRequest{
			Hash: blockHash.Hash,
		})
		if err != nil {
			return err
		}

		block, err := primitives.BlockFromProto(blockData.Block)
		if err != nil {
			return err
		}

		ex.slotInfo[slotNumber.SlotNumber] = SlotInfo{
			Slot:         slotNumber.SlotNumber,
			Proposer:     uint64(proposer.Proposer),
			BlockHash:    fmt.Sprintf("%x", blockHash.Hash),
			Attestations: make([]AttestationInfo, len(block.BlockBody.Attestations)),
		}

		for i := range block.BlockBody.Attestations {
			ex.slotInfo[slotNumber.SlotNumber].Attestations[i] = AttestationInfo{
				ShardBlockHash: block.BlockBody.Attestations[i].Data.ShardBlockHash.String(),
				Shard:          block.BlockBody.Attestations[i].Data.Shard,
				Slot:           block.BlockBody.Attestations[i].Data.Slot,
				JustifiedSlot:  block.BlockBody.Attestations[i].Data.JustifiedSlot,
			}
		}

		if len(ex.slotData) > 30 {
			ex.slotData = ex.slotData[:30]
		}
	}
	return nil
}

// StartExplorer starts the block explorer
func (ex *Explorer) StartExplorer() error {
	t := &Template{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		for {
			err := ex.UpdateData()
			if err != nil {
				panic(err)
			}
			<-ticker.C
		}
	}()

	e := echo.New()
	e.Renderer = t

	e.GET("/", func(c echo.Context) error {
		err := c.Render(http.StatusOK, "index.html", IndexData{ex.slotData})
		if err != nil {
			fmt.Println(err)
		}
		return err
	})

	e.GET("/s/:slot", func(c echo.Context) error {
		slotNumberString := c.Param("slot")
		slot, err := strconv.Atoi(slotNumberString)
		if err != nil {
			return err
		}

		slotInfo := ex.slotInfo[uint64(slot)]

		err = c.Render(http.StatusOK, "slot.html", slotInfo)
		if err != nil {
			fmt.Println(err)
		}
		return err
	})
	e.Logger.Fatal(e.Start(":1323"))

	return nil
}
