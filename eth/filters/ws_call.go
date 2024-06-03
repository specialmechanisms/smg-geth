package filters

import (
	"context"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)


// TODO nick - we should want to move this to a separate file
// we need a notice for the NewHeads method or we let it here?
type PoolBalanceMetaData struct {
	ExchangeName string
	Address      common.Address
	Topic        common.Hash
	// TODO nick-smc not sure about the type here yet
	BalanceMetaData interface{}
}

type NewHeadsWithPoolBalanceMetaData struct {
	Header              *types.Header
	PoolBalanceMetaData map[common.Address]PoolBalanceMetaData
}

// TODO nick maybe we want to move this to the top of the file
var exchangeName_UniswapV2 string
var exchangeName_UniswapV3 string
var exchangeName_BalancerV2 string
var exchangeName_OneInchV2 string
var mapOfExchangeNameToTopics = make(map[string][]common.Hash)

// TODO nick give this a better name
var flattenedValues []common.Hash
var numWorkers int
var allCurvePools []string
var err error

func init() {
	fmt.Println("SMG GETH v0.1")
	fmt.Println("NewHeads: init() called...")
	numWorkers = runtime.NumCPU() - 1
	if numWorkers < 1 {
		numWorkers = 1 // Ensure at least one worker
	}
	// create a map of ExchangeName -> Topics
	//  those exchange names are copied from Ninja's codebase
	exchangeName_UniswapV2 = "UniswapV2"
	exchangeName_UniswapV3 = "UniswapV3"
	exchangeName_BalancerV2 = "BalancerV2"
	// var exchangeName_Curve string = "Curve"
	exchangeName_OneInchV2 = "OneInchV2"

	mapOfExchangeNameToTopics[exchangeName_UniswapV3] = []common.Hash{
		common.HexToHash("0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67"), // univ3 swap
		common.HexToHash("0x7a53080ba414158be7ec69b987b5fb7d07dee101fe85488f0853ae16239d0bde"), // univ3 mint
		common.HexToHash("0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c"), // univ3 burn
	}
	mapOfExchangeNameToTopics[exchangeName_UniswapV2] = []common.Hash{
		common.HexToHash("0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1"), // uniswapV2 sync
	}
	//  TODO nick-smc check if we need both topics here
	mapOfExchangeNameToTopics[exchangeName_BalancerV2] = []common.Hash{
		common.HexToHash("0x2170c741c41531aec20e7c107c24eecfdd15e69c9bb0a8dd37b1840b9e0b207b"), // balancerV2 swap
		common.HexToHash("0xe5ce249087ce04f05a957192435400fd97868dba0e6a4b4c049abf8af80dae78"), // balancerV2 poolBalancesChanged
	}
	var exchangeName_OneInchV2 string = "OneInchV2"
	mapOfExchangeNameToTopics[exchangeName_OneInchV2] = []common.Hash{
		common.HexToHash("0x8bab6aed5a508937051a144e61d6e61336834a66aaee250a00613ae6f744c422"), // OneInchV2 Deposited
		common.HexToHash("0x3cae9923fd3c2f468aa25a8ef687923e37f957459557c0380fd06526c0b8cdbc"), // OneInchV2 Withdrawn
		common.HexToHash("0xbd99c6719f088aa0abd9e7b7a4a635d1f931601e9f304b538dc42be25d8c65c6"), // OneInchV2 Swapped
	}
	// Flatten the map values
	for _, values := range mapOfExchangeNameToTopics {
		flattenedValues = append(flattenedValues, values...)
	}
	allCurvePools, err = GetAllPools_Curve()
	if err != nil {
		log.Error("nickdebug NewHeads: error getting allCurvePools: ", err)
	} else {
		fmt.Println("nickdebug NewHeads: allCurvePools", allCurvePools)
	}

}

type MyWaitGroup struct {
	wg      sync.WaitGroup
	counter int
	mu      sync.Mutex
}

func (mwg *MyWaitGroup) Add(delta int) {
	mwg.mu.Lock()
	defer mwg.mu.Unlock()
	mwg.counter += delta
	mwg.wg.Add(delta)
}

func (mwg *MyWaitGroup) Done() {
	mwg.mu.Lock()
	defer mwg.mu.Unlock()
	mwg.counter--
	mwg.wg.Done()
}

func (mwg *MyWaitGroup) Wait() {
	mwg.wg.Wait()
}

func (mwg *MyWaitGroup) Count() int {
	mwg.mu.Lock()
	defer mwg.mu.Unlock()
	return mwg.counter
}

// curveWorker processes Curve pools to fetch balance metadata.
// func curveWorker(id int, pools <-chan string, results chan<- PoolBalanceMetaData) {
func curveWorker(id int, pools <-chan string, results chan<- PoolBalanceMetaData, curveWg *MyWaitGroup) {
	for pool := range pools {
		// log.Info("curveWorker count (start)", "count", curveWg.Count(), "pool", pool)
		// Fetch balance metadata for the Curve pool
		balanceMetaData, err := GetBalanceMetaData_Curve(pool)
		if err != nil {
			// Handle error, perhaps log it
			fmt.Printf("Worker %d: Error fetching balance metadata for Curve pool %s: %v\n", id, pool, err)
		}

		// Create PoolBalanceMetaData object
		poolAddress := common.HexToAddress(pool)
		poolBalanceMetaData := PoolBalanceMetaData{
			Address:         poolAddress,
			Topic:           common.Hash{}, // No topic for Curve in this example
			BalanceMetaData: balanceMetaData,
			ExchangeName:    "Curve",
		}

		// Send the result back
		results <- poolBalanceMetaData
		curveWg.Done()

		// log.Info("worker count (end)", "count", curveWg.Count(), "pool", pool)
	}
}

// Assuming that GetLogs returns []*types.Log
type Log = types.Log // Reusing the type from GetLogs
func logWorker(id int, logs <-chan *Log, results chan<- PoolBalanceMetaData, logWg *MyWaitGroup, currentBlockNumber *big.Int) {
	for eventLog := range logs {
		// log.Info("logWorker count (start)", "count", logWg.Count())

		address := eventLog.Address
		eventLogTopic := eventLog.Topics[0]
		balanceMetaData := interface{}(nil)
		var topicExchangeName string
		for exchangeName, topics := range mapOfExchangeNameToTopics {
			for _, topic := range topics {
				if topic == eventLogTopic {
					topicExchangeName = exchangeName
					break
				}
			}
		}

		var err error
		switch topicExchangeName {
		case exchangeName_UniswapV2:
			// log.Info("found UniswapV2 log...", "pool", address.Hex())
			balanceMetaData, err = GetBalanceMetaData_UniswapV2(address.Hex())
			// log.Info("finished UniswapV2 log", "pool", address.Hex())
		case exchangeName_UniswapV3:
			// log.Info("found UniswapV3 log...", "pool", address.Hex())
			balanceMetaData, err = GetBalanceMetaData_UniswapV3(address.Hex())
			// log.Info("finished UniswapV3 log", "pool", address.Hex())
		case exchangeName_BalancerV2:
			poolId := eventLog.Topics[1]
			// log.Info("found BalancerV2 log...", "poolId", poolId.Hex())
			balanceMetaData, address, err = GetBalanceMetaData_BalancerV2(poolId)
			// log.Info("finished BalancerV2 log", "poolId", poolId.Hex())
		case exchangeName_OneInchV2:
			// log.Info("found OneInchV2 log...", "address", address.Hex())
			balanceMetaData, err = GetBalanceMetaData_OneInchV2(address.Hex())
			AddPoolToActiveOneInchV2DecayPeriods(address, currentBlockNumber)
			// log.Info("finished OneInchV2 log", "address", address.Hex())
		default:
			log.Error("NewHeads: unknown exchangeName", "topicExchangeName", topicExchangeName)
		}
		if err != nil {
			switch e := err.(type) {
			case WrongFactoryAddressError:
				log.Info("NewHeads: pool has wrong factory address", "topicExchangeName", topicExchangeName, "address:", e.Address)
			default:
				log.Error("NewHeads: error getting balanceMetaData: ", "topicExchangeName", topicExchangeName, "address:", address.Hex(), "error", err)
			}
			balanceMetaData = interface{}(nil)
		}

		poolBalanceMetaData := PoolBalanceMetaData{
			Address:         address,
			Topic:           eventLogTopic,
			BalanceMetaData: balanceMetaData,
			ExchangeName:    topicExchangeName,
		}
		results <- poolBalanceMetaData
		logWg.Done()

		// log.Info("worker count (end)", "count", logWg.Count())
	}
}

func oneInchWorker(id int, pools <-chan common.Address, results chan<- PoolBalanceMetaData, oneInchWg *MyWaitGroup) {
	for poolAddress := range pools {
		// log.Info("oneInchWorker count (start)", "count", oneInchWg.Count())
		balanceMetaData := interface{}(nil)
		// Process the pool
		// Example: Fetching pool data (placeholder logic)
		balanceMetaData, err = GetBalanceMetaData_OneInchV2(poolAddress.Hex())
		if err != nil {
			log.Error("NewHeads: error getting balanceMetaData on OneinchV2: ", "address:", poolAddress.Hex(), "error", err)
		}

		// Send the result back
		poolBalanceMetaData := PoolBalanceMetaData{
			Address:         poolAddress,
			Topic:           common.HexToHash("0xbd99c6719f088aa0abd9e7b7a4a635d1f931601e9f304b538dc42be25d8c65c6"),
			BalanceMetaData: balanceMetaData,
			ExchangeName:    exchangeName_OneInchV2,
		}
		results <- poolBalanceMetaData
		oneInchWg.Done()
		// log.Info("oneInchWorker count (end)", "count", oneInchWg.Count())
	}
}

// NewHeads send a notification each time a new (header) block is appended to the chain.
func (api *FilterAPI) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		// var logWg, curveWg sync.WaitGroup
		var logWg, curveWg MyWaitGroup
		headers := make(chan *types.Header)
		headersSub := api.events.SubscribeNewHeads(headers)
		defer headersSub.Unsubscribe()

		for {
			select {
			case h := <-headers:
				start := time.Now()
				// measure the time since h.Time
				hTime := time.Unix(int64(h.Time), 0)
				log.Info("NewHeads: block discovered", "block number", h.Number, "duration", time.Since(hTime))

				// log.Info("NewHeads: new block found", "block number", h.Number)
				blockHash := h.Hash()

				filterCriteria := FilterCriteria{
					BlockHash: &blockHash,
					FromBlock: nil,
					ToBlock:   nil,
					Addresses: nil,
					Topics:    [][]common.Hash{flattenedValues},
				}

				logs, err := api.GetLogs(ctx, filterCriteria)
				// print the len of logs TODO nick remove this again
				// log.Info("NewHeads: len(logs)", "count", len(logs))
				if err != nil {
					log.Error("NewHeads: error getting logs: ", err)
					continue
				}

				// Create channels for logs and results
				logChan := make(chan *types.Log, len(logs))          // Channel to send logs to logWorkers
				results := make(chan PoolBalanceMetaData, len(logs)) // Channel to collect results
				// log.Info("NewHeads: logChan and results built")
				// Start logWorkers
				logWg.Add(len(logs))
				for w := 1; w <= numWorkers; w++ {
					go func(id int) {
						logWorker(id, logChan, results, &logWg, h.Number)
					}(w)
				}
				// Send logs to the logChan channel
				for _, log := range logs {
					logChan <- log
				}

				// Collect results from logWorkers
				newHeadsWithPoolBalanceMetaData := NewHeadsWithPoolBalanceMetaData{
					Header:              h,
					PoolBalanceMetaData: make(map[common.Address]PoolBalanceMetaData),
				}
				for i := 0; i < len(logs); i++ {
					select {
					case result := <-results:
						newHeadsWithPoolBalanceMetaData.PoolBalanceMetaData[result.Address] = result
					case <-time.After(200 * time.Millisecond):
						log.Error("Timeout waiting for result from logWorker")
					}
				}

				// Create channels for Curve pools and results
				curvePoolsChan := make(chan string, len(allCurvePools))            // Channel to send Curve pools to curveWorkers
				curveResults := make(chan PoolBalanceMetaData, len(allCurvePools)) // Channel to collect results from curveWorkers
				// log.Info("NewHeads: curvePoolsChan and curveResults built")

				// Start Curve workers
				curveWg.Add(len(allCurvePools))
				for w := 1; w <= numWorkers; w++ {
					go func(id int) {
						curveWorker(id, curvePoolsChan, curveResults, &curveWg)
					}(w)
				}
				// Populate the curvePoolsChan with the global list of all Curve pools
				for _, pool := range allCurvePools {
					curvePoolsChan <- pool
				}
				for i := 0; i < len(allCurvePools); i++ {
					select {
					case result := <-curveResults:
						newHeadsWithPoolBalanceMetaData.PoolBalanceMetaData[result.Address] = result
					case <-time.After(200 * time.Millisecond): // 200 millisecond timeout
						log.Error("Timeout waiting for result from curveWorker")
					}
				}

				// Create channels for OneInch decaying pools and results
				oneInchPools := GetAllDecayingOneinchPoolsData(h.Number) // Get pools that need to be queried
				// log.Info("NewHeads: oneInchPools", "count", len(oneInchPools))
				oneInchPoolsChan := make(chan common.Address, len(oneInchPools))    // Channel to send OneInch pools to workers
				oneInchResults := make(chan PoolBalanceMetaData, len(oneInchPools)) // Channel to collect results from OneInch workers)

				// Start OneInch workers
				var oneInchWg MyWaitGroup
				oneInchWg.Add(len(oneInchPools))
				for w := 1; w <= numWorkers; w++ { // numOneInchWorkers is the number of worker goroutines you want to start
					go oneInchWorker(w, oneInchPoolsChan, oneInchResults, &oneInchWg)
				}

				// Populate the oneInchPoolsChan with pools
				for _, pool := range oneInchPools {
					oneInchPoolsChan <- pool.PoolAddress
				}

				// Collect results from OneInch workers
				for i := 0; i < len(oneInchPools); i++ {
					select {
					case result := <-oneInchResults:
						newHeadsWithPoolBalanceMetaData.PoolBalanceMetaData[result.Address] = result
					case <-time.After(200 * time.Millisecond): // Timeout waiting for worker
						log.Error("Timeout waiting for result from oneInchWorker")
					}
				}

				// Wait for all workers to finish and then close the channels
				// log.Info("NewHeads: waiting for log workers to finish...", "logWg.Count()", logWg.Count())
				logWg.Wait()
				// log.Info("NewHeads: waiting for curve workers to finish...", "curveWg.Count()", curveWg.Count())
				curveWg.Wait()
				// log.Info("NewHeads: waiting for oneInch workers to finish...", "oneInchWg.Count()", oneInchWg.Count())
				oneInchWg.Wait()

				close(logChan)
				close(curvePoolsChan)
				// Wait for OneInch workers to finish and then close the channels
				close(oneInchPoolsChan)
				// log.Info("NewHeads: all channels closed")

				notifier.Notify(rpcSub.ID, newHeadsWithPoolBalanceMetaData)
				// get the timestamp of the block
				log.Info("NewHeads: time to process logs and notify", "duration", time.Since(start))

			}
		}
	}()

	return rpcSub, nil
}