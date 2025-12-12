package junocash

import (
	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
)

const (
	// MainnetMagic is mainnet network constant
	MainnetMagic wire.BitcoinNet = 0x02070cb5

	// TestnetMagic is testnet network constant
	TestnetMagic wire.BitcoinNet = 0x6ce123a7

	// RegtestMagic is regtest network constant
	RegtestMagic wire.BitcoinNet = 0xf6211d81
)

var (
	// MainNetParams are parser parameters for mainnet
	MainNetParams chaincfg.Params
	// TestNetParams are parser parameters for testnet
	TestNetParams chaincfg.Params
	// RegtestParams are parser parameters for regtest
	RegtestParams chaincfg.Params
)

func init() {
	// Mainnet configuration
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic

	// Address encoding magics (2-bytes prefix)
	MainNetParams.AddressMagicLen = 2
	MainNetParams.PubKeyHashAddrID = []byte{0x1C, 0xB8} // base58: t1
	MainNetParams.ScriptHashAddrID = []byte{0x1C, 0xBD} // base58: t3

	// Testnet configuration
	TestNetParams = chaincfg.TestNet3Params
	TestNetParams.Net = TestnetMagic

	// Address encoding magics (2-byte prefixes)
	TestNetParams.AddressMagicLen = 2
	TestNetParams.PubKeyHashAddrID = []byte{0x1D, 0x25} // base58: tm
	TestNetParams.ScriptHashAddrID = []byte{0x1C, 0xBA} // base58: t2

	// Regtest configuration
	RegtestParams = chaincfg.RegressionNetParams
	RegtestParams.Net = RegtestMagic
	// Regtest uses same address prefix as testnet
	RegtestParams.AddressMagicLen = 2
	RegtestParams.PubKeyHashAddrID = []byte{0x1D, 0x25} // base58: tm
	RegtestParams.ScriptHashAddrID = []byte{0x1C, 0xBA} // base58: t2
}

// JunoCashParser handle
type JunoCashParser struct {
	*btc.BitcoinLikeParser
	baseparser *bchain.BaseParser
}

// NewJunoCashParser returns new JunoCashParser instance
func NewJunoCashParser(params *chaincfg.Params, c *btc.Configuration) *JunoCashParser {
	return &JunoCashParser{
		BitcoinLikeParser: btc.NewBitcoinLikeParser(params, c),
		baseparser:        &bchain.BaseParser{},
	}
}

// GetChainParams contains network parameters for the main JunoCash network,
// the regression test JunoCash network, the test JunoCash network
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestNetParams)
		}
		if err == nil {
			err = chaincfg.Register(&RegtestParams)
		}
		if err != nil {
			panic(err)
		}
	}
	switch chain {
	case "test":
		return &TestNetParams
	case "regtest":
		return &RegtestParams
	default:
		return &MainNetParams
	}
}

// PackTx packs transaction to byte array using protobuf
func (p *JunoCashParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
	return p.baseparser.PackTx(tx, height, blockTime)
}

// UnpackTx unpacks transaction from protobuf byte array
func (p *JunoCashParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
	return p.baseparser.UnpackTx(buf)
}
