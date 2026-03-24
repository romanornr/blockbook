package junocash

import (
	"encoding/json"
	"testing"
)

func TestResGetBlockChainInfo_UnmarshalChainSupply(t *testing.T) {
	raw := []byte(`{
		"result": {
			"chain": "test",
			"blocks": 131567,
			"headers": 132750,
			"bestblockhash": "000164929677f8d2837836f81ee4caa9c4aced909b120143d153b688c81fbe70",
			"difficulty": 1.924575191709768,
			"size_on_disk": 65265928,
			"chainSupply": {
				"monitored": true,
				"chainValue": 1449793.74990004,
				"chainValueZat": 144979374990004
			},
			"consensus": {
				"chaintip": "4dec4df0",
				"nextblock": "4dec4df0"
			}
		},
		"error": null
	}`)

	var got ResGetBlockChainInfo
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got.Result.ChainSupply.ChainValue.String() != "1449793.74990004" {
		t.Fatalf("ChainSupply.ChainValue = %q, want %q", got.Result.ChainSupply.ChainValue.String(), "1449793.74990004")
	}
}
