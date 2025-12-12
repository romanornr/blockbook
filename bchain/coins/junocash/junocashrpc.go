package junocash

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"reflect"
	"regexp"

	"github.com/golang/glog"
	"github.com/juju/errors"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
	"github.com/trezor/blockbook/common"
)

// extractVersion extracts the version string (e.g., "v0.9.7-eaaeb5a853c") from the full daemon output
func extractVersion(fullVersion string) string {
	// Match version pattern: 'v' followed by digits, dots, and optional suffix (e.g., v0.9.7-eaaeb5a853c)
	re := regexp.MustCompile(`v\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?`)
	match := re.FindString(fullVersion)
	if match != "" {
		return match
	}
	return fullVersion // fallback to original if no match
}

// JunoCashRPC is an interface to JSON-RPC junocashd service
type JunoCashRPC struct {
	*btc.BitcoinRPC
}

// ResGetBlockChainInfo is a response to GetChainInfo request
type ResGetBlockChainInfo struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Chain         string            `json:"chain"`
		Blocks        int               `json:"blocks"`
		Headers       int               `json:"headers"`
		Bestblockhash string            `json:"bestblockhash"`
		Difficulty    common.JSONNumber `json:"difficulty"`
		Pruned        bool              `json:"pruned"`
		SizeOnDisk    int64             `json:"size_on_disk"`
		Consensus     struct {
			Chaintip  string `json:"chaintip"`
			Nextblock string `json:"nextblock"`
		} `json:"consensus"`
	} `json:"result"`
}

// NewJunoCashRPC returns new JunoCashRPC instance
func NewJunoCashRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}
	j := &JunoCashRPC{
		BitcoinRPC: b.(*btc.BitcoinRPC),
	}
	j.RPCMarshaler = JSONMarshalerV1JunoCash{}
	j.ChainConfig.SupportsEstimateSmartFee = false
	return j, nil
}

// Initialize initializes JunoCashRPC instance
func (j *JunoCashRPC) Initialize() error {
	ci, err := j.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	params := GetChainParams(chainName)

	j.Parser = NewJunoCashParser(params, j.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		j.Testnet = false
		j.Network = "livenet"
	} else {
		j.Testnet = true
		j.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// GetChainInfo returns info about the blockchain
func (j *JunoCashRPC) GetChainInfo() (*bchain.ChainInfo, error) {
	chainInfo := ResGetBlockChainInfo{}
	err := j.Call(&btc.CmdGetBlockChainInfo{Method: "getblockchaininfo"}, &chainInfo)
	if err != nil {
		return nil, err
	}
	if chainInfo.Error != nil {
		return nil, chainInfo.Error
	}

	// networkinfo not fully supported by junocashd
	networkInfo := btc.ResGetNetworkInfo{}

	junocashd := "junocashd"
	cmd := exec.Command("/opt/coins/nodes/junocash/bin/junocashd", "--version")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err == nil {
		junocashd = out.String()
	}

	return &bchain.ChainInfo{
		Bestblockhash:   chainInfo.Result.Bestblockhash,
		Blocks:          chainInfo.Result.Blocks,
		Chain:           chainInfo.Result.Chain,
		Difficulty:      string(chainInfo.Result.Difficulty),
		Headers:         chainInfo.Result.Headers,
		SizeOnDisk:      chainInfo.Result.SizeOnDisk,
		Version:         extractVersion(junocashd),
		Subversion:      string(networkInfo.Result.Subversion),
		ProtocolVersion: string(networkInfo.Result.ProtocolVersion),
		Timeoffset:      networkInfo.Result.Timeoffset,
		Consensus:       chainInfo.Result.Consensus,
		Warnings:        networkInfo.Result.Warnings,
	}, nil
}

// GetBlock returns block with given hash
func (j *JunoCashRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	type rpcBlock struct {
		bchain.BlockHeader
		Txs []bchain.Tx `json:"tx"`
	}
	type rpcBlockTxids struct {
		Txids []string `json:"tx"`
	}
	type resGetBlockV1 struct {
		Error  *bchain.RPCError `json:"error"`
		Result rpcBlockTxids    `json:"result"`
	}
	type resGetBlockV2 struct {
		Error  *bchain.RPCError `json:"error"`
		Result rpcBlock         `json:"result"`
	}

	var err error
	if hash == "" && height > 0 {
		hash, err = j.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}

	var rawResponse json.RawMessage
	resV2 := resGetBlockV2{}
	req := btc.CmdGetBlock{Method: "getblock"}
	req.Params.BlockHash = hash
	req.Params.Verbosity = 2
	err = j.Call(&req, &rawResponse)
	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	// IMPORTANT: Junocash/Zcash uses "valueZat" instead of "valueSat"
	rawResponse = bytes.ReplaceAll(rawResponse, []byte(`"valueZat"`), []byte(`"valueSat"`))
	err = json.Unmarshal(rawResponse, &resV2)
	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}

	if resV2.Error != nil {
		return nil, errors.Annotatef(resV2.Error, "hash %v", hash)
	}
	block := &bchain.Block{
		BlockHeader: resV2.Result.BlockHeader,
		Txs:         resV2.Result.Txs,
	}

	// transactions fetched in block with verbosity 2 do not contain txids
	resV1 := resGetBlockV1{}
	req.Params.Verbosity = 1
	err = j.Call(&req, &resV1)
	if err != nil {
		return nil, errors.Annotatef(err, "hash %v", hash)
	}
	if resV1.Error != nil {
		return nil, errors.Annotatef(resV1.Error, "hash %v", hash)
	}
	for i := range resV1.Result.Txids {
		block.Txs[i].Txid = resV1.Result.Txids[i]
	}
	return block, nil
}

// GetTransaction returns a transaction by the transaction ID
func (j *JunoCashRPC) GetTransaction(txid string) (*bchain.Tx, error) {
	r, err := j.getRawTransaction(txid)
	if err != nil {
		return nil, err
	}
	// IMPORTANT: Junocash/Zcash uses "valueZat" instead of "valueSat"
	r = bytes.ReplaceAll(r, []byte(`"valueZat"`), []byte(`"valueSat"`))
	tx, err := j.Parser.ParseTxFromJson(r)
	if err != nil {
		return nil, errors.Annotatef(err, "txid %v", txid)
	}
	tx.Blocktime = tx.Time
	tx.Txid = txid
	tx.CoinSpecificData = r
	return tx, nil
}

// getRawTransaction returns json as returned by backend
func (j *JunoCashRPC) getRawTransaction(txid string) (json.RawMessage, error) {
	glog.V(1).Info("rpc: getrawtransaction ", txid)

	res := btc.ResGetRawTransaction{}
	req := btc.CmdGetRawTransaction{Method: "getrawtransaction"}
	req.Params.Txid = txid
	req.Params.Verbose = true
	err := j.Call(&req, &res)

	if err != nil {
		return nil, errors.Annotatef(err, "txid %v", txid)
	}
	if res.Error != nil {
		if btc.IsMissingTx(res.Error) {
			return nil, bchain.ErrTxNotFound
		}
		return nil, errors.Annotatef(res.Error, "txid %v", txid)
	}
	return res.Result, nil
}

// GetTransactionForMempool returns a transaction by the transaction ID
func (j *JunoCashRPC) GetTransactionForMempool(txid string) (*bchain.Tx, error) {
	return j.GetTransaction(txid)
}

// GetMempoolEntry returns mempool data for given transaction
func (j *JunoCashRPC) GetMempoolEntry(txid string) (*bchain.MempoolEntry, error) {
	return nil, errors.New("GetMempoolEntry: not implemented")
}

// GetBlockRaw is not supported
func (j *JunoCashRPC) GetBlockRaw(hash string) (string, error) {
	return "", errors.New("GetBlockRaw: not supported")
}

// JSONMarshalerV1JunoCash is used for marshalling requests to junocashd
type JSONMarshalerV1JunoCash struct{}

type cmdUntypedParams struct {
	Method string        `json:"method"`
	Id     string        `json:"id"`
	Params []interface{} `json:"params"`
}

// Marshal converts Go command structs to JSON-RPC format with positional parameters.
// Zcash/Junocash RPC expects params as an array ["arg1", "arg2"], not as an object {"key": "value"}.
// This marshaler transforms typed Go structs into the array-based format junocashd expects.
func (JSONMarshalerV1JunoCash) Marshal(v interface{}) ([]byte, error) {
	u := cmdUntypedParams{}

	// Type switch handles known command types explicitly to ensure correct parameter ordering.
	// The order of parameters in the array MUST match what junocashd expects.
	switch v := v.(type) {
	case *btc.CmdGetBlock:
		// getblock expects: params[0] = blockhash (string), params[1] = verbosity (int)
		u.Method = v.Method
		u.Params = append(u.Params, v.Params.BlockHash) // Position 0: block hash
		u.Params = append(u.Params, v.Params.Verbosity) // Position 1: verbosity level (0, 1, or 2)
	case *btc.CmdGetRawTransaction:
		// getrawtransaction expects: params[0] = txid (string), params[1] = verbose (int: 0 or 1)
		// Note: junocashd expects an integer (0/1), not a boolean (true/false)
		var n int
		if v.Params.Verbose {
			n = 1 // Convert boolean true → integer 1
		}
		u.Method = v.Method
		u.Params = append(u.Params, v.Params.Txid) // Position 0: transaction ID
		u.Params = append(u.Params, n)             // Position 1: verbosity as int (0=raw hex, 1=decoded JSON)
	default:
		// Fallback: Use reflection to handle any other command types generically.
		// This extracts the Method field and converts Params struct fields to an array.
		{
			v := reflect.ValueOf(v).Elem() // Dereference pointer to get the struct

			// Extract the "Method" field (required for all RPC commands)
			f := v.FieldByName("Method")
			if !f.IsValid() || f.Kind() != reflect.String {
				return nil, btc.ErrInvalidValue
			}
			u.Method = f.String()

			// Extract the "Params" field and convert to array format
			f = v.FieldByName("Params")
			if f.IsValid() {
				var arr []interface{}
				switch f.Kind() {
				case reflect.Slice:
					// Params is already a slice - copy elements directly
					arr = make([]interface{}, f.Len())
					for i := 0; i < f.Len(); i++ {
						arr[i] = f.Index(i).Interface()
					}
				case reflect.Struct:
					// Params is a struct - convert each field to array element (in field order)
					arr = make([]interface{}, f.NumField())
					for i := 0; i < f.NumField(); i++ {
						arr[i] = f.Field(i).Interface()
					}
				default:
					return nil, btc.ErrInvalidValue
				}
				u.Params = arr
			}
		}
	}

	// Set a static request ID (junocashd doesn't use it for anything meaningful)
	u.Id = "-"

	// Ensure params is never nil (JSON-RPC requires an array, even if empty)
	if u.Params == nil {
		u.Params = make([]interface{}, 0)
	}

	// Marshal to JSON bytes for sending over HTTP
	d, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}

	return d, nil
}
