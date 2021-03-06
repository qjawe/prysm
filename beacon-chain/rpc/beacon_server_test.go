package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

var closedContext = "context closed"

type faultyPOWChainService struct {
	chainStartFeed *event.Feed
	hashesByHeight map[int][]byte
}

func (f *faultyPOWChainService) HasChainStartLogOccurred() (bool, uint64, error) {
	return false, uint64(time.Now().Unix()), nil
}
func (f *faultyPOWChainService) ChainStartFeed() *event.Feed {
	return f.chainStartFeed
}
func (f *faultyPOWChainService) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

func (f *faultyPOWChainService) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	if f.hashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

func (f *faultyPOWChainService) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

func (f *faultyPOWChainService) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

func (f *faultyPOWChainService) DepositRoot() [32]byte {
	return [32]byte{}
}

func (f *faultyPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (f *faultyPOWChainService) ChainStartDeposits() [][]byte {
	return [][]byte{}
}

type mockPOWChainService struct {
	chainStartFeed    *event.Feed
	latestBlockNumber *big.Int
	hashesByHeight    map[int][]byte
	blockTimeByHeight map[int]uint64
}

func (m *mockPOWChainService) HasChainStartLogOccurred() (bool, uint64, error) {
	return true, uint64(time.Unix(0, 0).Unix()), nil
}
func (m *mockPOWChainService) ChainStartFeed() *event.Feed {
	return m.chainStartFeed
}
func (m *mockPOWChainService) LatestBlockHeight() *big.Int {
	return m.latestBlockNumber
}

func (m *mockPOWChainService) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

func (m *mockPOWChainService) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	// Reverse the map of heights by hash.
	heightsByHash := make(map[[32]byte]int)
	for k, v := range m.hashesByHeight {
		h := bytesutil.ToBytes32(v)
		heightsByHash[h] = k
	}
	val, ok := heightsByHash[hash]
	if !ok {
		return false, nil, fmt.Errorf("could not fetch height for hash: %#x", hash)
	}
	return true, big.NewInt(int64(val)), nil
}

func (m *mockPOWChainService) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	k := int(height.Int64())
	val, ok := m.hashesByHeight[k]
	if !ok {
		return [32]byte{}, fmt.Errorf("could not fetch hash for height: %v", height)
	}
	return bytesutil.ToBytes32(val), nil
}

func (m *mockPOWChainService) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	h := int(height.Int64())
	return m.blockTimeByHeight[h], nil
}

func (m *mockPOWChainService) DepositRoot() [32]byte {
	root := []byte("depositroot")
	return bytesutil.ToBytes32(root)
}

func (m *mockPOWChainService) ChainStartDeposits() [][]byte {
	return [][]byte{}
}

func TestWaitForChainStart_ContextClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx: ctx,
		powChainService: &faultyPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: newMockChainService(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); !strings.Contains(err.Error(), closedContext) {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
}

func TestWaitForChainStart_AlreadyStarted(t *testing.T) {
	beaconServer := &BeaconServer{
		ctx: context.Background(),
		powChainService: &mockPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: newMockChainService(),
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
}

func TestWaitForChainStart_NotStartedThenLogFired(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconServer := &BeaconServer{
		ctx:            context.Background(),
		chainStartChan: make(chan time.Time, 1),
		powChainService: &faultyPOWChainService{
			chainStartFeed: new(event.Feed),
		},
		chainService: newMockChainService(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_WaitForChainStartServer(ctrl)
	mockStream.EXPECT().Send(
		&pb.ChainStartResponse{
			Started:     true,
			GenesisTime: uint64(time.Unix(0, 0).Unix()),
		},
	).Return(nil)
	go func(tt *testing.T) {
		if err := beaconServer.WaitForChainStart(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	beaconServer.chainStartChan <- time.Unix(0, 0)
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "Sending ChainStart log and genesis time to connected validator clients")
}

func TestLatestAttestation_ContextClosed(t *testing.T) {
	hook := logTest.NewGlobal()
	mockOperationService := &mockOperationService{}
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:              ctx,
		operationService: mockOperationService,
		chainService:     newMockChainService(),
	}
	exitRoutine := make(chan bool)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	go func(tt *testing.T) {
		if err := beaconServer.LatestAttestation(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "RPC context closed, exiting goroutine")
}

func TestLatestAttestation_FaultyServer(t *testing.T) {
	mockOperationService := &mockOperationService{}
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:                 ctx,
		operationService:    mockOperationService,
		incomingAttestation: make(chan *pbp2p.Attestation, 0),
		chainService:        newMockChainService(),
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)
	attestation := &pbp2p.Attestation{}

	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	mockStream.EXPECT().Send(attestation).Return(errors.New("something wrong"))
	// Tests a faulty stream.
	go func(tt *testing.T) {
		if err := beaconServer.LatestAttestation(&ptypes.Empty{}, mockStream); err.Error() != "something wrong" {
			tt.Errorf("Faulty stream should throw correct error, wanted 'something wrong', got %v", err)
		}
		<-exitRoutine
	}(t)

	beaconServer.incomingAttestation <- attestation
	cancel()
	exitRoutine <- true
}

func TestLatestAttestation_SendsCorrectly(t *testing.T) {
	hook := logTest.NewGlobal()
	operationService := &mockOperationService{}
	ctx, cancel := context.WithCancel(context.Background())
	beaconServer := &BeaconServer{
		ctx:                 ctx,
		operationService:    operationService,
		incomingAttestation: make(chan *pbp2p.Attestation, 0),
		chainService:        newMockChainService(),
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	exitRoutine := make(chan bool)
	attestation := &pbp2p.Attestation{}
	mockStream := internal.NewMockBeaconService_LatestAttestationServer(ctrl)
	mockStream.EXPECT().Send(attestation).Return(nil)
	// Tests a good stream.
	go func(tt *testing.T) {
		if err := beaconServer.LatestAttestation(&ptypes.Empty{}, mockStream); err != nil {
			tt.Errorf("Could not call RPC method: %v", err)
		}
		<-exitRoutine
	}(t)
	beaconServer.incomingAttestation <- attestation
	cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Sending attestation to RPC clients")
}

func TestPendingDeposits_UnknownBlockNum(t *testing.T) {
	p := &mockPOWChainService{
		latestBlockNumber: nil,
	}
	bs := BeaconServer{powChainService: p}

	_, err := bs.PendingDeposits(context.Background(), nil)
	if err.Error() != "latest PoW block number is unknown" {
		t.Errorf("Received unexpected error: %v", err)
	}
}

func TestPendingDeposits_OutsideEth1FollowWindow(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}
	d := internal.SetupDB(t)

	beaconState := &pbp2p.BeaconState{
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32: []byte("0x0"),
		},
		DepositIndex: 2,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	// Using the merkleTreeIndex as the block number for this test...
	readyDeposits := []*pbp2p.Deposit{
		{
			MerkleTreeIndex: 0,
			DepositData:     []byte("a"),
		},
		{
			MerkleTreeIndex: 1,
			DepositData:     []byte("b"),
		},
	}

	recentDeposits := []*pbp2p.Deposit{
		{
			MerkleTreeIndex: 2,
			DepositData:     []byte("c"),
		},
		{
			MerkleTreeIndex: 3,
			DepositData:     []byte("d"),
		},
	}
	for _, dp := range append(readyDeposits, recentDeposits...) {
		d.InsertDeposit(ctx, dp, big.NewInt(int64(dp.MerkleTreeIndex)))
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp, big.NewInt(int64(dp.MerkleTreeIndex)))
	}

	bs := &BeaconServer{
		beaconDB:        d,
		powChainService: p,
		chainService:    newMockChainService(),
	}

	result, err := bs.PendingDeposits(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PendingDeposits) != 0 {
		t.Errorf("Received unexpected list of deposits: %+v, wanted: 0", len(result.PendingDeposits))
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
	allResp, err := bs.PendingDeposits(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(allResp.PendingDeposits) != len(recentDeposits) {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(allResp.PendingDeposits),
			len(recentDeposits),
		)
	}
}

func TestPendingDeposits_CantReturnBelowStateDepositIndex(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}
	d := internal.SetupDB(t)

	beaconState := &pbp2p.BeaconState{
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32: []byte("0x0"),
		},
		DepositIndex: 10,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	readyDeposits := []*pbp2p.Deposit{
		{
			MerkleTreeIndex: 0,
			DepositData:     []byte("a"),
		},
		{
			MerkleTreeIndex: 1,
			DepositData:     []byte("b"),
		},
	}

	var recentDeposits []*pbp2p.Deposit
	for i := 2; i < 16; i++ {
		recentDeposits = append(recentDeposits, &pbp2p.Deposit{
			MerkleTreeIndex: uint64(i),
			DepositData:     []byte{byte(i)},
		})
	}

	for _, dp := range append(readyDeposits, recentDeposits...) {
		d.InsertDeposit(ctx, dp, big.NewInt(int64(dp.MerkleTreeIndex)))
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp, big.NewInt(int64(dp.MerkleTreeIndex)))
	}

	bs := &BeaconServer{
		beaconDB:        d,
		powChainService: p,
		chainService:    newMockChainService(),
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
	allResp, err := bs.PendingDeposits(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	expectedDeposits := 6
	if len(allResp.PendingDeposits) != expectedDeposits {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(allResp.PendingDeposits),
			expectedDeposits,
		)
	}
	if allResp.PendingDeposits[0].MerkleTreeIndex != beaconState.DepositIndex {
		t.Errorf(
			"Received unexpected merkle index: %d, wanted: %d",
			allResp.PendingDeposits[0].MerkleTreeIndex,
			beaconState.DepositIndex,
		)
	}
}

func TestPendingDeposits_CantReturnMoreThanMax(t *testing.T) {
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	p := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			int(height.Int64()): []byte("0x0"),
		},
	}
	d := internal.SetupDB(t)

	beaconState := &pbp2p.BeaconState{
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32: []byte("0x0"),
		},
		DepositIndex: 2,
	}
	if err := d.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}

	readyDeposits := []*pbp2p.Deposit{
		{
			MerkleTreeIndex: 0,
			DepositData:     []byte("a"),
		},
		{
			MerkleTreeIndex: 1,
			DepositData:     []byte("b"),
		},
	}

	var recentDeposits []*pbp2p.Deposit
	for i := 2; i < 22; i++ {
		recentDeposits = append(recentDeposits, &pbp2p.Deposit{
			MerkleTreeIndex: uint64(i),
			DepositData:     []byte{byte(i)},
		})
	}

	for _, dp := range append(readyDeposits, recentDeposits...) {
		d.InsertDeposit(ctx, dp, big.NewInt(int64(dp.MerkleTreeIndex)))
	}
	for _, dp := range recentDeposits {
		d.InsertPendingDeposit(ctx, dp, big.NewInt(int64(dp.MerkleTreeIndex)))
	}

	bs := &BeaconServer{
		beaconDB:        d,
		powChainService: p,
		chainService:    newMockChainService(),
	}

	// It should also return the recent deposits after their follow window.
	p.latestBlockNumber = big.NewInt(0).Add(p.latestBlockNumber, big.NewInt(10000))
	allResp, err := bs.PendingDeposits(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(allResp.PendingDeposits) != int(params.BeaconConfig().MaxDeposits) {
		t.Errorf(
			"Received unexpected number of pending deposits: %d, wanted: %d",
			len(allResp.PendingDeposits),
			int(params.BeaconConfig().MaxDeposits),
		)
	}
}

func TestEth1Data_EmptyVotesFetchBlockHashFailure(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	beaconServer := &BeaconServer{
		beaconDB: db,
		powChainService: &faultyPOWChainService{
			hashesByHeight: make(map[int][]byte),
		},
	}
	beaconState := &pbp2p.BeaconState{
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32: []byte{'a'},
		},
		Eth1DataVotes: []*pbp2p.Eth1DataVote{},
	}
	if err := beaconServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	want := "could not fetch ETH1_FOLLOW_DISTANCE ancestor"
	if _, err := beaconServer.Eth1Data(context.Background(), nil); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected error %v, received %v", want, err)
	}
}

func TestEth1Data_EmptyVotesOk(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	height := big.NewInt(int64(params.BeaconConfig().Eth1FollowDistance))
	deps := []*pbp2p.Deposit{
		{MerkleTreeIndex: 0, DepositData: []byte("a")},
		{MerkleTreeIndex: 1, DepositData: []byte("b")},
	}
	depsData := [][]byte{}
	for _, dp := range deps {
		db.InsertDeposit(context.Background(), dp, big.NewInt(0))
		depsData = append(depsData, dp.DepositData)
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(depsData, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		t.Fatal(err)
	}
	depositRoot := depositTrie.Root()
	beaconState := &pbp2p.BeaconState{
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32:       []byte("hash0"),
			DepositRootHash32: depositRoot[:],
		},
		Eth1DataVotes: []*pbp2p.Eth1DataVote{},
	}

	powChainService := &mockPOWChainService{
		latestBlockNumber: height,
		hashesByHeight: map[int][]byte{
			0: []byte("hash0"),
			1: beaconState.LatestEth1Data.BlockHash32,
		},
	}
	beaconServer := &BeaconServer{
		beaconDB:        db,
		powChainService: powChainService,
	}

	if err := beaconServer.beaconDB.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	result, err := beaconServer.Eth1Data(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// If the data vote objects are empty, the deposit root should be the one corresponding
	// to the deposit contract in the powchain service, fetched using powChainService.DepositRoot()
	if !bytes.Equal(result.Eth1Data.DepositRootHash32, depositRoot[:]) {
		t.Errorf(
			"Expected deposit roots to match, received %#x == %#x",
			result.Eth1Data.DepositRootHash32,
			depositRoot,
		)
	}
}

func TestEth1Data_NonEmptyVotesSelectsBestVote(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()

	eth1DataVotes := []*pbp2p.Eth1DataVote{
		{
			VoteCount: 1,
			Eth1Data: &pbp2p.Eth1Data{
				BlockHash32:       []byte("block0"),
				DepositRootHash32: []byte("deposit0"),
			},
		},
		{
			VoteCount: 2,
			Eth1Data: &pbp2p.Eth1Data{
				BlockHash32:       []byte("block1"),
				DepositRootHash32: []byte("deposit1"),
			},
		},
		// We include the case in which the vote counts might match and in that
		// case we break ties by checking which block hash has the greatest
		// block height in the eth1.0 chain, accordingly.
		{
			VoteCount: 3,
			Eth1Data: &pbp2p.Eth1Data{
				BlockHash32:       []byte("block2"),
				DepositRootHash32: []byte("deposit2"),
			},
		},
		{
			VoteCount: 3,
			Eth1Data: &pbp2p.Eth1Data{
				BlockHash32:       []byte("block4"),
				DepositRootHash32: []byte("deposit3"),
			},
		},
	}
	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: eth1DataVotes,
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32: []byte("stub"),
		},
	}
	if err := db.SaveState(ctx, beaconState); err != nil {
		t.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	beaconServer := &BeaconServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight: map[int][]byte{
				0: beaconState.LatestEth1Data.BlockHash32,
				1: beaconState.Eth1DataVotes[0].Eth1Data.BlockHash32,
				2: beaconState.Eth1DataVotes[1].Eth1Data.BlockHash32,
				3: beaconState.Eth1DataVotes[3].Eth1Data.BlockHash32,
				// We will give the hash at index 2 in the beacon state's latest eth1 votes
				// priority in being selected as the best vote by giving it the highest block number.
				4: beaconState.Eth1DataVotes[2].Eth1Data.BlockHash32,
			},
		},
	}
	result, err := beaconServer.Eth1Data(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Vote at index 2 should have won the best vote selection mechanism as it had the highest block number
	// despite being tied at vote count with the vote at index 3.
	if !bytes.Equal(result.Eth1Data.BlockHash32, beaconState.Eth1DataVotes[2].Eth1Data.BlockHash32) {
		t.Errorf(
			"Expected block hashes to match, received %#x == %#x",
			result.Eth1Data.BlockHash32,
			beaconState.Eth1DataVotes[2].Eth1Data.BlockHash32,
		)
	}
	if !bytes.Equal(result.Eth1Data.DepositRootHash32, beaconState.Eth1DataVotes[2].Eth1Data.DepositRootHash32) {
		t.Errorf(
			"Expected deposit roots to match, received %#x == %#x",
			result.Eth1Data.DepositRootHash32,
			beaconState.Eth1DataVotes[2].Eth1Data.DepositRootHash32,
		)
	}
}

func TestBlockTree_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	justifiedState := &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorBalances: make([]uint64, 11),
	}
	for i := 0; i < len(justifiedState.ValidatorBalances); i++ {
		justifiedState.ValidatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := hashutil.HashBeaconBlock(justifiedBlock)
	validators := []*pbp2p.Validator{{ExitEpoch: params.BeaconConfig().FarFutureEpoch}}
	balances := []uint64{params.BeaconConfig().MaxDepositAmount}
	b1 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("A"),
	}
	b1Root, _ := hashutil.HashBeaconBlock(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("C"),
	}
	b2Root, _ := hashutil.HashBeaconBlock(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("D"),
	}
	b3Root, _ := hashutil.HashBeaconBlock(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 4,
		ParentRootHash32: b1Root[:],
		RandaoReveal:     []byte("B"),
	}
	b4Root, _ := hashutil.HashBeaconBlock(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 4,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 5,
		ParentRootHash32: b3Root[:],
		RandaoReveal:     []byte("E"),
	}
	b5Root, _ := hashutil.HashBeaconBlock(b5)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 5,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b5Root); err != nil {
		t.Fatal(err)
	}
	attestationTargets := make(map[uint64]*pbp2p.AttestationTarget)
	// We give block A 3 votes.
	attestationTargets[0] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRootHash32,
		BlockRoot:  b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRootHash32,
		BlockRoot:  b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRootHash32,
		BlockRoot:  b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRootHash32,
		BlockRoot:  b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:       b5.Slot,
		ParentRoot: b5.ParentRootHash32,
		BlockRoot:  b5Root[:],
	}

	tree := []*pb.BlockTreeResponse_TreeNode{
		{
			Block:             b1,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b2,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b3,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b4,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b5,
			ParticipatedVotes: 1 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
	}
	for _, node := range tree {
		if err := db.SaveBlock(node.Block); err != nil {
			t.Fatal(err)
		}
	}

	headState := &pbp2p.BeaconState{
		Slot: b4.Slot,
	}
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	resp, err := bs.BlockTree(ctx, &ptypes.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(resp.Tree, func(i, j int) bool {
		return string(resp.Tree[i].Block.RandaoReveal) < string(resp.Tree[j].Block.RandaoReveal)
	})
	sort.Slice(tree, func(i, j int) bool {
		return string(tree[i].Block.RandaoReveal) < string(tree[j].Block.RandaoReveal)
	})
	for i := range resp.Tree {
		if !proto.Equal(resp.Tree[i].Block, tree[i].Block) {
			t.Errorf("Expected %v, received %v", tree[i].Block, resp.Tree[i].Block)
		}
	}
}
func TestBlockTreeBySlots_ArgsValildation(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	justifiedState := &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorBalances: make([]uint64, 11),
	}
	for i := 0; i < len(justifiedState.ValidatorBalances); i++ {
		justifiedState.ValidatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := hashutil.HashBeaconBlock(justifiedBlock)
	validators := []*pbp2p.Validator{{ExitEpoch: params.BeaconConfig().FarFutureEpoch}}
	balances := []uint64{params.BeaconConfig().MaxDepositAmount}
	b1 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("A"),
	}
	b1Root, _ := hashutil.HashBeaconBlock(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("C"),
	}
	b2Root, _ := hashutil.HashBeaconBlock(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("D"),
	}
	b3Root, _ := hashutil.HashBeaconBlock(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 4,
		ParentRootHash32: b1Root[:],
		RandaoReveal:     []byte("B"),
	}
	b4Root, _ := hashutil.HashBeaconBlock(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 4,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 5,
		ParentRootHash32: b3Root[:],
		RandaoReveal:     []byte("E"),
	}
	b5Root, _ := hashutil.HashBeaconBlock(b5)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 5,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b5Root); err != nil {
		t.Fatal(err)
	}
	attestationTargets := make(map[uint64]*pbp2p.AttestationTarget)
	// We give block A 3 votes.
	attestationTargets[0] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRootHash32,
		BlockRoot:  b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRootHash32,
		BlockRoot:  b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRootHash32,
		BlockRoot:  b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRootHash32,
		BlockRoot:  b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:       b5.Slot,
		ParentRoot: b5.ParentRootHash32,
		BlockRoot:  b5Root[:],
	}

	tree := []*pb.BlockTreeResponse_TreeNode{
		{
			Block:             b1,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b2,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b3,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b4,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b5,
			ParticipatedVotes: 1 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
	}
	for _, node := range tree {
		if err := db.SaveBlock(node.Block); err != nil {
			t.Fatal(err)
		}
	}
	headState := &pbp2p.BeaconState{
		Slot: b4.Slot,
	}
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}
	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	if _, err := bs.BlockTreeBySlots(ctx, nil); err == nil {
		// There should be a "argument 'TreeBlockSlotRequest' cannot be nil" error
		t.Fatal(err)
	}
	slotRange := &pb.TreeBlockSlotRequest{
		SlotFrom: params.BeaconConfig().GenesisSlot + 4,
		SlotTo:   params.BeaconConfig().GenesisSlot + 3,
	}
	if _, err := bs.BlockTreeBySlots(ctx, slotRange); err == nil {
		// There should be a 'Upper limit of slot range cannot be lower than the lower limit' error.
		t.Fatal(err)
	}
}
func TestBlockTreeBySlots_OK(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	ctx := context.Background()
	// We want to ensure that if our block tree looks as follows, the RPC response
	// returns the correct information.
	//                   /->[A, Slot 3, 3 Votes]->[B, Slot 4, 3 Votes]
	// [Justified Block]->[C, Slot 3, 2 Votes]
	//                   \->[D, Slot 3, 2 Votes]->[SKIP SLOT]->[E, Slot 5, 1 Vote]
	justifiedState := &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot,
		ValidatorBalances: make([]uint64, 11),
	}
	for i := 0; i < len(justifiedState.ValidatorBalances); i++ {
		justifiedState.ValidatorBalances[i] = params.BeaconConfig().MaxDepositAmount
	}
	if err := db.SaveJustifiedState(justifiedState); err != nil {
		t.Fatal(err)
	}
	justifiedBlock := &pbp2p.BeaconBlock{
		Slot: params.BeaconConfig().GenesisSlot,
	}
	if err := db.SaveJustifiedBlock(justifiedBlock); err != nil {
		t.Fatal(err)
	}
	justifiedRoot, _ := hashutil.HashBeaconBlock(justifiedBlock)
	validators := []*pbp2p.Validator{{ExitEpoch: params.BeaconConfig().FarFutureEpoch}}
	balances := []uint64{params.BeaconConfig().MaxDepositAmount}
	b1 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("A"),
	}
	b1Root, _ := hashutil.HashBeaconBlock(b1)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b1Root); err != nil {
		t.Fatal(err)
	}
	b2 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("C"),
	}
	b2Root, _ := hashutil.HashBeaconBlock(b2)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b2Root); err != nil {
		t.Fatal(err)
	}
	b3 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 3,
		ParentRootHash32: justifiedRoot[:],
		RandaoReveal:     []byte("D"),
	}
	b3Root, _ := hashutil.HashBeaconBlock(b3)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 3,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b3Root); err != nil {
		t.Fatal(err)
	}
	b4 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 4,
		ParentRootHash32: b1Root[:],
		RandaoReveal:     []byte("B"),
	}
	b4Root, _ := hashutil.HashBeaconBlock(b4)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 4,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b4Root); err != nil {
		t.Fatal(err)
	}
	b5 := &pbp2p.BeaconBlock{
		Slot:             params.BeaconConfig().GenesisSlot + 5,
		ParentRootHash32: b3Root[:],
		RandaoReveal:     []byte("E"),
	}
	b5Root, _ := hashutil.HashBeaconBlock(b5)
	if err := db.SaveHistoricalState(ctx, &pbp2p.BeaconState{
		Slot:              params.BeaconConfig().GenesisSlot + 5,
		ValidatorRegistry: validators,
		ValidatorBalances: balances,
	}, b5Root); err != nil {
		t.Fatal(err)
	}
	attestationTargets := make(map[uint64]*pbp2p.AttestationTarget)
	// We give block A 3 votes.
	attestationTargets[0] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[1] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}
	attestationTargets[2] = &pbp2p.AttestationTarget{
		Slot:       b1.Slot,
		ParentRoot: b1.ParentRootHash32,
		BlockRoot:  b1Root[:],
	}

	// We give block C 2 votes.
	attestationTargets[3] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRootHash32,
		BlockRoot:  b2Root[:],
	}
	attestationTargets[4] = &pbp2p.AttestationTarget{
		Slot:       b2.Slot,
		ParentRoot: b2.ParentRootHash32,
		BlockRoot:  b2Root[:],
	}

	// We give block D 2 votes.
	attestationTargets[5] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRootHash32,
		BlockRoot:  b3Root[:],
	}
	attestationTargets[6] = &pbp2p.AttestationTarget{
		Slot:       b3.Slot,
		ParentRoot: b3.ParentRootHash32,
		BlockRoot:  b3Root[:],
	}

	// We give block B 3 votes.
	attestationTargets[7] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[8] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}
	attestationTargets[9] = &pbp2p.AttestationTarget{
		Slot:       b4.Slot,
		ParentRoot: b4.ParentRootHash32,
		BlockRoot:  b4Root[:],
	}

	// We give block E 1 vote.
	attestationTargets[10] = &pbp2p.AttestationTarget{
		Slot:       b5.Slot,
		ParentRoot: b5.ParentRootHash32,
		BlockRoot:  b5Root[:],
	}

	tree := []*pb.BlockTreeResponse_TreeNode{
		{
			Block:             b1,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b2,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b3,
			ParticipatedVotes: 2 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b4,
			ParticipatedVotes: 3 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
		{
			Block:             b5,
			ParticipatedVotes: 1 * params.BeaconConfig().MaxDepositAmount,
			TotalVotes:        params.BeaconConfig().MaxDepositAmount,
		},
	}
	for _, node := range tree {
		if err := db.SaveBlock(node.Block); err != nil {
			t.Fatal(err)
		}
	}

	headState := &pbp2p.BeaconState{
		Slot: b4.Slot,
	}
	if err := db.UpdateChainHead(ctx, b4, headState); err != nil {
		t.Fatal(err)
	}

	bs := &BeaconServer{
		beaconDB:       db,
		targetsFetcher: &mockChainService{targets: attestationTargets},
	}
	slotRange := &pb.TreeBlockSlotRequest{
		SlotFrom: params.BeaconConfig().GenesisSlot + 3,
		SlotTo:   params.BeaconConfig().GenesisSlot + 4,
	}
	resp, err := bs.BlockTreeBySlots(ctx, slotRange)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Tree) != 2 {
		t.Logf("Incorrect number of nodes in tree, expected: %d, actual: %d", 2, len(resp.Tree))
	}
}

func Benchmark_Eth1Data(b *testing.B) {
	db := internal.SetupDB(b)
	defer internal.TeardownDB(b, db)
	ctx := context.Background()

	hashesByHeight := make(map[int][]byte)

	beaconState := &pbp2p.BeaconState{
		Eth1DataVotes: []*pbp2p.Eth1DataVote{},
		LatestEth1Data: &pbp2p.Eth1Data{
			BlockHash32: []byte("stub"),
		},
	}
	numOfVotes := 1000
	for i := 0; i < numOfVotes; i++ {
		blockhash := []byte{'b', 'l', 'o', 'c', 'k', byte(i)}
		deposit := []byte{'d', 'e', 'p', 'o', 's', 'i', 't', byte(i)}
		beaconState.Eth1DataVotes = append(beaconState.Eth1DataVotes,
			&pbp2p.Eth1DataVote{
				VoteCount: uint64(i),
				Eth1Data: &pbp2p.Eth1Data{
					BlockHash32:       blockhash,
					DepositRootHash32: deposit,
				},
			})
		hashesByHeight[i] = blockhash
	}
	hashesByHeight[numOfVotes+1] = []byte("stub")

	if err := db.SaveState(ctx, beaconState); err != nil {
		b.Fatal(err)
	}
	currentHeight := params.BeaconConfig().Eth1FollowDistance + 5
	beaconServer := &BeaconServer{
		beaconDB: db,
		powChainService: &mockPOWChainService{
			latestBlockNumber: big.NewInt(int64(currentHeight)),
			hashesByHeight:    hashesByHeight,
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := beaconServer.Eth1Data(context.Background(), nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
