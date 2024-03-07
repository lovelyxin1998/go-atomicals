package types

import (
	"encoding/hex"
	"log"
)

type BitworkInfo struct {
	Prefix string `json:"prefix"`
	Ext    byte   `json:"ext"`

	PrefixBytes   []byte `json:"-"`
	PrefixPartial *byte  `json:"-"`
}

type Mint_params struct {
	Id            string `json:"id"`
	FinalCopyData string `json:"finalCopyData"`
	Status        int64  `json:"status"`

	Bitworkc string `json:"bitworkc"`

	FundingUtxoTxid  string `json:"fundingUtxoTxid"`
	FundingUtxoIndex uint32 `json:"fundingUtxoIndex"`
	FundingUtxoValue int64  `json:"fundingUtxoValue"`

	P2trOutputHex string `json:"p2trOutputHex"`
	P2trAmount    int64  `json:"p2trAmount"`

	FundingOutputHex string `json:"fundingOutputHex"`
	SelfAmount       int64  `json:"selfAmount"`

	Sequence int64 `json:"sequence"`
}

type AdditionalParams struct {
	WorkerBitworkInfoCommit *BitworkInfo `json:"workerBitworkInfoCommit"`
}

func (bw *BitworkInfo) ParsePreifx() {
	var (
		prefixBytes []byte
		err         error
	)
	if len(bw.Prefix)&1 == 0 {
		prefixBytes, err = hex.DecodeString(bw.Prefix)
		if err != nil {
			log.Fatalf("hex.DecodeString(bw.Prefix[:len(bw.Prefix)-1]) failed: %v", err)
		}
	} else {
		prefixBytes, err = hex.DecodeString(bw.Prefix[:len(bw.Prefix)-1])
		if err != nil {
			log.Fatalf("hex.DecodeString(bw.Prefix[:len(bw.Prefix)-1]) failed: %v", err)
		}
		bw.PrefixPartial = new(byte)
		if bw.Prefix[len(bw.Prefix)-1] > '9' {
			*bw.PrefixPartial = byte(bw.Prefix[len(bw.Prefix)-1] - 'a' + 10)
		} else {
			*bw.PrefixPartial = byte(bw.Prefix[len(bw.Prefix)-1] - '0')
		}
	}
	bw.PrefixBytes = prefixBytes
}
