//go:build cuda
// +build cuda

package work

// #include <stdint.h>
//uint32_t scanhash_sha256d(int thr_id, unsigned char* in, unsigned int inlen, unsigned char *target, unsigned int target_len, char pp, char ext, unsigned int threads, unsigned int start_seq, unsigned int *hashes_done);
//#cgo LDFLAGS: -L. -L../../cuda -lhash
import "C"
import (
	"fmt"
	"go-atomicals/pkg/types"
	"log"

	"github.com/mindprince/gonvml"

	"time"
)

var MAX_SEQUENCE = int64(4294967295)
var globalParams types.Mint_params

func Update(input types.Mint_params) {
	globalParams = input
}

func compareStr(str1 string, str2 string) bool {
	res := str1 == str2
	return res
}

var deviceNum int

func Initialize() {
	deviceNum = 1
	// devcieNumStr := os.Getenv("CUDA_DEVICE_NUM")
	// if devcieNumStr != "" {
	// 	deviceNum = int(devcieNumStr[0] - '0')
	// }

	err := gonvml.Initialize()
	if err != nil {
		fmt.Println("初始化 NVML 失败:", err)
		return
	}
	defer gonvml.Shutdown()

	count, err := gonvml.DeviceCount()
	if err != nil {
		fmt.Println("获取显卡数量失败:", err)
		return
	}

	deviceNum = int(count)

	log.Printf("deviceNum: %v", deviceNum)
}

func Mine(input *types.Mint_params, workInfo *types.BitworkInfo, add *types.AdditionalParams, serializedTx []byte, threads uint32) {

	result := make(chan int64)

	log.Printf("开始计算 任务id: %v", input.Id)
	start := time.Now()

	var res = int64(0)
	for i := 0; i < deviceNum; i++ {
		go mine(i, input, workInfo, add, serializedTx, threads, &res, result)
	}

	for i := 0; i < deviceNum; i++ {
		<-result
	}

	input.Sequence = res
	log.Printf("结束计算 任务id: %v,计算时间：%d s", input.Id, int64(time.Since(start).Seconds()))
}

// func mine(input *Mint_params,workInfo *BitworkInfo,serializedTx []byte, threads uint32, result chan<- Mint_params) {
func mine(device_id int, input *types.Mint_params, workInfo *types.BitworkInfo, add *types.AdditionalParams, serializedTx []byte, threads uint32, res *int64, result chan<- int64) {

	hashesDone := C.uint(0)
	var (
		pp       = -1
		ext      = -1
		Sequence = uint32(0)
		//res      = int64(0)
	)
	//log.Printf(string(workInfo.PrefixBytes), len(workInfo.PrefixBytes))

	if add.WorkerBitworkInfoCommit.PrefixPartial != nil {
		pp = int(*add.WorkerBitworkInfoCommit.PrefixPartial)
	}
	if add.WorkerBitworkInfoCommit.Ext != 0 {
		ext = int(add.WorkerBitworkInfoCommit.Ext)
	}

	// num := new(big.Int)
	// num.SetString(input.Id, 16)
	// result := new(big.Int)
	// result.Mod(num, big.NewInt(20))

	// device_id := int(result.Int64())
	// log.Printf("开始计算 任务id: %v", input.Id)

	// start := time.Now()
	for {

		compareResult := compareStr(input.Id, globalParams.Id)
		if compareResult == false || input.Status != 0 {
			break
		}

		seq := C.scanhash_sha256d(
			C.int(device_id), // device id
			(*C.uchar)(&serializedTx[0]),
			C.uint(len(serializedTx)),
			(*C.uchar)(&workInfo.PrefixBytes[0]),
			C.uint(len(workInfo.PrefixBytes)),
			C.char(pp),
			C.char(ext),
			C.uint(threads),
			C.uint(Sequence),
			&hashesDone,
		)

		if *res != 0 && *res != MAX_SEQUENCE {
			break
		}

		if int64(seq) != MAX_SEQUENCE {
			*res = int64(seq)
			break
		}

		Sequence += threads * uint32(deviceNum)
	}
	result <- 1
	//log.Printf("结束计算 hashrate: %d/s", int64(float64(threads)/time.Since(start).Seconds()))
	//fmt.Println("结束计算 device_id", device_id)
	//log.Printf("结束计算 任务id: %v,计算时间：%d s", input.Id, int64(time.Since(start).Seconds()))

	//input.Sequence = res
	//result <- input
}
