//go:build cuda
// +build cuda

package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"sync"
	"sync/atomic"

	//"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"

	"bytes"
	"log"

	//"os"

	"github.com/btcsuite/btcd/wire"
	//"net/http"

	"time"

	"encoding/json"

	"net/http"

	"github.com/gorilla/websocket"

	"go-atomicals/pkg/types"
	"go-atomicals/pkg/work"
)

var (
	updated = false

	wsUrl        = "ws://144.76.71.189:29904"
	httpUrl      = "http://144.76.71.189:29903"
	globalParams = types.Mint_params{}
)
var number_of_workers int64
var mutex sync.Mutex
var lastPostTime = time.Now()
var lastWorkTime = time.Now()

func deal(input types.Mint_params) {

	// input := Mint_params{
	// 	Bitworkc: "123",

	// 	FundingUtxoTxid:  "1514187b3fa9555c599052601c9b9de1abc632e651060b1ffae27efe6ffa4381",
	// 	FundingUtxoIndex: uint32(0),
	// 	FundingUtxoValue: 300,

	// 	P2trOutputHex: "512077d3ccf2726c66bd334cdd9d490102c67986a17627fe310c603b0e5aec0d3fb7",
	// 	P2trAmount:    int64(13426),

	// 	FundingOutputHex: "512077d3ccf2726c66bd334cdd9d490102c67986a17627fe310c603b0e5aec0d3fb7",
	// 	SelfAmount:       int64(716790),
	// }

	atomic.AddInt64(&number_of_workers, 1)

	globalParams = input
	work.Update(globalParams)

	bitworkInfo := types.BitworkInfo{
		Prefix: input.Bitworkc,
	}

	add := types.AdditionalParams{
		WorkerBitworkInfoCommit: &types.BitworkInfo{
			Prefix: input.Bitworkc,
		},
	}
	add.WorkerBitworkInfoCommit.ParsePreifx()
	bitworkInfo.ParsePreifx()

	var serializedTx []byte
	SerializedTx(input, &serializedTx)
	//fmt.Println(serializedTx)

	threads := uint32(1 * 10000 * 10000)

	if input.Status == 0 {
		fmt.Println("新的work", input.Bitworkc, input.Id)

		work.Mine(&input, &bitworkInfo, &add, serializedTx, threads)
	} else {
		log.Println("无任务，睡眠")
		time.Sleep(60 * time.Second)
	}

	atomic.AddInt64(&number_of_workers, -1)

	if input.Sequence != 0 {
		postWork(input)
	} else if input.Status == 0 {
		log.Println("遍历完seq仍未算出,重新请求")
	}
}

func postWork(input types.Mint_params) {
	jsonMessage, err := json.Marshal(input)
	if err != nil {
		log.Println("序列化消息失败：", err)
		return
	}

	url := httpUrl + "/postWork"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil {
		log.Println("发送消息失败：", err)
		return
	}
	defer resp.Body.Close()

	if input.Sequence == 0 {
		lastPostTime = time.Now()
		log.Println("遍历完seq仍未算出,重新请求")
	} else {
		log.Println("消息已发送到服务器：", string(jsonMessage))
	}
}

func getParams() {
	url := httpUrl + "/getMinerParams"
	resp, err := http.Get(url)
	if err != nil {
		log.Println("发送消息失败：", err)
		return
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return
	}

	lastWorkTime = time.Now()
	// 输出响应内容
	//fmt.Println("响应内容:", string(respBody))
	go dealMessage(respBody)
}

func SerializedTx(input types.Mint_params, serializedTx *[]byte) {
	msgTx := wire.NewMsgTx(wire.TxVersion)

	fundingTxid, _ := chainhash.NewHashFromStr(input.FundingUtxoTxid)
	funding_iutput := wire.NewOutPoint(fundingTxid, input.FundingUtxoIndex)
	txIn := wire.NewTxIn(funding_iutput, nil, nil)
	txIn.Sequence = 0
	msgTx.AddTxIn(txIn)

	p2trOutput, _ := hex.DecodeString(input.P2trOutputHex)
	p2trOut := wire.NewTxOut(int64(input.P2trAmount), p2trOutput)
	msgTx.AddTxOut(p2trOut)

	selfOutput, _ := hex.DecodeString(input.FundingOutputHex)
	selfOut := wire.NewTxOut(int64(input.SelfAmount), selfOutput)
	msgTx.AddTxOut(selfOut)

	buf := bytes.NewBuffer(make([]byte, 0, msgTx.SerializeSizeStripped()))
	msgTx.SerializeNoWitness(buf)

	//fmt.Println(msgTx.TxHash())
	*serializedTx = buf.Bytes()
}

func autoGetWork() {
	for {
		if atomic.LoadInt64(&number_of_workers) < 1 {

			currentTime := time.Now()
			duration := currentTime.Sub(lastWorkTime)

			if duration > 200*time.Millisecond {
				go getParams()
				lastWorkTime = time.Now()
			}

		}

		time.Sleep(100 * time.Millisecond)
		if globalParams.Status != 0 {
			time.Sleep(50 * time.Second)
		}
	}
}

func main() {
	fmt.Println("version:", 2)
	work.Initialize()
	go autoGetWork()
	for {
		wsListen()
		time.Sleep(5 * time.Second)
	}

}

func dealMessage(message []byte) {
	var input types.Mint_params
	//fmt.Println(string(message))
	// 解析JSON数据
	err := json.Unmarshal(message, &input)
	if err != nil {
		log.Fatal(err)
		return
	}

	if input.FundingUtxoTxid != globalParams.FundingUtxoTxid || input.FundingUtxoIndex != globalParams.FundingUtxoIndex || input.Bitworkc != globalParams.Bitworkc {

		//log.Println("遍历完seq仍未算出,重新请求")
		go deal(input)
		return
	}

	if atomic.LoadInt64(&number_of_workers) >= 1 {
		//fmt.Println("work进程过多", number_of_workers)
		return
	}

	deal(input)
}

func wsListen() {
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()

	// conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsUrl, nil)
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		fmt.Println("websocket", err.Error())
		//go wsListen()
		return
	}
	defer conn.Close()

	fmt.Println("Connect ws")
	startTime := time.Now()
	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello, server!"))
	if err != nil {
		fmt.Println("Failed to send message:", err.Error())
	}

	for {
		if time.Now().Sub(startTime) > 1*time.Minute {
			fmt.Println("Reconnect")
			//go wsListen()
			return
		}
		_, message, err := conn.ReadMessage()
		//fmt.Println(string(message))
		if err != nil {
			fmt.Println("websocket", err.Error())
			//go wsListen()
			return
		}

		go dealMessage(message)
		//fmt.Println(toMintBlockNumber,userNonce)
		startTime = time.Now()

	}
}
