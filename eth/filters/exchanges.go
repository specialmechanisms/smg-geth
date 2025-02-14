package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

var client *ethclient.Client
var parsedABI_uniswapv2 abi.ABI
var parsedABI_ERC20 abi.ABI
var parsedABI_uniswapv3_multicall abi.ABI
var parsedABI_pancakeswapv3_multicall abi.ABI
var parsedABI_uniswapv3_pool abi.ABI
var parsedABI_balancerv2_vault abi.ABI
var parsedABI_balancerv2_pool abi.ABI
var parsedABI_OneInchV2_Mooniswap_Pool abi.ABI
var parsedABI_Curve_V2Pool abi.ABI
var parsedABI_Curve_V2Pool_2Tokens abi.ABI
var parsedABI_Curve_LPToken abi.ABI
var parsedABI_ZRXV4 abi.ABI
var parsedABI_Tempo abi.ABI
var activeOneInchV2DecayPeriods map[common.Address]OneInchV2DecayPeriod
var blacklistArray_OneinchV2 []string
var blacklist_OneinchV2 map[string]bool
var Balancerv2TestPoolIds []string
var KnownEthereumAddresses []string
var activeOneInchV2DecayPeriods_Lock sync.Mutex

func init() {
	var err error
	client, err = ethclient.Dial("http://localhost:8545")
	if err != nil {
		log.Error("Failed to connect to the Ethereum client: %v", err)
	}

	parsedABI_uniswapv2, err = abi.JSON(strings.NewReader(ABI_UniswapV2))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_uniswapv3_multicall, err = abi.JSON(strings.NewReader(ABI_UniswapV3_Multicall))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_pancakeswapv3_multicall, err = abi.JSON(strings.NewReader(ABI_PancakeSwapV3_Multicall))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_uniswapv3_pool, err = abi.JSON(strings.NewReader(ABI_UniswapV3_Pool))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_balancerv2_vault, err = abi.JSON(strings.NewReader(ABI_BalancerV2_Vault))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_balancerv2_pool, err = abi.JSON(strings.NewReader(ABI_BalancerV2_WeightedPool))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_OneInchV2_Mooniswap_Pool, err = abi.JSON(strings.NewReader(ABI_OneInchV2_Mooniswap_Pool))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_Curve_V2Pool, err = abi.JSON(strings.NewReader(ABI_Curve_V2Pool))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_Curve_V2Pool_2Tokens, err = abi.JSON(strings.NewReader(ABI_Curve_V2Pool_2Tokens))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_Curve_LPToken, err = abi.JSON(strings.NewReader(ABI_Curve_LPToken))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_ERC20, err = abi.JSON(strings.NewReader(ABI_ERC20))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_ZRXV4, err = abi.JSON(strings.NewReader(ABI_ZRXV4))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}
	parsedABI_Tempo, err = abi.JSON(strings.NewReader(ABI_Tempo))
	if err != nil {
		log.Error("Failed to parse contract ABI: %v", err)
	}

	activeOneInchV2DecayPeriods = make(map[common.Address]OneInchV2DecayPeriod)
	// those pools are bugged, never have any TXs on it and cause div by zero errors which slows down the node and sometimes even crashes it
	blacklistArray_OneinchV2 = []string{"0x22c6289db7e8eab6aa12c35a044410327c4d9f93", "0x642eef38c4a9bad89c264e8b567ab81b42373c09", "0xb30116c66483fd87de3cbe14e7e0a764b632e5ca", "0x3d1a7a585c5e6ce429e8bd74b2f44db3b643462d", "0x1a5dab604981cb24980b6b6646718f832ec7af3c", "0xf3cd77de1b99610441978bf542cae7d58673fb7d", "0xfde1ba8b3ef2d8a9d2e1665deb62a1a49fd4e4d1", "0xcad46eff448e22802359488b5873f5886242e438", "0xe64f91911883ea26c146fb094fab5dcec14a0925", "0xa7363cf21b379865eb04475e70901bc2fe0dbb5d", "0x5c2e1346c91313d55c6573276ba797e0d56f1b8e", "0x12e7d705025d5e5bd465d4153c1425cad5cefeb3", "0xd138b94dd76a63ed6683764760d030ee1f6017d7", "0x9cc0bf948954f90877624c09d2c5a28b0d40b2f7", "0x2c8b7fb5814c878805dc09544594977af1c0c3c2", "0x71c6e39a7945df102e9ef5d9f3534d3ca923d8f4", "0x339f882e761cb568127a674b61a4a835e2b97975", "0x713749988b4c6ed072fdc8c9063d0d10432f4743", "0xce7ae35e05ba226dcdd2ed23c974b7f90744c6d9", "0x0f0e6e11dccb4d08cd3e0e6501a9e5bd4fba94b8", "0xca6993d4a4e1ae19f98e64ee4069c43d759461ee", "0x611fcab149b34b857e8526cd1500c92f6ecae313", "0x5b445e5c775c93cddde2f307032dcb6efac6a594", "0x90f2fad22c3796ce5ac3a466e9a6c707bf72b512", "0xefa579e56923b876880c376280563b4d0232d2d1", "0x062d37257479435da4ba3c0a8c2e376da027e29b", "0x335565d37ac8554fe16dcb817514150d3059da1c", "0xb4cc12a091ff1778d19fd13461265dacb3f1599c", "0x45187b597f55a9bd9ef218bd833a4ac102d8ed34", "0x70c34a9b4d7a5c16f902a5274f9fb7ac18908daf", "0x98bdf79efc11a58bfe349ade4df8073cc452e283", "0x653a191f7b41319c4931f40c5b2e8b5c7bacc5b1", "0x4f339ff82ecf7efb987534df98602d0fafa4680f", "0xaf9f860bf2e67bd5227c04a8978d851740e94af3", "0x52959f4fd37c2e1f1ef45d48b9135fb523e0f831", "0xae1cddedd0320a6cadb1075d8094df2beb056bac", "0xc2946d6c8698cb0276b711757380d3ba43663576", "0x24df89e467b134a860c72f291297edd4e6f82c73", "0x084f67f06518572adf435d12abdfb455acaa90a1", "0x4020819c2c96962460fd1ce0c1eba8a52747d4a5", "0x6023cbb069ccbda746340e65e2b3094558e50d7d", "0xb2a26335f7215748f28001cf3c64dc5fadc9de43", "0x1a33e47f47318e5879998493e9de89966370fcb7", "0x18677c37f4142127598c872054e4dbb8cfbee202", "0x30e5d83276bc7ae54cb448df93e80212944d9ee9", "0x89bff5e0aec73979caac7ee2aa8c9a9773e3c9e0", "0x386b3ac9afab9e0f8a39d4cdd1585bcbda165f85", "0xfce443acd6aad3ea497dff392249bc75093d160f", "0x51a6315bf905b973409888dfd69a9958ae1982bf", "0x5267065e406cca5647551fd0239841bd280c1332", "0x94eb287e8b23908307b046dde327dc9feb8595fe", "0x57b7e58e427831a1a0fc0ceef11b21459c35ba3c", "0x304dceb0fabbafc08c02adcf492ddfc26a48f5ee", "0x2bc4b2bd0eab774a12e2a42df214cee1676bb6ff", "0xd688123dfffcb9c215def2ce6aa503f6e55717a7", "0xfebbe4cccbad7c6d6cff6272b1f3ad36bb1cc798", "0xa097d9d3a67b4b5cefe75fb97b6e817fb78b12d7", "0x9ac1359e4e70cfe77bb33fa659f809afefd7c64f", "0x23ecf94669570778afe5b14c8c320f285da42550", "0xe07c3809ac9fedebe249c8fdfc7615dc4c4cec60", "0x358a9447ee1d23b3995b70442d02dec221a7a7b9", "0x036d35985c250f394ae590807a27018aae225c2a"}
	// Create a map for constant-time lookups
	blacklist_OneinchV2 = make(map[string]bool)
	Balancerv2TestPoolIds = []string{"0x32296969ef14eb0c6d29669c550d4a0449130230000200000000000000000080", "0x1e19cf2d73a72ef1332c882f20534b6519be0276000200000000000000000112", "0x1e19cf2d73a72ef1332c882f20534b6519be0276000200000000000000000112", "0x06df3b2bbb68adc8b0e302443692037ed9f91b42000000000000000000000063", "0x514f35a92a13bc7093f299af5d8ebb1387e42d6b0002000000000000000000c9", "0x27c9f71cc31464b906e0006d4fcbc8900f48f15f00020000000000000000010f", "0xa7ff759dbef9f3efdd1d59beee44b966acafe214000200000000000000000180", "0xde8c195aa41c11a0c4787372defbbddaa31306d2000200000000000000000181", "0x9137f3a026fa419a7a9a0ba8df6601d4b0abfd260002000000000000000001ab", "0xe8cc7e765647625b95f59c15848379d10b9ab4af0002000000000000000001de", "0x1bccaac02bae336c6352acc3b772059ef1142fa70002000000000000000001f0", "0xbc5f4f9332d8415aaf31180ab4661c9141cc84e4000200000000000000000262", "0x0578292cb20a443ba1cde459c985ce14ca2bdee5000100000000000000000269", "0x8eb6c82c3081bbbd45dcac5afa631aac53478b7c000100000000000000000270", "0xf506984c16737b1a9577cadeda02a49fd612aff80002000000000000000002a9", "0x99a14324cfd525a34bbc93ac7e348929909d57fd00020000000000000000030e", "0x48607651416a943bf5ac71c41be1420538e78f87000200000000000000000327", "0x8167a1117691f39e05e9131cfa88f0e3a620e96700020000000000000000038c", "0xdbc4f138528b6b893cbcc3fd9c15d8b34d0554ae0002000000000000000003bf", "0xd1ec5e215e8148d76f4460e4097fd3d5ae0a35580002000000000000000003d3", "0x5512a4bbe7b3051f92324bacf25c02b9000c4a500001000000000000000003d7", "0xe4010ef5e37dc23154680f23c4a0d48bfca91687000200000000000000000432", "0xfd1cf6fd41f229ca86ada0584c63c49c3d66bbc9000200000000000000000438"}
	KnownEthereumAddresses = []string{"0x0000000000000000000000000000000000000000", "0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE"}
}

type WrongFactoryAddressError struct {
	Address string
}

func (e WrongFactoryAddressError) Error() string {
	return fmt.Sprintf("pool has wrong factory address: %s", e.Address)
}

type OneInchV2DecayPeriod struct {
	PoolAddress common.Address
	StartBlock  big.Int
	EndBlock    big.Int
}

func AddPoolToActiveOneInchV2DecayPeriods(poolAddress common.Address, startBlock *big.Int) {
	// log.Info("Adding pool to active decay periods", "poolAddress", poolAddress.Hex(), "startBlock", startBlock)
	instance_OneInchV2_Mooniswap_Pool := bind.NewBoundContract(poolAddress, parsedABI_OneInchV2_Mooniswap_Pool, client, client, client)

	// Fetch the decayPeriod from the pool
	var decayPeriodResponse []interface{}
	callOpts := &bind.CallOpts{}
	err := instance_OneInchV2_Mooniswap_Pool.Call(callOpts, &decayPeriodResponse, "decayPeriod")
	if err != nil {
		log.Info("Error fetching decayPeriod from contract:", "err", err)
		return // Consider handling the error appropriately
	}
	// log.Info("decayPeriodResponse", "decayPeriodResponse", decayPeriodResponse)
	// Convert the decayPeriod to a big.Int
	decayPeriod := decayPeriodResponse[0].(*big.Int)
	// log.Info("decayPeriod", "decayPeriod", decayPeriod)

	// The decay period is in seconds, so we need to convert it to blocks and add a buffer of 3 blocks
	decayPeriodInBlocks := new(big.Int).Div(decayPeriod, big.NewInt(13))
	decayPeriodInBlocks.Add(decayPeriodInBlocks, big.NewInt(3))
	// log.Info("decayPeriodInBlocks", "decayPeriodInBlocks", decayPeriodInBlocks)

	// Calculate the endBlock
	endBlock := new(big.Int).Add(startBlock, decayPeriodInBlocks)
	// log.Info("endBlock", "endBlock", endBlock)

	// Add or overwrite the pool in the list of active decay periods
	activeOneInchV2DecayPeriods_Lock.Lock()
	activeOneInchV2DecayPeriods[poolAddress] = OneInchV2DecayPeriod{
		PoolAddress: poolAddress,
		StartBlock:  *startBlock,
		EndBlock:    *endBlock,
	}
	activeOneInchV2DecayPeriods_Lock.Unlock()
	// log.Info("activeOneInchV2DecayPeriods", "activeOneInchV2DecayPeriods", activeOneInchV2DecayPeriods)
}

func GetAllDecayingOneinchPoolsData(currentBlock *big.Int) []OneInchV2DecayPeriod {
	validPools := make([]OneInchV2DecayPeriod, 0)

	for address, period := range activeOneInchV2DecayPeriods {
		if period.EndBlock.Cmp(currentBlock) > 0 {
			// Pool is still within the decay period
			validPools = append(validPools, period)
		} else {
			// Decay period has ended, remove it from the map
			delete(activeOneInchV2DecayPeriods, address)
		}
	}

	return validPools
}

type MetaData_OneInchV2 struct {
	Balance_token0_src  float64
	Balance_token1_src  float64
	Balance_token0_dst  float64
	Balance_token1_dst  float64
	ExchangeFee         float64
	ExchangeSlippageFee float64
}

// some pools are broken. this function helps us to filter them out
func isDivisionByZeroError(err error) bool {
	return strings.Contains(err.Error(), "SafeMath: division by zero")
}

func getBalanceMetaData_OneInchV2(poolAddress string) (MetaData_OneInchV2, error) {
	var metaData MetaData_OneInchV2

	instance_OneInchV2_Mooniswap_Pool := bind.NewBoundContract(common.HexToAddress(poolAddress), parsedABI_OneInchV2_Mooniswap_Pool, client, client, client)

	// Fetch the tokens from the pool
	var rawTokensResponse []interface{}
	callOpts := &bind.CallOpts{}
	err := instance_OneInchV2_Mooniswap_Pool.Call(callOpts, &rawTokensResponse, "getTokens")
	if err != nil {
		log.Info("Error fetching tokens from contract:", "err", err)
		return metaData, err
	}

	var tokens []common.Address
	for _, rawTokenData := range rawTokensResponse {
		addrSlice, ok := rawTokenData.([]common.Address)
		if !ok {
			log.Info("Error: Unexpected format for tokens in response")
			continue
		}
		tokens = append(tokens, addrSlice...)
	}

	balancesForAddition := make([]*big.Int, 0)
	balancesForRemoval := make([]*big.Int, 0)
	for i, token := range tokens {
		if i > 1 {
			log.Info("Warning: More than 2 tokens detected for pool: Skipping this pool due to faulty contract.", "poolAddress", poolAddress)
			return metaData, nil // Returning the current metaData without any further processing
		}
		// Get balance for addition
		var balanceForAdditionResponse []interface{}
		err = instance_OneInchV2_Mooniswap_Pool.Call(callOpts, &balanceForAdditionResponse, "getBalanceForAddition", token)
		if err != nil {
			if isDivisionByZeroError(err) {
				log.Info("Warning: Division by zero error detected for pool: Skipping this pool due to faulty contract.", "poolAddress", poolAddress)
				return metaData, nil // Returning the current metaData without any further processing
			}
			return metaData, err // For other errors
		}
		balanceForAdditionResponseBigInt := balanceForAdditionResponse[0].(*big.Int)
		balancesForAddition = append(balancesForAddition, balanceForAdditionResponseBigInt)

		// Get balance for removal
		var balanceForRemovalResponse []interface{}
		err = instance_OneInchV2_Mooniswap_Pool.Call(callOpts, &balanceForRemovalResponse, "getBalanceForRemoval", token)
		if err != nil {
			if isDivisionByZeroError(err) {
				log.Info("Warning: Division by zero error detected for pool:  Skipping this pool due to faulty contract.", "poolAddress", poolAddress)
				return metaData, nil // Returning the current metaData without any further processing
			}
			return metaData, err // For other errors
		}
		balanceForRemovalResponseBigInt := balanceForRemovalResponse[0].(*big.Int)
		balancesForRemoval = append(balancesForRemoval, balanceForRemovalResponseBigInt)
	}

	metaData.Balance_token0_src, err = ConvertWeiUnitsToEtherUnits_UsingTokenAddress(balancesForAddition[0], tokens[0].Hex())
	if err != nil {
		return metaData, err
	}
	metaData.Balance_token0_dst, err = ConvertWeiUnitsToEtherUnits_UsingTokenAddress(balancesForRemoval[0], tokens[0].Hex())
	if err != nil {
		return metaData, err
	}
	metaData.Balance_token1_src, err = ConvertWeiUnitsToEtherUnits_UsingTokenAddress(balancesForAddition[1], tokens[1].Hex())
	if err != nil {
		return metaData, err
	}
	metaData.Balance_token1_dst, err = ConvertWeiUnitsToEtherUnits_UsingTokenAddress(balancesForRemoval[1], tokens[1].Hex())
	if err != nil {
		return metaData, err
	}
	// log.Info(poolAddress, "metaData", metaData)

	// Fetch exchange fee
	var exchangeFeeResponse []interface{}
	err = instance_OneInchV2_Mooniswap_Pool.Call(callOpts, &exchangeFeeResponse, "fee")
	if err != nil {
		return metaData, err
	}
	metaData.ExchangeFee, err = ConvertWeiUnitsToEtherUnits_UsingDecimals(exchangeFeeResponse[0].(*big.Int), 18)
	if err != nil {
		return metaData, err
	}

	// Fetch slippage fee
	var slippageFeeResponse []interface{}
	err = instance_OneInchV2_Mooniswap_Pool.Call(callOpts, &slippageFeeResponse, "slippageFee")
	if err != nil {
		return metaData, err
	}
	metaData.ExchangeSlippageFee, err = ConvertWeiUnitsToEtherUnits_UsingDecimals(slippageFeeResponse[0].(*big.Int), 18)
	if err != nil {
		return metaData, err
	}

	return metaData, nil
}

func GetAllPools_OneInchV2() ([]string, error) {
	var pools []string

	// Populate the blacklist map from the array
	for _, pool := range blacklistArray_OneinchV2 {
		blacklist_OneinchV2[pool] = true
	}

	var data map[string]map[string]interface{}
	err := json.Unmarshal([]byte(Cache_OneInchV2), &data)
	if err != nil {
		return nil, err
	}

	for _, item := range data {
		if exchange, ok := item["exchange"].(string); ok {
			// Check if the pool is in the blacklist
			if _, isBlacklisted := blacklist_OneinchV2[exchange]; !isBlacklisted {
				pools = append(pools, exchange)
			}
		} else {
			return nil, fmt.Errorf("missing or invalid exchange field in item")
		}
	}

	return pools, nil
}

type MetaData_BalancerV2 struct {
	Address        string
	Tokens         []string
	Balances       []float64
	Fee            float64
	ScalingFactors []*big.Int
}

func getBalanceMetaData_BalancerV2(poolId common.Hash) (MetaData_BalancerV2, common.Address, error) {
	// the event gets fired on the vault contract and not on the pool.
	// we will get the poolAddress from the poolId and return the poolAddress the address of PoolBalanceMetaData struct outside this function can get updated
	var metaData MetaData_BalancerV2

	var vaultAddress common.Address = common.HexToAddress("0xBA12222222228d8Ba445958a75a0704d566BF2C8")
	instance_balancerv2_vault := bind.NewBoundContract(vaultAddress, parsedABI_balancerv2_vault, client, client, client)

	poolAddress := common.BytesToAddress(poolId.Bytes()[:20])
	metaData.Address = poolAddress.Hex()
	instance_balancerv2_weightedPool := bind.NewBoundContract(poolAddress, parsedABI_balancerv2_pool, client, client, client)

	// TODO nick-smc check the vault of the pool to make sure it is a balancer pool (like in the uniswapv3 function)

	// we need to do 3 calls
	// 1. getPoolTokens -> on vault
	// 2. getPoolFees -> on the pool
	// 3. getPoolScalingFactors -> on the pool, but sometimes there is no scaling factors, so we need to handle that case

	// 1. getPoolTokens
	var tokensAndBalances []interface{}
	callOpts := &bind.CallOpts{}
	err := instance_balancerv2_vault.Call(callOpts, &tokensAndBalances, "getPoolTokens", poolId)
	if err != nil {
		log.Info("getBalanceMetaData_BalancerV2: Failed to retrieve value of variable:", "err", err)
		return metaData, poolAddress, err
	}
	addresses := tokensAndBalances[0].([]common.Address)
	balances := tokensAndBalances[1].([]*big.Int)

	for i := 0; i < len(addresses); i++ {
		token := addresses[i].Hex()
		metaData.Tokens = append(metaData.Tokens, token)
		balance_wei := balances[i]
		balance_ether, err := ConvertWeiUnitsToEtherUnits_UsingTokenAddress(balance_wei, token)
		if err != nil {
			log.Info("getBalanceMetaData_BalancerV2: Failed to convert balance to ether units:", "err", err)
			return metaData, poolAddress, err
		}
		metaData.Balances = append(metaData.Balances, balance_ether)
	}

	// 2. getSwapFeePercentage
	var poolFee []interface{}
	err = instance_balancerv2_weightedPool.Call(callOpts, &poolFee, "getSwapFeePercentage")
	if err != nil {
		log.Info("getBalanceMetaData_BalancerV2: Failed to retrieve value of variable:", "err", err)
		return metaData, poolAddress, err
	}
	fee_bigInt := poolFee[0].(*big.Int)
	// divide fee_bigInt by 1e18. i just use WETH contract here because it has 18 decimals
	metaData.Fee, err = ConvertWeiUnitsToEtherUnits_UsingTokenAddress(fee_bigInt, "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	if err != nil {
		log.Info("getBalanceMetaData_BalancerV2: Failed to convert fee to ether units:", "err", err)
		return metaData, poolAddress, err
	}

	// // 3. getPoolScalingFactors
	var poolScalingFactors []interface{}
	err = instance_balancerv2_weightedPool.Call(callOpts, &poolScalingFactors, "getScalingFactors")
	if err != nil {
		if strings.Contains(err.Error(), "execution reverted") {
			log.Info("The above execution reverted warning can be ignored. it is handled in the code and expected to happen.")
			// The getScalingFactors function doesn't exist for this pool.
			// Continue without logging an error.
			metaData.ScalingFactors = nil // Explicitly set ScalingFactors to nil
		} else {
			// An unexpected error occurred.
			log.Info("getBalanceMetaData_BalancerV2: Failed to retrieve value of variable:", "err", err)
			return metaData, poolAddress, err
		}
	} else {
		metaData.ScalingFactors = poolScalingFactors[0].([]*big.Int)
	}

	return metaData, poolAddress, nil
}

//	BEGIN UNISWAPV3 MULTICALL
//
// Structs for unmarshaling contract data
// These structs match the field names in the Ethereum contract
type TickData struct {
	Tick           *big.Int
	LiquidityNet   *big.Int
	LiquidityGross *big.Int
}

type ContractResponse struct {
	SqrtPriceX96 *big.Int
	Liquidity    *big.Int
	TickData     []TickData
}

// Structs for marshaling JSON response
// These structs use JSON field names used by ninja (python)
type Ticks struct {
	Tick           *big.Int `json:"tick"`
	LiquidityNet   *big.Int `json:"liquidityNet"`
	LiquidityGross *big.Int `json:"liquidityGross"`
}

type ResponseStruct_UniswapV3Multicall struct {
	SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	Liquidity    *big.Int `json:"liquidity"`
	Ticks        []Ticks  `json:"ticks"`
}

func getBalanceMetaData_UniswapV3(poolAddress string) (ResponseStruct_UniswapV3Multicall, error) {
	var metaData ResponseStruct_UniswapV3Multicall

	// The response_factoryCall is a pointer to a common.Address that will store the address returned by the factory function of the UniswapV3 pool contract.
	// The responseSlice is a slice of interface{} that holds a pointer to response_factoryCall, and it’s used because the Call function expects its second argument
	//  to be a pointer to a slice of interface{}, where it will store the returned values from the contract call.
	response_factoryCall := new(common.Address)
	var responseSlice []interface{} = make([]interface{}, 1)
	responseSlice[0] = &response_factoryCall
	callOpts := &bind.CallOpts{}
	poolAddressConverted := common.HexToAddress(poolAddress)
	instance_uniswapV3_pool := bind.NewBoundContract(poolAddressConverted, parsedABI_uniswapv3_pool, client, client, client)

	// get the factory address of the pool - this way we can check if it is a uniswapv3 pool or some obscure clone
	err := instance_uniswapV3_pool.Call(callOpts, &responseSlice, "factory")
	if err != nil {
		return metaData, err
	}

	uniswapV3FactoryAddress := "0x1F98431c8aD98523631AE4a59f267346ea31F984"
	pancakeswapV3FactoryAddress := "0x0BFbCF9fa4f9C56B0F40a671Ad40E0805A091865"
	poolFactoryAddress := response_factoryCall.Hex()

	// Check if the poolFactoryAddress matches either Uniswap V3 or PancakeSwap V3 factory addresses
	if poolFactoryAddress != uniswapV3FactoryAddress && poolFactoryAddress != pancakeswapV3FactoryAddress {
		err = WrongFactoryAddressError{Address: poolAddress}
		return metaData, err
	}

	var multiCallAddress common.Address
	var instance_multicall *bind.BoundContract

	// Depending on the factory address, set the multiCallAddress and instance accordingly
	if poolFactoryAddress == uniswapV3FactoryAddress {
		multiCallAddress = common.HexToAddress("0x6560CEe7DC9C8498C3Fc81e214A99EE73E818870")
		instance_multicall = bind.NewBoundContract(multiCallAddress, parsedABI_uniswapv3_multicall, client, client, client)
	} else if poolFactoryAddress == pancakeswapV3FactoryAddress {
		// Assuming you have the address and ABI for PancakeSwap V3 multicall
		multiCallAddress = common.HexToAddress("0xdF27134C742bc24e0b3a54dc7c2Fd55033C219E0")
		instance_multicall = bind.NewBoundContract(multiCallAddress, parsedABI_pancakeswapv3_multicall, client, client, client)
	}

	// Now you can use instance_multicall for further operations

	var response []interface{}
	getNAdjacentTickWordsInBothDirections := uint16(20)
	err = instance_multicall.Call(callOpts, &response, "getExchangePriceInputData", poolAddressConverted, getNAdjacentTickWordsInBothDirections)
	if err != nil {
		log.Info("getBalanceMetaData_UniswapV3: Failed to retrieve value of variable:", err)
		return metaData, err
	}

	if len(response) == 0 {
		err = fmt.Errorf("response is empty")
		return metaData, err
	}

	// Marshal the first item in the response into JSON bytes
	var contractResponse ContractResponse
	bytes, err := json.Marshal(response[0])
	if err != nil {
		log.Info("getBalanceMetaData_UniswapV3: failed to marshal response[0]:", err)
		return metaData, err
	}

	// Unmarshal the JSON bytes into a ContractResponse struct
	err = json.Unmarshal(bytes, &contractResponse)
	if err != nil {
		log.Info("getBalanceMetaData_UniswapV3: failed to unmarshal into ContractResponse:", err)
		return metaData, err
	}

	metaData.SqrtPriceX96 = contractResponse.SqrtPriceX96
	metaData.Liquidity = contractResponse.Liquidity

	for _, tickData := range contractResponse.TickData {
		ticks := Ticks{
			Tick:           tickData.Tick,
			LiquidityNet:   tickData.LiquidityNet,
			LiquidityGross: tickData.LiquidityGross,
		}
		metaData.Ticks = append(metaData.Ticks, ticks)
	}

	return metaData, nil
}

//  END UNISWAPV3 MULTICALL

// start curve
// modified version that returns the metaData as an interface because we have different pool types

type MetaData_CurveV2 struct {
	Balances_wei []*big.Int
	PriceScales  []*big.Int
	D            *big.Int
}

func getBalanceMetaData_Curve(poolAddress string) (interface{}, error) {
	// find out the pool type
	poolType, err := GetPoolType_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	switch poolType {
	case "v1":
		return GetPoolBalancesWei_Curve(poolAddress)
	case "v2":
		return getBalanceMetaData_v2_Curve(poolAddress)
	case "v2_2Tokens":
		return getBalanceMetaData_v2_2Tokens_Curve(poolAddress)
	case "metaEth":
		return GetPoolBalancesWei_Curve(poolAddress)
	case "metaStable":
		return getBalanceMetaData_metaStable_Curve(poolAddress)
	default:
		return nil, fmt.Errorf("pool type not found for curve pool: %s", poolAddress)
	}
}

func getBalanceMetaData_v2_Curve(poolAddress string) (*MetaData_CurveV2, error) {
	balances_wei, err := GetPoolBalancesWei_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	priceScales, err := GetPriceScalesV2_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	d, err := Get_D_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	metaData := &MetaData_CurveV2{
		Balances_wei: balances_wei,
		PriceScales:  priceScales,
		D:            d,
	}

	return metaData, nil
}

type MetaData_CurveV2_2Tokens struct {
	Balances_wei []*big.Int
	PriceScale   *big.Int
	D            *big.Int
}

func getBalanceMetaData_v2_2Tokens_Curve(poolAddress string) (*MetaData_CurveV2_2Tokens, error) {
	balances_wei, err := GetPoolBalancesWei_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	priceScale, err := GetPriceScaleV2_2Tokens_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	d, err := Get_D_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	metaData := &MetaData_CurveV2_2Tokens{
		Balances_wei: balances_wei,
		PriceScale:   priceScale,
		D:            d,
	}

	return metaData, nil
}

type MetaData_CurveV2_MetaStable struct {
	Balances_MetaPool_wei  []*big.Int
	Balances_BasePool_wei  []*big.Int
	LPTokenSupply_BasePool *big.Int
}

func getBalanceMetaData_metaStable_Curve(poolAddress string) (*MetaData_CurveV2_MetaStable, error) {
	basePoolAddress, err := GetBasePoolAddress_MetaStable_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	balances_MetaPool_wei, err := GetPoolBalancesWei_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	balances_BasePool_wei, err := GetPoolBalancesWei_Curve(basePoolAddress)
	if err != nil {
		return nil, err
	}

	lpTokenSupply_BasePool, err := GetLPTokenSupply_BasePool_MetaStable_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	metaData := &MetaData_CurveV2_MetaStable{
		Balances_MetaPool_wei:  balances_MetaPool_wei,
		Balances_BasePool_wei:  balances_BasePool_wei,
		LPTokenSupply_BasePool: lpTokenSupply_BasePool,
	}

	return metaData, nil
}

func GetPoolBalancesWei_Curve(poolAddress string) ([]*big.Int, error) {
	tokens, _, err := GetTokensAndDecimals_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{}
	poolAddressConverted := common.HexToAddress(poolAddress)
	balances_wei := make([]*big.Int, len(tokens))

	for i, token := range tokens {
		if token == "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee" {
			// This is Ether, get the balance from the pool directly
			balance, err := client.BalanceAt(context.Background(), poolAddressConverted, nil)
			if err != nil {
				return nil, err
			}
			balances_wei[i] = balance
		} else {
			// This is a token, get the balance from the token contract
			contractAddress := common.HexToAddress(token)
			instance_ERC20 := bind.NewBoundContract(contractAddress, parsedABI_ERC20, client, client, client)

			result := []interface{}{new(*big.Int)}
			err = instance_ERC20.Call(callOpts, &result, "balanceOf", poolAddressConverted)
			if err != nil {
				return nil, err
			}
			balances_wei[i] = *result[0].(**big.Int)
		}
	}

	// No need to convert to Ether units, just return the balances in Wei
	return balances_wei, nil
}

func GetPriceScalesV2_Curve(poolAddress string) ([]*big.Int, error) {
	tokens, _, err := GetTokensAndDecimals_Curve(poolAddress)
	if err != nil {
		return nil, err
	}

	callOpts := &bind.CallOpts{}
	poolAddressConverted := common.HexToAddress(poolAddress)
	priceScales := make([]*big.Int, len(tokens)-1)

	for i := 0; i < len(tokens)-1; i++ {
		instancePool := bind.NewBoundContract(poolAddressConverted, parsedABI_Curve_V2Pool, client, client, client)

		result := []interface{}{new(*big.Int)}
		bigI := big.NewInt(int64(i))
		err = instancePool.Call(callOpts, &result, "price_scale", bigI)
		if err != nil {
			return nil, err
		}
		priceScales[i] = *result[0].(**big.Int)
	}

	return priceScales, nil
}

func GetPriceScaleV2_2Tokens_Curve(poolAddress string) (*big.Int, error) {
	instancePool := bind.NewBoundContract(common.HexToAddress(poolAddress), parsedABI_Curve_V2Pool_2Tokens, client, client, client)

	callOpts := &bind.CallOpts{}
	result := []interface{}{new(*big.Int)}
	err := instancePool.Call(callOpts, &result, "price_scale")
	if err != nil {
		return nil, err
	}
	return *result[0].(**big.Int), nil
}

func Get_D_Curve(poolAddress string) (*big.Int, error) {
	poolAddressConverted := common.HexToAddress(poolAddress)
	instancePool := bind.NewBoundContract(poolAddressConverted, parsedABI_Curve_V2Pool, client, client, client)

	callOpts := &bind.CallOpts{}
	result := []interface{}{new(*big.Int)}
	err := instancePool.Call(callOpts, &result, "D")
	if err != nil {
		return nil, err
	}
	return *result[0].(**big.Int), nil
}

func GetTokensAndDecimals_Curve(exchange string) ([]string, []int, error) {
	var data map[string]map[string]interface{}
	err := json.Unmarshal([]byte(Cache_Curve), &data)
	if err != nil {
		return nil, nil, err
	}

	exchangeData, ok := data[exchange]
	if !ok {
		return nil, nil, fmt.Errorf("exchange not found")
	}

	tokensInterface, ok := exchangeData["tokens"].([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("tokens not found or not an array")
	}

	tokens := make([]string, len(tokensInterface))
	for i, v := range tokensInterface {
		tokens[i] = v.(string)
	}

	decimalsInterface, ok := exchangeData["decimals"].([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("decimals not found or not an array")
	}

	decimals := make([]int, len(decimalsInterface))
	for i, v := range decimalsInterface {
		decimals[i] = int(v.(float64))
	}

	return tokens, decimals, nil
}

func GetPoolType_Curve(exchange string) (string, error) {
	var data map[string]map[string]interface{}
	err := json.Unmarshal([]byte(Cache_Curve), &data)
	if err != nil {
		return "", err
	}

	exchangeData, ok := data[exchange]
	if !ok {
		return "", fmt.Errorf("exchange not found")
	}

	poolType, ok := exchangeData["poolType"].(string)
	if !ok {
		return "", fmt.Errorf("type not found or not a string")
	}

	return poolType, nil
}

func GetBasePoolAddress_MetaStable_Curve(metaPoolAddress string) (string, error) {
	var data map[string]map[string]interface{}
	err := json.Unmarshal([]byte(Cache_Curve), &data)
	if err != nil {
		return "", err
	}

	metaPoolData, ok := data[metaPoolAddress]
	if !ok {
		return "", fmt.Errorf("metaPool not found")
	}

	basePool, ok := metaPoolData["basePool"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("basePool not found or not a map")
	}

	basePoolAddress, ok := basePool["exchange"].(string)
	if !ok {
		return "", fmt.Errorf("exchange not found or not a string")
	}

	return basePoolAddress, nil
}

func GetLPTokenSupply_BasePool_MetaStable_Curve(poolAddress string) (*big.Int, error) {
	tokens, _, err := GetTokensAndDecimals_Curve(poolAddress)
	if err != nil {
		return nil, err
	}
	//  call totalSupply on the last token in the tokens array
	contractAddress := common.HexToAddress(tokens[len(tokens)-1])
	instance_LPToken := bind.NewBoundContract(contractAddress, parsedABI_Curve_LPToken, client, client, client)

	callOpts := &bind.CallOpts{}
	result := []interface{}{new(*big.Int)}
	err = instance_LPToken.Call(callOpts, &result, "totalSupply")
	if err != nil {
		return nil, err
	}
	return *result[0].(**big.Int), nil
}

// create a function that returns all curve pools in a list
// use the curve cache to get the pools
// pools are called exchange there
func GetAllPools_Curve() ([]string, error) {
	var pools []string

	var data map[string]map[string]interface{}
	err := json.Unmarshal([]byte(Cache_Curve), &data)
	if err != nil {
		return nil, err
	}

	for exchange := range data {
		pools = append(pools, exchange)
	}

	return pools, nil
}

// END CURVE

// TODO nick-smc i think i need to improve logging here
func getBalanceMetaData_UniswapV2(poolAddress string) ([]float64, error) {
	var metaData []float64

	var contractAddress common.Address = common.HexToAddress(poolAddress)
	instance_uniswapV2 := bind.NewBoundContract(contractAddress, parsedABI_uniswapv2, client, client, client)

	// TODO nick-smc you can check the factory of the pool to make sure it is a uniswapv2 pool (like in the uniswapv3 function)
	// TODO nick-smc that way we can even find out if it is a uniswapv2 or sushiswap pool etc.
	// --> get the factory address from the router contract and compare it to the factory address of the pool.
	// you want to init all the factory contracts in the init function and then use them here
	// you can then return the right exchange name and override it outside of the function

	// get reserves in wei
	var reserves []interface{}
	callOpts := &bind.CallOpts{}
	err := instance_uniswapV2.Call(callOpts, &reserves, "getReserves")
	if err != nil {
		fmt.Printf("Failed to retrieve value of variable: %v", err)
		return metaData, err
	}

	// get token0 address
	var token0Address []interface{}
	err = instance_uniswapV2.Call(callOpts, &token0Address, "token0")
	if err != nil {
		fmt.Printf("Failed to retrieve value of variable: %v", err)
		return metaData, err
	}

	// get token1 address
	var token1Address []interface{}
	err = instance_uniswapV2.Call(callOpts, &token1Address, "token1")
	if err != nil {
		fmt.Printf("Failed to retrieve value of variable: %v", err)
		return metaData, err
	}

	// get token0 decimals
	var token0Decimals []interface{}
	contractAddress = token0Address[0].(common.Address)
	instance_token0 := bind.NewBoundContract(contractAddress, parsedABI_uniswapv2, client, client, client)
	err = instance_token0.Call(callOpts, &token0Decimals, "decimals")
	if err != nil {
		fmt.Printf("Failed to retrieve value of variable: %v", err)
		return metaData, err
	}

	// get token1 decimals
	var token1Decimals []interface{}
	contractAddress = token1Address[0].(common.Address)
	instance_token1 := bind.NewBoundContract(contractAddress, parsedABI_uniswapv2, client, client, client)
	err = instance_token1.Call(callOpts, &token1Decimals, "decimals")
	if err != nil {
		fmt.Printf("Failed to retrieve value of variable: %v", err)
		return metaData, err
	}

	// convert reserves that are in wei units to ether units using the decimals
	reserves0_bigInt, ok := reserves[0].(*big.Int)
	if !ok {
		fmt.Printf("Failed to assert type: %v", err)
		return metaData, fmt.Errorf("failed to assert type: %v", err)
	}
	token0Reserves, err := ConvertWeiUnitsToEtherUnits_UsingTokenAddress(reserves0_bigInt, token0Address[0].(common.Address).Hex())
	if err != nil {
		fmt.Printf("Failed to convert to ether units: %v", err)
		return metaData, err
	}
	reserves1_bigInt, ok := reserves[1].(*big.Int)
	if !ok {
		fmt.Printf("Failed to assert type: %v", err)
		return metaData, fmt.Errorf("failed to assert type: %v", err)
	}
	token1Reserves, err := ConvertWeiUnitsToEtherUnits_UsingTokenAddress(reserves1_bigInt, token1Address[0].(common.Address).Hex())
	if err != nil {
		fmt.Printf("Failed to convert to ether units: %v", err)
		return metaData, err
	}

	metaData = append(metaData, token0Reserves)
	metaData = append(metaData, token1Reserves)

	return metaData, nil
}

