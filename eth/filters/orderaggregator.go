package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
)

// TODO nick clean up the file
type Order struct {
	OrderHash     string      `json:"orderHash"`
	OrderBookName string      `json:"orderBookName"`
	OffChainData  interface{} `json:"offChainData"`
	OnChainData   OnChainData `json:"onChainData,omitempty"`
}

type OnChainData struct {
	OrderInfo               interface{} `json:"orderInfo"`
	MakerBalance_weiUnits   *big.Int    `json:"makerBalance_weiUnits"`
	MakerAllowance_weiUnits *big.Int    `json:"makerAllowance_weiUnits"`
}

var (
	ctx       = context.Background()
	sharedRdb *redis.Client
	lastID    = "0"
	mu        sync.Mutex
	// In-memory data structure to store order data, keyed by orderHash
	orderDataStore = make(map[string]Order)
)

var OrderBookAggregatorServiceEnabled (bool) = false

func startOrderBookAggregatorService() {

	waitForHTTPServerToStart()

	log.Println("startOrderBookAggregatorService: orderbook aggregator started")
	if OrderBookAggregatorServiceEnabled {
		log.Println("startOrderBookAggregatorService: orderbook aggregator service is enabled")
	} else {
		log.Println("startOrderBookAggregatorService: orderbook aggregator service is disabled")
		return
	}

	initRedis()

	log.Println("startOrderBookAggregatorService: redis initialized")

	// TODO nick-0x test this as soon as you have the orderAggregator running. we need snapshots to test this
	// Fetch initial snapshot and initialize local state
	// fetchSnapshot()
	// log.Println("startOrderBookAggregatorService: initial snapshot fetched")
	// Start a goroutine to process updates continuously
	// processExistingOrders()

	go processUpdates()
	log.Println("startOrderBookAggregatorService: processUpdates started")
}

// TODO nick-0x test this as soon as you have the orderAggregator running.
func fetchSnapshot() {
	log.Println("fetchSnapshot: fetching initial snapshot")
	for {
		// Fetch the latest snapshot from shared Redis using XRevRange
		snapshot, err := sharedRdb.XRevRangeN(ctx, "snapshotStream", "+", "-", 1).Result()
		if err != nil {
			log.Printf("fetchSnapshot: Failed to read snapshot: %v", err)
			time.Sleep(5 * time.Second) // Retry after 5 seconds
			continue
		}

		if len(snapshot) == 0 {
			log.Println("fetchSnapshot: No snapshot found, retrying...")
			time.Sleep(5 * time.Second) // Retry after 5 seconds
			continue
		}

		// Initialize local state with the snapshot
		fmt.Println("fetchSnapshot: Snapshot:", snapshot)
		snapshotData := snapshot[0].Values["snapshot"].(string)
		err = json.Unmarshal([]byte(snapshotData), &orderDataStore)
		if err != nil {
			log.Printf("fetchSnapshot: Failed to unmarshal snapshot data: %v", err)
			time.Sleep(5 * time.Second) // Retry after 5 seconds
			continue
		}

		// Update the last_id to the ID of the snapshot
		lastID = snapshot[0].ID
		break
	}
}

func updateOrdersOnchainData(orderHash string) {
	// Retrieve existing order data
	mu.Lock()
	order := orderDataStore[orderHash]
	mu.Unlock()
	// log.Println("updateOrdersOnchainData: order:", order)

	// Handle missing or empty fields
	if order.OnChainData.MakerAllowance_weiUnits == nil || order.OnChainData.MakerBalance_weiUnits == nil || order.OnChainData.OrderInfo == nil {
		switch order.OrderBookName {
		case ORDERBOOKNAME_ZRX:
			zrxOrder, err := ZRXConvertOrderToZRXOrder(order)
			if err != nil {
				log.Printf("updateOrdersOnchainData: Failed to convert order to ZRXOrder: %v", err)
				return
			}
			onChainData, err := ZRXGetOnChainData(zrxOrder)
			if err != nil {
				log.Printf("updateOrdersOnchainData: Failed to get ZRX on-chain data: %v", err)
				return
			}
			order.OnChainData = onChainData
		case ORDERBOOKNAME_TEMPO:
			tempoOrder, err := tempoConvertOrderToTempoOrder(order)
			if err != nil {
				log.Printf("updateOrdersOnchainData: Failed to convert order to TempoOrder: %v", err)
				return
			}
			onChainData, err := tempoGetOnChainData(tempoOrder)
			if err != nil {
				log.Printf("updateOrdersOnchainData: Failed to get Tempo on-chain data: %v", err)
				return
			}
			order.OnChainData = onChainData
		// Add cases for other order books here
		default:
			log.Printf("updateOrdersOnchainData: Unknown order book name: %s", order.OrderBookName)
			return
		}

		// Write the update back to the stream if needed
		update := map[string]interface{}{
			"orderHash":   orderHash,
			"onChainData": order.OnChainData,
		}
		writeUpdateToStream(update)
	}
}

// TODO nick-0x test this as soon as you have the orderAggregator running. we need to have a order book to test this well
//
//	you might want to checkout processUpdates for things like convertValuesToStringsAndRemoveScientificNotation
func processExistingOrders() {
	log.Println("processExistingOrders: processing existing orders")
	for orderHash, orderData := range orderDataStore {
		// Process the order data
		fmt.Printf("processExistingOrders: Processing order data for %s: %v\n", orderHash, orderData)
		updateOrdersOnchainData(orderHash)
	}
	log.Println("processExistingOrders: all existing orders processed")
}

func processUpdates() {
	for {
		// Fetch updates from Redis
		updates, err := sharedRdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{"updateStream", lastID},
			Block:   0, // Blocking indefinitely for new updates
		}).Result()
		if err != nil {
			log.Println("processUpdates: Failed to read updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(updates) == 0 || len(updates[0].Messages) == 0 {
			log.Println("processUpdates: No updates found")
			continue
		}

	updateLoop:
		for _, update := range updates[0].Messages {
			lastID = update.ID

			// Create a new Order object to hold the update
			var orderUpdate Order

			// Deserialize the "data" field into a map
			var updateData map[string]interface{}
			if err := json.Unmarshal([]byte(update.Values["data"].(string)), &updateData); err != nil {
				log.Printf("processUpdates: Failed to unmarshal update data: %v", err)
				continue
			}

			// Convert all values in the map to strings
			updateData = convertValuesToStringsAndRemoveScientificNotation(updateData)

			// Iterate over the key-value pairs in the update
			var hasOffChainData bool = false
			for key, value := range updateData {
				switch key {
				case "orderHash":
					orderUpdate.OrderHash = value.(string)
				case "orderBookName":
					orderUpdate.OrderBookName = value.(string)
				case "offChainData":
					orderUpdate.OffChainData = value
					hasOffChainData = true
				case "deleted":
					// if the order is deleted, remove it from the orderDataStore
					if deleted, ok := value.(string); ok && strings.ToLower(deleted) == "true" {
						mu.Lock()
						delete(orderDataStore, orderUpdate.OrderHash)
						mu.Unlock()
						break updateLoop
					}
				}

			}
			// if updateData lacks orderHash, skip the update
			if orderUpdate.OrderHash == "" {
				log.Println("processUpdates: orderHash not found, skipping")
				continue
			}
			if !hasOffChainData {
				continue
			}

			// Retrieve existing order data or create a new entry if it doesn't exist
			mu.Lock()
			order, exists := orderDataStore[orderUpdate.OrderHash]
			if !exists {
				order = Order{OrderHash: orderUpdate.OrderHash}
			}

			if orderUpdate.OffChainData != nil {
				order.OffChainData = orderUpdate.OffChainData
			}
			if orderUpdate.OrderBookName != "" {
				order.OrderBookName = orderUpdate.OrderBookName
			}

			// Update the in-memory data store
			orderDataStore[order.OrderHash] = order
			mu.Unlock()

			updateOrdersOnchainData(order.OrderHash)
		}

		// this is required to release the lock to create the snapshot.
		// we might want to keep a timeout here on the nodes as well
		time.Sleep(50 * time.Millisecond)
	}

}

func writeUpdateToStream(updateMap map[string]interface{}) error {
	// Convert the updateMap to a byte slice
	data, err := json.Marshal(updateMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updateMap: %v", err)
	}

	// Write the data to the Redis stream
	err = sharedRdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "updateStream",
		Values: map[string]interface{}{
			"data": data,
		},
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to write update to stream: %v", err)
	}

	return nil
}

func getBalanceMetaData_OrderBooks(address common.Address, eventLog *Log) (interface{}, error) {
	switch address {
	case ORDERBOOKADDRESS_ZRX:
		return getBalanceMetaData_Zrx(eventLog)
	case ORDERBOOKADDRESS_TEMPO:
		return getBalanceMetaData_Tempo(eventLog)
	// Add cases for other order books here
	default:
		return "", fmt.Errorf("address not implemented in getBalanceMetaData_OrderBook: %s", address.Hex())
	}
}
