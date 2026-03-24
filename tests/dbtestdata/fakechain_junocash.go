package dbtestdata

import "github.com/trezor/blockbook/bchain"

type fakeJunoCashBlockChain struct {
	*fakeBlockChain
	coinName   string
	network    string
	testnet    bool
	totalCoins string
}

func NewFakeJunoCashBlockChain(parser bchain.BlockChainParser, totalCoins string) (bchain.BlockChain, error) {
	return &fakeJunoCashBlockChain{
		fakeBlockChain: &fakeBlockChain{&bchain.BaseChain{Parser: parser}},
		coinName:       "Junocash",
		network:        "main",
		totalCoins:     totalCoins,
	}, nil
}

func NewFakeJunoCashTestnetBlockChain(parser bchain.BlockChainParser, totalCoins string) (bchain.BlockChain, error) {
	return &fakeJunoCashBlockChain{
		fakeBlockChain: &fakeBlockChain{&bchain.BaseChain{Parser: parser}},
		coinName:       "Junocash Testnet",
		network:        "test",
		testnet:        true,
		totalCoins:     totalCoins,
	}, nil
}

func (c *fakeJunoCashBlockChain) IsTestnet() bool {
	return c.testnet
}

func (c *fakeJunoCashBlockChain) GetNetworkName() string {
	return c.network
}

func (c *fakeJunoCashBlockChain) GetCoinName() string {
	return c.coinName
}

func (c *fakeJunoCashBlockChain) GetSubversion() string {
	return "/JunoCash:0.0.1/"
}

func (c *fakeJunoCashBlockChain) GetChainInfo() (v *bchain.ChainInfo, err error) {
	return &bchain.ChainInfo{
		Chain:         c.GetNetworkName(),
		Blocks:        2,
		Headers:       2,
		Bestblockhash: GetTestBitcoinTypeBlock2(c.Parser).BlockHeader.Hash,
		TotalCoins:    c.totalCoins,
		Version:       "001001",
		Subversion:    c.GetSubversion(),
	}, nil
}
