package chain

// import (
// 	"fmt"
// 	"github.com/faiface/pixel"
// 	"github.com/faiface/pixel/imdraw"
// 	"github.com/faiface/pixel/pixelgl"
// 	"github.com/faiface/pixel/text"
// 	"github.com/phoreproject/synapse/chainhash"
// 	"github.com/phoreproject/synapse/primitives"
// 	"github.com/prysmaticlabs/go-ssz"
// 	"golang.org/x/image/colornames"
// 	"golang.org/x/image/font/basicfont"
// 	"sync"
// 	"time"
// )

// var GlobalVis = &visualizer{
// 	shards:     make(map[uint64]*visualizerNotifee, 0),
// 	lastUpdate: time.Unix(0, 0),
// }

// type visualizer struct {
// 	lock       sync.Mutex
// 	shards     map[uint64]*visualizerNotifee
// 	lastUpdate time.Time
// }

// type visualizerNotifee struct {
// 	shardID  uint64
// 	blocks []*primitives.ShardBlock
// 	finalizedHashes map[chainhash.Hash]bool
// 	mgr *ShardManager
// 	lock sync.Mutex
// }

// func (v *visualizerNotifee) AddBlock(block *primitives.ShardBlock, newTip bool) {
// 	v.lock.Lock()
// 	defer v.lock.Unlock()
// 	v.blocks = append(v.blocks, block)

// 	bh, _ := ssz.HashTreeRoot(block)
// 	v.finalizedHashes[bh] = false
// }

// func (v *visualizerNotifee) FinalizeBlockHash(blockHash chainhash.Hash, slot uint64) {
// 	v.lock.Lock()
// 	defer v.lock.Unlock()

// 	v.finalizedHashes[blockHash] = true
// }

// func (v *visualizer) addShard(mgr *ShardManager) {
// 	v.lock.Lock()
// 	defer v.lock.Unlock()

// 	v.shards[mgr.ShardID] = &visualizerNotifee{
// 		shardID: mgr.ShardID,
// 		blocks: []*primitives.ShardBlock{},
// 		finalizedHashes: make(map[chainhash.Hash]bool),
// 		mgr: mgr,
// 	}

// 	mgr.RegisterNotifee(v.shards[mgr.ShardID])
// }

// func (v *visualizer) runWindow() {
// 	cfg := pixelgl.WindowConfig{
// 		Title:  "Shard Visualization",
// 		Bounds: pixel.R(0, 0, 1024, 768),
// 	}
// 	win, err := pixelgl.NewWindow(cfg)
// 	if err != nil {
// 		panic(err)
// 	}

// 	imd := imdraw.New(nil)

// 	basicAtlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
// 	txts := make([]*text.Text, 0, len(v.shards))

// 	fps := time.Tick(time.Second * 1)

// 	for !win.Closed() {
// 		win.Clear(colornames.Skyblue)

// 		currentY := 20

// 		v.lastUpdate = time.Now()
// 		v.lock.Lock()
// 		imd.Clear()

// 		txts = make([]*text.Text, 0, len(v.shards))

// 		for _, mgr := range v.shards {
// 			currentY += 40

// 			basicTxt := text.New(pixel.V(50, float64(currentY)), basicAtlas)
// 			_, _ = fmt.Fprintf(basicTxt, "Shard %d", mgr.mgr.ShardID)

// 			txtBounds := basicTxt.Bounds()

// 			imd.Color = pixel.RGB(0, 0, 1)
// 			imd.Push(pixel.V(txtBounds.Min.X-10, txtBounds.Min.Y-10))
// 			imd.Push(pixel.V(txtBounds.Min.X-10, txtBounds.Max.Y+10))
// 			imd.Push(pixel.V(txtBounds.Max.X+10, txtBounds.Max.Y+10))
// 			imd.Push(pixel.V(txtBounds.Min.X-10, txtBounds.Min.Y-10))
// 			imd.Rectangle(0)

// 			currentX := txtBounds.Max.X + 10

// 			txts = append(txts, basicTxt)

// 			blocksState := mgr.mgr.stateManager.HasAny()

// 			mgr.lock.Lock()
// 			for _, current := range mgr.blocks {
// 				basicTxt := text.New(pixel.V(currentX+10, float64(currentY)), basicAtlas)
// 				_, _ = fmt.Fprintf(basicTxt, "B%d", current.Header.Slot)

// 				txtBounds := basicTxt.Bounds()

// 				bh, _ := ssz.HashTreeRoot(current)

// 				finalized, found := mgr.finalizedHashes[bh]

// 				r := 0.2
// 				if _, found := blocksState[bh]; found {
// 					r = 0.7
// 				}
// 				if !found || !finalized {
// 					imd.Color = pixel.RGB(r, 0.7, 0.2)
// 				} else {
// 					imd.Color = pixel.RGB(r, 0.2, 0.2)
// 				}
// 				imd.Push(pixel.V(txtBounds.Min.X-10, txtBounds.Min.Y-10))
// 				imd.Push(pixel.V(txtBounds.Min.X-10, txtBounds.Max.Y+10))
// 				imd.Push(pixel.V(txtBounds.Max.X+10, txtBounds.Max.Y+10))
// 				imd.Push(pixel.V(txtBounds.Min.X-10, txtBounds.Min.Y-10))
// 				imd.Rectangle(0)

// 				currentX = txtBounds.Max.X + 10

// 				txts = append(txts, basicTxt)
// 			}
// 			mgr.lock.Unlock()
// 		}
// 		v.lock.Unlock()

// 		imd.Draw(win)
// 		for _, txt := range txts {
// 			txt.Draw(win, pixel.IM)
// 		}
// 		win.Update()

// 		<-fps
// 	}
// }

// func (v *visualizer) Start() {
// 	pixelgl.Run(v.runWindow)
// }
