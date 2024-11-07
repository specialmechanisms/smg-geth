package filters

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// create a struct for the redis config host port and password
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

var DefaultRedisConfig = RedisConfig{
	Host:     "localhost",
	Port:     6379,
	Password: "",
	DB:       0,
}

// ConvertWeiUnitsToEtherUnits_UsingTokenAddress takes in tokenAmount as a big.Int and token address,
// and returns the balance in ether units. It now also returns an error if it cannot complete the conversion.
func ConvertWeiUnitsToEtherUnits_UsingTokenAddress(tokenAmount *big.Int, tokenAddress string) (float64, error) {
	// Check if tokenAmount is nil, zero, or negative
	if tokenAmount == nil || tokenAmount.Sign() <= 0 {
		if tokenAmount.Sign() == 0 {
			return 0, nil
		}
		err := errors.New("tokenAmount is nil or negative")
		fmt.Printf("Error: %v, tokenAmount: %v", err, tokenAmount)
		return 0, err
	}

	// if tokenAddress is ether, then convert tokenAmount to ether units and return it
	for _, knownAddress := range KnownEthereumAddresses {
		if tokenAddress == knownAddress {
			// convert tokenAmount that are in wei units to ether units
			tokenAmount_float64 := new(big.Float).SetInt(tokenAmount)
			tokenAmount_etherUnits, _ := new(big.Float).Quo(tokenAmount_float64, big.NewFloat(math.Pow(10.0, 18))).Float64()
			return tokenAmount_etherUnits, nil // Return nil error on success
		}
	}

	// Check if tokenAddress is a valid Ethereum address
	if !common.IsHexAddress(tokenAddress) {
		err := fmt.Errorf("invalid token address: %s", tokenAddress)
		fmt.Printf("Error: %v, tokenAddress: %s", err, tokenAddress)
		return 0, err
	}
	var contractAddress common.Address = common.HexToAddress(tokenAddress)
	instance_ERC20 := bind.NewBoundContract(contractAddress, parsedABI_ERC20, client, client, client)
	// Get token decimals
	var tokenDecimals []interface{}
	callOpts := &bind.CallOpts{}
	err := instance_ERC20.Call(callOpts, &tokenDecimals, "decimals")
	if err != nil {
		fmt.Printf("Failed to retrieve token decimals: %v, tokenAddress: %s", err, tokenAddress)
		return 0, err // Return the error to the caller
	}
	if len(tokenDecimals) == 0 {
		err := fmt.Errorf("tokenDecimals is empty")
		fmt.Printf("Error: %v, tokenAddress: %s", err, tokenAddress)
		return 0, err // Return an error indicating tokenDecimals is empty
	}
	// Convert tokenAmount that are in wei units to ether units using the decimals
	tokenDecimals_float64 := float64(tokenDecimals[0].(uint8))
	tokenAmount_float64 := new(big.Float).SetInt(tokenAmount)
	tokenAmount_etherUnits, _ := new(big.Float).Quo(tokenAmount_float64,
		new(big.Float).Mul(big.NewFloat(math.Pow(10.0, tokenDecimals_float64)), big.NewFloat(1))).Float64()
	return tokenAmount_etherUnits, nil // Return nil error on success
}

// create a function that takes in tokemAmount as a bigInt and decimals as int and returns the balance in ether units
func ConvertWeiUnitsToEtherUnits_UsingDecimals(tokenAmount *big.Int, decimals int) (float64, error) {
	// Check if tokenAmount is nil, zero, or negative
	if tokenAmount == nil || tokenAmount.Sign() <= 0 {
		if tokenAmount.Sign() == 0 {
			return 0, nil
		}
		err := errors.New("tokenAmount is nil or negative")
		fmt.Printf("Error: %v, tokenAmount: %v", err, tokenAmount)
		return 0, err
	}
	// Check for valid decimals value (typically, 0 <= decimals <= 18 for Ethereum tokens)
	if decimals < 0 || decimals > 36 {
		return 0, fmt.Errorf("invalid decimals: %d. Decimals should be between 0 and 36", decimals)
	}
	// convert tokenAmount that are in wei units to ether units using the decimals
	tokenDecimals_float64 := float64(decimals)
	tokenAmount_float64 := new(big.Float).SetInt(tokenAmount)
	tokenAmount_etherUnits, _ := new(big.Float).Quo(tokenAmount_float64, new(big.Float).Mul(big.NewFloat(math.Pow(10.0, tokenDecimals_float64)), big.NewFloat(1))).Float64()
	return tokenAmount_etherUnits, nil
}

type MetaData_ERC20Balances struct {
	SenderAddress              common.Address `json:"SenderAddress"`
	SenderBalance_etherUnits   float64        `json:"SenderBalance_etherUnits"`
	ReceiverAddress            common.Address `json:"ReceiverAddress"`
	ReceiverBalance_etherUnits float64        `json:"ReceiverBalance_etherUnits"`
}

type MetaData_ERC20Allowance struct {
	OwnerAddress      common.Address `json:"OwnerAddress"`
	SpenderAddress    common.Address `json:"SpenderAddress"`
	Amount_etherUnits float64        `json:"Amount_etherUnits"`
}

func GetERC20TokenBalance(wallet common.Address, token common.Address) (*big.Int, error) {
	instance_ERC20 := bind.NewBoundContract(token, parsedABI_ERC20, client, client, client)

	result := []interface{}{new(*big.Int)}
	callOpts := &bind.CallOpts{}
	err = instance_ERC20.Call(callOpts, &result, "balanceOf", wallet)
	if err != nil {
		return nil, err
	}
	balance_wei := *result[0].(**big.Int)
	return balance_wei, nil
}

func GetERC20TokenAllowance(token common.Address, owner common.Address, spender common.Address) (*big.Int, error) {
	instance_ERC20 := bind.NewBoundContract(token, parsedABI_ERC20, client, client, client)

	result := []interface{}{new(*big.Int)}
	callOpts := &bind.CallOpts{}
	err = instance_ERC20.Call(callOpts, &result, "allowance", owner, spender)
	if err != nil {
		return nil, err
	}
	allowance_wei := *result[0].(**big.Int)
	return allowance_wei, nil
}

func getBalanceMetaData_ERC20_Transfer(tokenAddress common.Address, eventLog *Log) (MetaData_ERC20Balances, error) {
	var result MetaData_ERC20Balances

	// make sure the event log has a length of at least 3
	if len(eventLog.Topics) != 3 {
		return result, fmt.Errorf("event log does not have the correct number of topics. token: %s", tokenAddress.Hex())
	}

	senderAddress := common.HexToAddress(eventLog.Topics[1].Hex())
	receiverAddress := common.HexToAddress(eventLog.Topics[2].Hex())

	senderBalance_weiUnits, err := GetERC20TokenBalance(senderAddress, tokenAddress)
	if err != nil {
		return result, fmt.Errorf("failed to get balance: %v", err)
	}
	receiverBalance_weiUnits, err := GetERC20TokenBalance(receiverAddress, tokenAddress)
	if err != nil {
		return result, fmt.Errorf("failed to get balance: %v", err)
	}

	senderBalance_etherUnits, err := ConvertWeiUnitsToEtherUnits_UsingTokenAddress(
		senderBalance_weiUnits, tokenAddress.Hex())
	if err != nil {
		return result, fmt.Errorf("failed to convert balance to ether units: %v", err)
	}
	receiverBalance_etherUnits, err := ConvertWeiUnitsToEtherUnits_UsingTokenAddress(
		receiverBalance_weiUnits, tokenAddress.Hex())
	if err != nil {
		return result, fmt.Errorf("failed to convert balance to ether units: %v", err)
	}

	result = MetaData_ERC20Balances{
		SenderAddress:              senderAddress,
		SenderBalance_etherUnits:   senderBalance_etherUnits,
		ReceiverAddress:            receiverAddress,
		ReceiverBalance_etherUnits: receiverBalance_etherUnits,
	}
	return result, nil
}

func getBalanceMetaData_ERC20_Allowance(tokenAddress common.Address, eventLog *Log) (MetaData_ERC20Allowance, error) {
	// emit Approval(msg.sender, usr, wad);
	var result MetaData_ERC20Allowance
	// we do not need to make a call because everything is already in the event log

	// make sure the event log has a length of at least 3
	if len(eventLog.Topics) != 3 {
		return result, fmt.Errorf(
			"event log does not have the correct number of topics. token: %s", tokenAddress.Hex())
	}

	ownerAddress := common.HexToAddress(eventLog.Topics[1].Hex())
	spenderAddress := common.HexToAddress(eventLog.Topics[2].Hex())
	amount_weiUnits := new(big.Int).SetBytes(eventLog.Data)
	amount_etherUnits, err := ConvertWeiUnitsToEtherUnits_UsingTokenAddress(
		amount_weiUnits, tokenAddress.Hex())
	if err != nil {
		return result, fmt.Errorf("failed to convert allowance to ether units: %v", err)
	}
	result = MetaData_ERC20Allowance{
		OwnerAddress:      ownerAddress,
		SpenderAddress:    spenderAddress,
		Amount_etherUnits: amount_etherUnits,
	}
	return result, nil
}

func getBalanceMetaData_ERC20(tokenAddress common.Address, eventLog *Log) (interface{}, error) {
	// Get the balance of the token

	// make sure the event log has a length of at least 3
	if len(eventLog.Topics) != 3 {
		return nil, fmt.Errorf("event log does not have the correct number of topics. token: %s", tokenAddress.Hex())
	}

	// call the decimals function of the token to make increase the chance of the token to be a valid ERC20 token
	_, err := GetTokenDecimals(tokenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get token decimals for tokenAddress: %v", tokenAddress)
	}

	switch eventLog.Topics[0].Hex() {
	case topic_erc20Transfer:
		return getBalanceMetaData_ERC20_Transfer(tokenAddress, eventLog)
	case topic_erc20Allowance:
		return getBalanceMetaData_ERC20_Allowance(tokenAddress, eventLog)
	default:
		return "", fmt.Errorf("failed to get balance meta data for ERC20 event log")
	}
}

func GetTokenDecimals(tokenAddress common.Address) (int, error) {
	// get token decimals
	var tokenDecimals []interface{}

	for _, knownAddress := range KnownEthereumAddresses {
		if tokenAddress.Hex() == knownAddress {
			return 18, nil
		}
	}

	instance_token := bind.NewBoundContract(tokenAddress, parsedABI_ERC20, client, client, client)
	callOpts := &bind.CallOpts{}
	err := instance_token.Call(callOpts, &tokenDecimals, "decimals")
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve value of variable: %v", err)
	}
	return int(tokenDecimals[0].(uint8)), nil
}

// intToBool converts an integer to a boolean.
func intToBool(i int) bool {
	return i != 0
}

func convertValuesToStringsAndRemoveScientificNotation(data map[string]interface{}) map[string]interface{} {
	for key, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			data[key] = convertValuesToStringsAndRemoveScientificNotation(v)
		case []interface{}:
			for i, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					v[i] = convertValuesToStringsAndRemoveScientificNotation(itemMap)
				} else {
					v[i] = fmt.Sprintf("%v", item)
				}
			}
			data[key] = v
		case float64:
			data[key] = fmt.Sprintf("%.0f", v)
		case int:
			data[key] = fmt.Sprintf("%d", v)
		default:
			data[key] = fmt.Sprintf("%v", value)
		}
	}
	return data
}

func waitForHTTPServerToStart() {
	// doing a random call until we get a valid response to know that the server has started
	for {
		_, err := GetERC20TokenBalance(
			common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
			common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"))
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

// initRedis initializes the Redis client using the values from DefaultRedisConfig.
func initRedis() {
	log.Println("initRedis: initializing redis...")

	// Initialize Redis client using DefaultRedisConfig
	sharedRdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", DefaultRedisConfig.Host, DefaultRedisConfig.Port),
		Password: DefaultRedisConfig.Password,
		DB:       DefaultRedisConfig.DB,
	})
	// log the hostname
	log.Println("initRedis: HOSTNAME: ", DefaultRedisConfig.Host)
	log.Println("initRedis: connected to redis")
}
