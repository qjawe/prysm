// Package backend contains utilities for simulating an entire
// ETH 2.0 beacon chain for e2e tests and benchmarking
// purposes.
package backend

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	b "github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	log "github.com/sirupsen/logrus"
)

// SimulatedBackend allowing for a programmatic advancement
// of an in-memory beacon chain for client test runs
// and other e2e use cases.
type SimulatedBackend struct {
	chainService       *blockchain.ChainService
	beaconDB           *db.BeaconDB
	state              *pb.BeaconState
	prevBlockRoots     [][32]byte
	inMemoryBlocks     []*pb.BeaconBlock
	historicalDeposits []*pb.Deposit
}

// SimulatedObjects is a container to hold the
// required primitives for generation of a beacon
// block.
type SimulatedObjects struct {
	simDeposit          *StateTestDeposit
	simProposerSlashing *StateTestProposerSlashing
	simAttesterSlashing *StateTestAttesterSlashing
	simValidatorExit    *StateTestValidatorExit
}

// NewSimulatedBackend creates an instance by initializing a chain service
// utilizing a mockDB which will act according to test run parameters specified
// in the common ETH 2.0 client test YAML format.
func NewSimulatedBackend() (*SimulatedBackend, error) {
	db, err := db.SetupDB()
	if err != nil {
		return nil, fmt.Errorf("could not setup simulated backend db: %v", err)
	}
	cs, err := blockchain.NewChainService(context.Background(), &blockchain.Config{
		BeaconDB: db,
	})
	if err != nil {
		return nil, err
	}
	return &SimulatedBackend{
		chainService:       cs,
		beaconDB:           db,
		inMemoryBlocks:     make([]*pb.BeaconBlock, 0),
		historicalDeposits: make([]*pb.Deposit, 0),
	}, nil
}

// SetupBackend sets up the simulated backend with simulated deposits, and initializes the
// state and genesis block.
func (sb *SimulatedBackend) SetupBackend(numOfDeposits uint64) ([]*bls.SecretKey, error) {
	initialDeposits, privKeys, err := generateInitialSimulatedDeposits(numOfDeposits)
	if err != nil {
		return nil, fmt.Errorf("could not simulate initial validator deposits: %v", err)
	}
	if err := sb.setupBeaconStateAndGenesisBlock(initialDeposits); err != nil {
		return nil, fmt.Errorf("could not set up beacon state and initialize genesis block %v", err)
	}
	return privKeys, nil
}

// DB returns the underlying db instance in the simulated
// backend.
func (sb *SimulatedBackend) DB() *db.BeaconDB {
	return sb.beaconDB
}

// GenerateBlockAndAdvanceChain generates a simulated block and runs that block though
// state transition.
func (sb *SimulatedBackend) GenerateBlockAndAdvanceChain(objects *SimulatedObjects, privKeys []*bls.SecretKey) error {
	prevBlockRoot := sb.prevBlockRoots[len(sb.prevBlockRoots)-1]
	// We generate a new block to pass into the state transition.
	newBlock, newBlockRoot, err := generateSimulatedBlock(
		sb.state,
		prevBlockRoot,
		sb.historicalDeposits,
		objects,
		privKeys,
	)
	if err != nil {
		return fmt.Errorf("could not generate simulated beacon block %v", err)
	}
	newState := sb.state
	newState.LatestEth1Data = newBlock.Body.Eth1Data
	newState, err = state.ExecuteStateTransition(
		context.Background(),
		sb.state,
		newBlock,
		state.DefaultConfig(),
	)
	if err != nil {
		return fmt.Errorf("could not execute state transition: %v", err)
	}

	sb.state = newState
	sb.prevBlockRoots = append(sb.prevBlockRoots, newBlockRoot)
	sb.inMemoryBlocks = append(sb.inMemoryBlocks, newBlock)
	if len(newBlock.Body.Deposits) > 0 {
		sb.historicalDeposits = append(sb.historicalDeposits, newBlock.Body.Deposits...)
	}

	return nil
}

// Shutdown closes the db associated with the simulated backend.
func (sb *SimulatedBackend) Shutdown() error {
	return sb.beaconDB.Close()
}

// State is a getter to return the current beacon state
// of the backend.
func (sb *SimulatedBackend) State() *pb.BeaconState {
	return sb.state
}

// InMemoryBlocks returns the blocks that have been processed by the simulated
// backend.
func (sb *SimulatedBackend) InMemoryBlocks() []*pb.BeaconBlock {
	return sb.inMemoryBlocks
}

// RunForkChoiceTest uses a parsed set of chaintests from a YAML file
// according to the ETH 2.0 client chain test specification and runs them
// against the simulated backend.
func (sb *SimulatedBackend) RunForkChoiceTest(testCase *ForkChoiceTestCase) error {
	defer db.TeardownDB(sb.beaconDB)
	// Utilize the config parameters in the test case to setup
	// the DB and set global config parameters accordingly.
	// Config parameters include: ValidatorCount, ShardCount,
	// CycleLength, MinCommitteeSize, and more based on the YAML
	// test language specification.
	c := params.BeaconConfig()
	c.ShardCount = testCase.Config.ShardCount
	c.SlotsPerEpoch = testCase.Config.CycleLength
	c.TargetCommitteeSize = testCase.Config.MinCommitteeSize
	params.OverrideBeaconConfig(c)

	// Then, we create the validators based on the custom test config.
	validators := make([]*pb.Validator, testCase.Config.ValidatorCount)
	for i := uint64(0); i < testCase.Config.ValidatorCount; i++ {
		validators[i] = &pb.Validator{
			ExitEpoch: params.BeaconConfig().ActivationExitDelay,
			Pubkey:    []byte{},
		}
	}
	// TODO(#718): Next step is to update and save the blocks specified
	// in the case case into the DB.
	//
	// Then, we call the updateHead routine and confirm the
	// chain's head is the expected result from the test case.
	return nil
}

// RunShuffleTest uses validator set specified from a YAML file, runs the validator shuffle
// algorithm, then compare the output with the expected output from the YAML file.
func (sb *SimulatedBackend) RunShuffleTest(testCase *ShuffleTestCase) error {
	defer db.TeardownDB(sb.beaconDB)
	seed := common.HexToHash(testCase.Seed)
	testIndices := make([]uint64, testCase.Count, testCase.Count)
	for i := uint64(0); i < testCase.Count; i++ {
		testIndices[i] = i
	}
	shuffledList := make([]uint64, testCase.Count)
	for i := uint64(0); i < testCase.Count; i++ {
		si, err := utils.ShuffledIndex(i, testCase.Count, seed)
		if err != nil {
			log.Error(err)
			return err
		}
		shuffledList[i] = si
	}
	if !reflect.DeepEqual(shuffledList, testCase.Shuffled) {
		return fmt.Errorf("shuffle result error: expected %v, actual %v", testCase.Shuffled, shuffledList)
	}
	return nil
}

// RunStateTransitionTest advances a beacon chain state transition an N amount of
// slots from a genesis state, with a block being processed at every iteration
// of the state transition function.
func (sb *SimulatedBackend) RunStateTransitionTest(testCase *StateTestCase) error {
	defer db.TeardownDB(sb.beaconDB)
	setTestConfig(testCase)

	privKeys, err := sb.initializeStateTest(testCase)
	if err != nil {
		return fmt.Errorf("could not initialize state test %v", err)
	}
	averageTimesPerTransition := []time.Duration{}
	startSlot := uint64(0)
	for i := startSlot; i < startSlot+testCase.Config.NumSlots; i++ {
		// If the slot is marked as skipped in the configuration options,
		// we simply run the state transition with a nil block argument.
		if sliceutil.IsInUint64(i, testCase.Config.SkipSlots) {
			continue
		}

		simulatedObjects := sb.generateSimulatedObjects(testCase, i)
		startTime := time.Now()

		if err := sb.GenerateBlockAndAdvanceChain(simulatedObjects, privKeys); err != nil {
			return fmt.Errorf("could not generate the block and advance the chain %v", err)
		}

		endTime := time.Now()
		averageTimesPerTransition = append(averageTimesPerTransition, endTime.Sub(startTime))
	}

	log.Infof(
		"with %d initial deposits, each state transition took average time = %v",
		testCase.Config.DepositsForChainStart,
		averageDuration(averageTimesPerTransition),
	)

	if err := sb.compareTestCase(testCase); err != nil {
		return err
	}

	return nil
}

// initializeStateTest sets up the environment by generating all the required objects in order
// to proceed with the state test.
func (sb *SimulatedBackend) initializeStateTest(testCase *StateTestCase) ([]*bls.SecretKey, error) {
	initialDeposits, privKeys, err := generateInitialSimulatedDeposits(testCase.Config.DepositsForChainStart)
	if err != nil {
		return nil, fmt.Errorf("could not simulate initial validator deposits: %v", err)
	}
	if err := sb.setupBeaconStateAndGenesisBlock(initialDeposits); err != nil {
		return nil, fmt.Errorf("could not set up beacon state and initialize genesis block %v", err)
	}
	return privKeys, nil
}

// setupBeaconStateAndGenesisBlock creates the initial beacon state and genesis block in order to
// proceed with the test.
func (sb *SimulatedBackend) setupBeaconStateAndGenesisBlock(initialDeposits []*pb.Deposit) error {
	var err error
	genesisTime := time.Date(2018, 9, 0, 0, 0, 0, 0, time.UTC).Unix()
	sb.state, err = state.GenesisBeaconState(initialDeposits, uint64(genesisTime), nil)
	if err != nil {
		return fmt.Errorf("could not initialize simulated beacon state: %v", err)
	}
	sb.state.LatestStateRoots = make([][]byte, params.BeaconConfig().SlotsPerHistoricalRoot)
	sb.historicalDeposits = initialDeposits

	// We do not expect hashing initial beacon state and genesis block to
	// fail, so we can safely ignore the error below.
	// #nosec G104
	stateRoot, err := hashutil.HashProto(sb.state)
	if err != nil {
		return fmt.Errorf("could not tree hash state: %v", err)
	}
	genesisBlock := b.NewGenesisBlock(stateRoot[:])
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock.Body)
	if err != nil {
		return fmt.Errorf("could not tree hash genesis block: %v", err)
	}
	sb.state.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	genesisRoot, err := ssz.SigningRoot(sb.state.LatestBlockHeader)
	if err != nil {
		return err
	}
	// We now keep track of generated blocks for each state transition in
	// a slice.
	sb.prevBlockRoots = [][32]byte{genesisRoot}
	sb.inMemoryBlocks = append(sb.inMemoryBlocks, genesisBlock)
	return nil
}

// generateSimulatedObjects generates the simulated objects depending on the testcase and current slot.
func (sb *SimulatedBackend) generateSimulatedObjects(testCase *StateTestCase, slotNumber uint64) *SimulatedObjects {
	// If the slot is not skipped, we check if we are simulating a deposit at the current slot.
	var simulatedDeposit *StateTestDeposit
	for _, deposit := range testCase.Config.Deposits {
		if deposit.Slot == slotNumber {
			simulatedDeposit = deposit
			break
		}
	}
	var simulatedProposerSlashing *StateTestProposerSlashing
	for _, pSlashing := range testCase.Config.ProposerSlashings {
		if pSlashing.Slot == slotNumber {
			simulatedProposerSlashing = pSlashing
			break
		}
	}
	var simulatedAttesterSlashing *StateTestAttesterSlashing
	for _, cSlashing := range testCase.Config.AttesterSlashings {
		if cSlashing.Slot == slotNumber {
			simulatedAttesterSlashing = cSlashing
			break
		}
	}
	var simulatedValidatorExit *StateTestValidatorExit
	for _, exit := range testCase.Config.ValidatorExits {
		if exit.Epoch == slotNumber/params.BeaconConfig().SlotsPerEpoch {
			simulatedValidatorExit = exit
			break
		}
	}

	return &SimulatedObjects{
		simDeposit:          simulatedDeposit,
		simProposerSlashing: simulatedProposerSlashing,
		simAttesterSlashing: simulatedAttesterSlashing,
		simValidatorExit:    simulatedValidatorExit,
	}
}

// compareTestCase compares the state in the simulated backend against the values in inputted test case. If
// there are any discrepancies it returns an error.
func (sb *SimulatedBackend) compareTestCase(testCase *StateTestCase) error {
	if sb.state.Slot != testCase.Results.Slot {
		return fmt.Errorf(
			"incorrect state slot after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			sb.state.Slot,
			testCase.Results.Slot,
		)
	}
	if len(sb.state.ValidatorRegistry) != testCase.Results.NumValidators {
		return fmt.Errorf(
			"incorrect num validators after %d state transitions without blocks, wanted %d, received %d",
			testCase.Config.NumSlots,
			testCase.Results.NumValidators,
			len(sb.state.ValidatorRegistry),
		)
	}
	for _, slashed := range testCase.Results.SlashedValidators {
		if !sb.state.ValidatorRegistry[slashed].Slashed {
			return fmt.Errorf(
				"expected validator at index %d to have been slashed",
				slashed,
			)
		}
		if sb.state.ValidatorRegistry[slashed].ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf(
				"expected validator at index %d to have exited",
				slashed,
			)
		}
		if sb.state.ValidatorRegistry[slashed].WithdrawableEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf(
				"expected validator at index %d withdrawable epoch to have been changed, received: %d",
				slashed,
				sb.state.ValidatorRegistry[slashed].WithdrawableEpoch,
			)
		}
	}
	for _, exited := range testCase.Results.ExitedValidators {
		if sb.state.ValidatorRegistry[exited].ExitEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf(
				"expected validator at index %d to have exited",
				exited,
			)
		}
		if sb.state.ValidatorRegistry[exited].WithdrawableEpoch != params.BeaconConfig().FarFutureEpoch {
			return fmt.Errorf(
				"expected validator at index %d withdrawable epoch to have been changed, received: %d",
				exited,
				sb.state.ValidatorRegistry[exited].WithdrawableEpoch,
			)
		}
	}
	return nil
}

func setTestConfig(testCase *StateTestCase) {
	// We setup the initial configuration for running state
	// transition tests below.
	c := params.BeaconConfig()
	c.SlotsPerEpoch = testCase.Config.SlotsPerEpoch
	c.DepositsForChainStart = testCase.Config.DepositsForChainStart
	params.OverrideBeaconConfig(c)
}

func averageDuration(times []time.Duration) time.Duration {
	sum := int64(0)
	for _, t := range times {
		sum += t.Nanoseconds()
	}
	return time.Duration(sum / int64(len(times)))
}
