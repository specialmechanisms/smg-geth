// {'maker': '0xf7f9912512a5447295c872f35e157f4dd3f60af7', 'makerToken': '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2',
// 'makerAmount': 310000000000000000, 'takerTokenRecipient': '0xf7f9912512a5447295c872f35e157f4dd3f60af7',
// 'takerToken': '0x6b175474e89094c44da98b954eedeac495271d0f', 'takerAmountMin': 500000000000000000000,
// 'takerAmountDecayRate': 0, 'data': 182731631036575856403770922421532039922998,
// 'signature': '0x7ec8608224dd78be12fc43c91ea29c24b4e298815ee39720b6733c314b6671561193ddbcb43f34f0bf19b47f3f776249eaf28d36617a79924b46eb8c76ff6f031b',
// 'orderHash': '0xd8d443522637eb23b5355b881aa8062237877700bf7cfed9b9ee1034523dde8d',
//
//	'dataDecoded': {
//			'begin': 1729185078, 'expiry': 1729385078, 'partiallyFillable': True, 'authorization': False,
//			'usePermit2': False, 'nonce': 67}, 'makerAmountFilled': 0, 'status': 'fillable', 'statusTimeline': [{'status': 'fillable', 'timestamp': '2024-10-17T17:11:18Z', 'slotNumber': 10196754, 'blockNumber': 20986691}], 'fills': []}
package filters

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ORDERBOOKNAME_TEMPO    = "tempo"
	ORDERBOOKADDRESS_TEMPO = common.HexToAddress("0x93be362993d5B3954dbFdA49A1Ad1844c8083A30")
	PERMIT2ADDRESS         = common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3")
)

type TempoOrder struct {
	Order
	OffChainData TempoOffChainData_SignedOrder
	OnChainData  TempoOnChainData
}

type TempoOrderRaw struct {
	Order
	OffChainData TempoOffChainDataRaw
	OnChainData  TempoOnChainData
}

type TempoOffChainDataRaw struct {
	Maker                string `json:"maker"`
	MakerToken           string `json:"makerToken"`
	MakerAmount          string `json:"makerAmount"`
	TakerTokenRecipient  string `json:"takerTokenRecipient"`
	TakerToken           string `json:"takerToken"`
	TakerAmountMin       string `json:"takerAmountMin"`
	TakerAmountDecayRate string `json:"takerAmountDecayRate"`
	Data                 string `json:"data"`
	Signature            string `json:"signature"`
	OrderHash            string `json:"orderHash"`
}

type TempoOffChainData_SignedOrder struct {
	Order     TempoOffChainDataOrder `json:"order"`
	Signature []byte                 `json:"signature"`
	// not sure if we need this because it is already a field of Order
	// OrderHash              common.Hash            `json:"orderHash"`
}

type TempoOffChainDataOrder struct {
	Maker                common.Address `json:"maker"`
	MakerToken           common.Address `json:"makerToken"`
	MakerAmount          *big.Int       `json:"makerAmount"`
	TakerTokenRecipient  common.Address `json:"takerTokenRecipient"`
	TakerToken           common.Address `json:"takerToken"`
	TakerAmountMin       *big.Int       `json:"takerAmountMin"`
	TakerAmountDecayRate *big.Int       `json:"takerAmountDecayRate"`
	Data                 *big.Int       `json:"data"`
	// VerifyingContract    common.Address `json:"verifyingContract,omitempty"`
	// ChainID              int            `json:"chainId,omitempty"`
}

type TempoOnChainData struct {
	MakerBalance_weiUnits   *big.Int       `json:"makerBalance_weiUnits"`
	MakerAllowance_weiUnits *big.Int       `json:"makerAllowance_weiUnits"`
	OrderInfo               TempoOrderInfo `json:"orderInfo"`
}

type TempoOrderInfo struct {
	// OrderHash           common.Hash `json:"orderHash"`
	OrderStatus         int      `json:"orderStatus"`
	MakerAmountFilled   *big.Int `json:"makerAmountFilled"`
	MakerAmountFillable *big.Int `json:"makerAmountFillable"`
}

type TempoData struct {
	Begin                 int64
	Expiry                int64
	PartiallyFillable     bool
	AuthorizationRequired bool
	UsePermit2            bool
	Nonce                 *big.Int
}

// tempoDecodeOrderDataInt decodes the order data from a big.Int into a TempoData struct.
// The data is encoded in a specific format where each field is packed into specific bits of the big.Int.
// This function extracts each field using bitwise operations.
func tempoDecodeOrderDataInt(data *big.Int) (TempoData, error) {
	var tempoData TempoData

	// Extract the 'begin' field (first 64 bits)
	tempoData.Begin = new(big.Int).And(data, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1))).Int64()
	// Extract the 'expiry' field (next 64 bits)
	tempoData.Expiry = new(big.Int).And(new(big.Int).Rsh(data, 64), new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1))).Int64()
	// Extract the 'partiallyFillable' field (1 bit at position 128)
	tempoData.PartiallyFillable = intToBool(int(new(big.Int).And(new(big.Int).Rsh(data, 128), big.NewInt(0x1)).Int64()))
	// Extract the 'authorizationRequired' field (1 bit at position 129)
	tempoData.AuthorizationRequired = intToBool(int(new(big.Int).And(new(big.Int).Rsh(data, 129), big.NewInt(0x1)).Int64()))
	// Extract the 'usePermit2' field (1 bit at position 130)
	tempoData.UsePermit2 = intToBool(int(new(big.Int).And(new(big.Int).Rsh(data, 130), big.NewInt(0x1)).Int64()))
	// Extract the 'nonce' field (remaining bits starting at position 131)
	tempoData.Nonce = new(big.Int).Rsh(data, 131)

	return tempoData, nil
}

// MarshalJSON implements the json.Marshaler interface for TempoOffChainData_SignedOrder
func (o TempoOffChainData_SignedOrder) MarshalJSON() ([]byte, error) {
	type Alias TempoOffChainData_SignedOrder
	return json.Marshal(&struct {
		Signature string `json:"signature"`
		*Alias
	}{
		Signature: fmt.Sprintf("0x%x", o.Signature),
		Alias:     (*Alias)(&o),
	})
}

func tempoConvertOrderToTempoOrder(order Order) (TempoOrder, error) {
	tempoOrderRaw, err := tempoConvertOrderToTempoOrderRaw(order)
	if err != nil {
		return TempoOrder{}, fmt.Errorf(
			"tempoConvertOrderToTempoOrder: failed to convert order to TempoOrderRaw: %v", err)
	}
	tempoOrder, err := tempoConvertTempoOrderRawToTempoOrder(tempoOrderRaw)
	if err != nil {
		return TempoOrder{}, fmt.Errorf(
			"tempoConvertOrderToTempoOrder: failed to convert TempoOrderRaw to TempoOrder: %v", err)
	}
	return tempoOrder, nil
}

func tempoConvertOrderToTempoOrderRaw(order Order) (TempoOrderRaw, error) {
	var tempoOrderRaw TempoOrderRaw

	// convert the order to TempoOrderRaw
	tempoOrderRaw.Order = order

	// convert the offChainData to TempoOffChainDataRaw
	offChainData, ok := order.OffChainData.(TempoOffChainDataRaw)
	if !ok {
		// Attempt to manually unmarshal the offChainData
		data, err := json.Marshal(order.OffChainData)
		if err != nil {
			return tempoOrderRaw, fmt.Errorf("failed to marshal offChainData: %v", err)
		}
		var tempoOffChainDataRaw TempoOffChainDataRaw
		err = json.Unmarshal(data, &tempoOffChainDataRaw)
		if err != nil {
			return tempoOrderRaw, fmt.Errorf("failed to unmarshal offChainData: %v", err)
		}
		offChainData = tempoOffChainDataRaw
	}
	tempoOrderRaw.OffChainData = offChainData

	log.Printf("tempoConvertOrderToTempoOrderRaw: Successfully converted offChainData: %+v", offChainData)
	return tempoOrderRaw, nil
}

func tempoConvertTempoOrderRawToTempoOrder(tempoOrderRaw TempoOrderRaw) (TempoOrder, error) {
	var tempoOrder TempoOrder

	// convert the order to TempoOrder
	tempoOrder.Order = tempoOrderRaw.Order

	// convert the offChainData to TempoOffChainData_SignedOrder
	offChainData, err := tempoCreateOffChainData(tempoOrderRaw.OffChainData)
	if err != nil {
		return tempoOrder, fmt.Errorf("failed to convert offChainData to TempoOffChainData_SignedOrder: %v", err)
	}
	tempoOrder.OffChainData = offChainData

	return tempoOrder, nil
}

func tempoCreateOffChainData(tempoOffChainDataRaw TempoOffChainDataRaw) (TempoOffChainData_SignedOrder, error) {
	var tempoOffChainData TempoOffChainData_SignedOrder

	// Helper function to convert string to *big.Int
	toBigInt := func(value string) *big.Int {
		bigIntValue, ok := new(big.Int).SetString(value, 10)
		if !ok {
			log.Fatalf("invalid value: %s", value)
		}
		return bigIntValue
	}

	// Convert the order to TempoOffChainDataOrder
	offChainDataOrder := TempoOffChainDataOrder{
		Maker:                common.HexToAddress(tempoOffChainDataRaw.Maker),
		MakerToken:           common.HexToAddress(tempoOffChainDataRaw.MakerToken),
		MakerAmount:          toBigInt(tempoOffChainDataRaw.MakerAmount),
		TakerTokenRecipient:  common.HexToAddress(tempoOffChainDataRaw.TakerTokenRecipient),
		TakerToken:           common.HexToAddress(tempoOffChainDataRaw.TakerToken),
		TakerAmountMin:       toBigInt(tempoOffChainDataRaw.TakerAmountMin),
		TakerAmountDecayRate: toBigInt(tempoOffChainDataRaw.TakerAmountDecayRate),
		Data:                 toBigInt(tempoOffChainDataRaw.Data),
	}

	// Convert the signature to []byte
	signature := common.FromHex(tempoOffChainDataRaw.Signature)

	tempoOffChainData.Order = offChainDataOrder
	tempoOffChainData.Signature = signature

	return tempoOffChainData, nil
}

func tempoGetOnChainData(tempoOrder TempoOrder) (OnChainData, error) {
	var onChainData OnChainData

	// at this point we do only have the offchain data. we need to fetch the onchain data from the blockchain

	// get the maker balance
	onChainData.MakerBalance_weiUnits, err = GetERC20TokenBalance(
		tempoOrder.OffChainData.Order.Maker, tempoOrder.OffChainData.Order.MakerToken)
	if err != nil {
		return onChainData, fmt.Errorf("failed to get maker balance: %v", err)
	}

	// Get the maker allowance
	// set the spender address based on the permit2 flag
	var spender common.Address
	decodedDataInt, err := tempoDecodeOrderDataInt(tempoOrder.OffChainData.Order.Data)
	if err != nil {
		return onChainData, fmt.Errorf("failed to decode order data: %v", err)
	}
	if decodedDataInt.UsePermit2 {
		spender = PERMIT2ADDRESS
	} else {
		spender = ORDERBOOKADDRESS_TEMPO
	}
	onChainData.MakerAllowance_weiUnits, err = GetERC20TokenAllowance(
		tempoOrder.OffChainData.Order.MakerToken, tempoOrder.OffChainData.Order.Maker, spender)
	if err != nil {
		return onChainData, fmt.Errorf("failed to get maker allowance: %v", err)
	}

	// get the order info
	onChainData.OrderInfo, err = tempoGetOrderInfo(tempoOrder)
	if err != nil {
		return onChainData, fmt.Errorf("failed to get order info: %v", err)
	}

	return onChainData, nil

}

func tempoGetOrderInfo(tempoOrder TempoOrder) (TempoOrderInfo, error) {
	log.Println("tempoGetOrderInfo: tempoOrder: ", tempoOrder)
	log.Println("tempoGetOrderInfo: orderHash: ", tempoOrder.Order.OrderHash)

	var orderInfoResponse []interface{}

	instance_tempoContract := bind.NewBoundContract(ORDERBOOKADDRESS_TEMPO, parsedABI_Tempo, client, client, client)

	inputParameters := &struct {
		Order     TempoOffChainDataOrder `json:"order"`
		Signature []byte                 `json:"signature"`
	}{
		Order: TempoOffChainDataOrder{
			Maker:                tempoOrder.OffChainData.Order.Maker,
			MakerToken:           tempoOrder.OffChainData.Order.MakerToken,
			MakerAmount:          tempoOrder.OffChainData.Order.MakerAmount,
			TakerTokenRecipient:  tempoOrder.OffChainData.Order.TakerTokenRecipient,
			TakerToken:           tempoOrder.OffChainData.Order.TakerToken,
			TakerAmountMin:       tempoOrder.OffChainData.Order.TakerAmountMin,
			TakerAmountDecayRate: tempoOrder.OffChainData.Order.TakerAmountDecayRate,
			Data:                 tempoOrder.OffChainData.Order.Data,
		},
		Signature: tempoOrder.OffChainData.Signature,
	}

	// DEBUG BLOCK TODO nick remove this whole block after you tested on live
	// lets decode the data and print the values
	// Log the entire tempoOrder structure
	log.Println("tempoGetOrderInfo: tempoOrder: ", tempoOrder)
	// Log the OffChainData
	log.Println("tempoGetOrderInfo: tempoOrder.OffChainData: ", tempoOrder.OffChainData)
	// Log the Order
	log.Println("tempoGetOrderInfo: tempoOrder.OffChainData.Order: ", tempoOrder.OffChainData.Order)
	// Log the Data field
	log.Println("tempoGetOrderInfo: tempoOrder.OffChainData.Order.Data: ", tempoOrder.OffChainData.Order.Data)
	decodedData, err := tempoDecodeOrderDataInt(tempoOrder.OffChainData.Order.Data)
	if err != nil {
		log.Println("tempoGetOrderInfo: failed to decode order data: ", err)
		return TempoOrderInfo{}, err
	}
	log.Println("tempoGetOrderInfo: decodedData: ", decodedData)

	// Call the getOrderStatus function on the contract
	callOpts := &bind.CallOpts{}
	err = instance_tempoContract.Call(callOpts, &orderInfoResponse, "getOrderStatus", inputParameters)
	if err != nil {
		log.Println("tempoGetOrderInfo: failed to get order info: ", err)
		return TempoOrderInfo{}, err
	}
	log.Println("tempoGetOrderInfo: orderInfoResponse", orderInfoResponse)

	// Assert the response to the expected struct
	orderInfo := orderInfoResponse[0].(struct {
		OrderHash           [32]byte `json:"orderHash"`
		Status              uint8    `json:"status"`
		MakerAmountFilled   *big.Int `json:"makerAmountFilled"`
		MakerAmountFillable *big.Int `json:"makerAmountFillable"`
	})

	// make sure the OrderHash matches order.OrderHash
	log.Println("tempoOrder.Order.OrderHash", tempoOrder.Order.OrderHash)
	log.Println("orderInfo.OrderHash", common.BytesToHash(orderInfo.OrderHash[:]).Hex())
	if tempoOrder.Order.OrderHash != common.BytesToHash(orderInfo.OrderHash[:]).Hex() {
		log.Println("tempoGetOrderInfo: orderHash does not match. This can happen while testing," +
			"because the order was created in local env but we are querying the blockchain")
	}

	return TempoOrderInfo{
		OrderStatus:         int(orderInfo.Status),
		MakerAmountFilled:   orderInfo.MakerAmountFilled,
		MakerAmountFillable: orderInfo.MakerAmountFillable,
	}, nil
}

// TODO nick-0x test this as soon as you have the orderAggregator running. we need to have a order book to test this well
func getBalanceMetaData_Tempo(eventLog *Log) (TempoOrderInfo, error) {
	orderHash := common.BytesToHash(eventLog.Data[0:32])

	// get the offChain data from orderDataStore
	order, ok := orderDataStore[orderHash.Hex()]
	if !ok {
		log.Println("getBalanceMetaData_ZrxOrderBook: order not found in orderDataStore")
		return TempoOrderInfo{}, fmt.Errorf("failed to get order from orderDataStore")
	}

	// Check if OffChainData is of the expected type
	offChainData, ok := order.OffChainData.(TempoOffChainData_SignedOrder)
	if !ok {
		log.Println("getBalanceMetaData_TempoOrderBook: OffChainData is not of type TempoOffChainData_SignedOrder")
		return TempoOrderInfo{}, fmt.Errorf("OffChainData is not of type TempoOffChainData_SignedOrder")
	}

	// do the OrderInfo call
	tempoOrder := TempoOrder{
		Order:        order,
		OffChainData: offChainData,
	}
	return tempoGetOrderInfo(tempoOrder)
}
