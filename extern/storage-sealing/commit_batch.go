package sealing

import (
	"context"
	"sync"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
)

var (
	// TODO: config!

	CommitBatchMax  uint64 = 400 // adjust based on real-world gas numbers
	CommitBatchMin  uint64 = 5
	CommitBatchWait        = 5 * time.Minute
)

type CommitBatcherApi interface {
	SendMsg(ctx context.Context, from, to address.Address, method abi.MethodNum, value, maxFee abi.TokenAmount, params []byte) (cid.Cid, error)
}

type CommitBatcher struct {
	api     CommitBatcherApi
	maddr   address.Address
	mctx    context.Context
	addrSel AddrSel
	feeCfg  FeeConfig

	/*todo map[SectorLocation]*bitfield.BitField // MinerSectorLocation -> BitField

	waiting map[abi.SectorNumber][]chan cid.Cid*/

	notify, stop, stopped chan struct{}
	force                 chan chan *cid.Cid
	lk                    sync.Mutex
}

func NewCommitBatcher(mctx context.Context, maddr address.Address, api CommitBatcherApi, addrSel AddrSel, feeCfg FeeConfig) *CommitBatcher {
	b := &CommitBatcher{
		api:     api,
		maddr:   maddr,
		mctx:    mctx,
		addrSel: addrSel,
		feeCfg:  feeCfg,

		notify:  make(chan struct{}, 1),
		force:   make(chan chan *cid.Cid),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}

	go b.run()

	return b
}

func (b *CommitBatcher) run() {
	var forceRes chan *cid.Cid
	var lastMsg *cid.Cid

	for {
		if forceRes != nil {
			forceRes <- lastMsg
			forceRes = nil
		}
		lastMsg = nil

		var sendAboveMax, sendAboveMin bool
		select {
		case <-b.stop:
			close(b.stopped)
			return
		case <-b.notify:
			sendAboveMax = true
		case <-time.After(TerminateBatchWait):
			sendAboveMin = true
		case fr := <-b.force: // user triggered
			forceRes = fr
		}

		var err error
		lastMsg, err = b.processBatch(sendAboveMax, sendAboveMin)
		if err != nil {
			log.Warnw("TerminateBatcher processBatch error", "error", err)
		}
	}
}

func (b *CommitBatcher) processBatch(notif, after bool) (*cid.Cid, error) {
	return nil, nil
}

func (b *CommitBatcher) Flush(ctx context.Context) (*cid.Cid, error) {
	resCh := make(chan *cid.Cid, 1)
	select {
	case b.force <- resCh:
		select {
		case res := <-resCh:
			return res, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (b *CommitBatcher) Pending(ctx context.Context) ([]abi.SectorID, error) {
	b.lk.Lock()
	defer b.lk.Unlock()

	panic("todo")
}

func (b *CommitBatcher) Stop(ctx context.Context) error {
	close(b.stop)

	select {
	case <-b.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
