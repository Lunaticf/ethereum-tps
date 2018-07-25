package benchmark

import (
	"context"
	"crypto/ecdsa"
	crand "crypto/rand"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/golang/glog"
	"strings"
)

var RetryCount = 30

type ResultData struct {
	StartTime      time.Time
	StartTimeP     time.Time
	FinishedTx     int64
	FinishedTxP    int64
	MaxTPS         float64
	MaxPendingTx   int64
	MaxWaitingTime float64
	PendingTx      int64
	Locker         sync.Mutex
}

type Benchmark struct {
	JsonrpcEndpoint string
	MainKey         string
	BalanceLimit    int64
	GasLmit         int64
	GasPrice        int64
	ResultData      *ResultData
}

type BenchmarkTwo struct {
	JsonrpcEndpoint string
	MainKey         string
	BalanceLimit    int64
	GasLmit         int64
	GasPrice        int64
	ResultData      *ResultData
}

func NewBenchmark(jsonrpcEndpoint, mainKey string, balanceLimit, gasLimit, gasPrice int64) *Benchmark {
	return &Benchmark{
		JsonrpcEndpoint: jsonrpcEndpoint,
		MainKey:         mainKey,
		BalanceLimit:    balanceLimit,
		GasLmit:         gasLimit,
		GasPrice:        gasPrice,
		ResultData: &ResultData{
			FinishedTx:  0,
			FinishedTxP: 0,
			PendingTx:   0,
		},
	}
}

func (b *Benchmark) Run(index int64) {
	switch index {
	case 1:
		b.runOne()
	case 2:
		b.runTwo()
	}
}

func (b *Benchmark) runOne() {
	// open grpc connection
	conn, err := ethclient.Dial(b.JsonrpcEndpoint)
	if err != nil {
		glog.Errorf("Failed to dial, url: %s, err: %s\n", b.JsonrpcEndpoint, err)
		return
	}
	networkId, err := conn.NetworkID(context.Background())
	if err != nil {
		glog.Errorf("Failed to get networkId, err: %s\n", err)
		return
	}
	glog.V(1).Infof("Connect to Ethereum success, networkId: %d\n", networkId)

	// gen key
	keyChannel := make(chan *ecdsa.PrivateKey, 1000)
	go func() {
		for {
			privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), crand.Reader)
			if err != nil {
				glog.Errorf("ecdsa.GenerateKey failed: %v\n", err)
			}
			keyChannel <- privateKeyECDSA
		}
	}()

	// start test
	b.ResultData.StartTime = time.Now()
	b.ResultData.StartTimeP = time.Now()
	go func() {
		fromKey, _ := crypto.HexToECDSA(b.MainKey)
		b.distributeEthereum(conn, fromKey, keyChannel)
	}()
	select {}
}

func (b *Benchmark) distributeEthereum(conn *ethclient.Client, fromKey *ecdsa.PrivateKey, keyChannel chan *ecdsa.PrivateKey) {

	// check balance
	fromAddress := crypto.PubkeyToAddress(fromKey.PublicKey)
	ctx := context.Background()
	var balance *big.Int
	var err error
	for i := 0; i < RetryCount; i++ {
		balance, err = conn.BalanceAt(context.Background(), fromAddress, nil)
		if err != nil && strings.Contains(err.Error(), "cannot assign requested address") {
			glog.V(4).Infof("conn.BalanceAt, connect: cannot assign requested address, retry in 1s, count=%d\n", i)
			time.Sleep(time.Duration(1) * time.Second)
			continue
		}
		break
	}
	limit := big.NewInt(b.BalanceLimit) // 10^16 = 0.01eth
	if balance.Cmp(limit) < 0 {
		glog.V(4).Infof("Account %s reach balance limit.\n", fromAddress.Hex())
		return
	}

	// calculate distribute amount
	count := 2
	limit.Neg(limit)
	balance.Add(balance, limit)
	balance.Div(balance, big.NewInt(int64(count)))

	// distribute eth
	for i := 0; i < count; i++ {
		var signedTx *types.Transaction
		// SendTransaction
		toKey := <-keyChannel
		toAddress := crypto.PubkeyToAddress(toKey.PublicKey)
		for i := 0; i < RetryCount; i++ {
			nonce, _ := conn.NonceAt(ctx, fromAddress, nil)
			tx := types.NewTransaction(nonce, toAddress, balance, uint64(b.GasLmit), big.NewInt(b.GasPrice), nil)
			signedTx, err = types.SignTx(tx, types.HomesteadSigner{}, fromKey)
			err = conn.SendTransaction(ctx, signedTx)
			if err != nil && strings.Contains(err.Error(), "cannot assign requested address") {
				glog.V(4).Infof("connect: cannot assign requested address, retry in 1s, count=%d\n", i)
				time.Sleep(time.Duration(1) * time.Second)
				continue
			}
			break
		}
		if err != nil {
			glog.Errorf("SendTransaction failed after retry: %s\n", err)
			return
		}

		// update data
		b.ResultData.Locker.Lock()
		var tpsP float64
		if time.Now().Sub(b.ResultData.StartTimeP).Seconds() > 60 { // update period data if more than 1min
			tpsP = float64(b.ResultData.FinishedTxP) / time.Now().Sub(b.ResultData.StartTimeP).Seconds()
			if tpsP > b.ResultData.MaxTPS {
				b.ResultData.MaxTPS = tpsP
			}

			// reset period
			b.ResultData.StartTimeP = time.Now()
			b.ResultData.FinishedTxP = 0
		}
		tps := float64(b.ResultData.FinishedTx) / time.Now().Sub(b.ResultData.StartTime).Seconds()
		if tps > b.ResultData.MaxTPS {
			b.ResultData.MaxTPS = tps
		}
		glog.V(1).Infof("MaxTPS: %f; MaxPendingTx: %d; MaxWaitTime: %f; Period TPS: %f; Average TPS: %f; FinishedTx: %d; PendingTx: %d;\n",
			b.ResultData.MaxTPS,
			b.ResultData.MaxPendingTx,
			b.ResultData.MaxWaitingTime,
			tpsP,
			tps,
			b.ResultData.FinishedTx,
			b.ResultData.PendingTx)
		b.ResultData.PendingTx = b.ResultData.PendingTx + 1
		if b.ResultData.PendingTx > b.ResultData.MaxPendingTx {
			b.ResultData.MaxPendingTx = b.ResultData.PendingTx
		}
		b.ResultData.Locker.Unlock()

		// begin of tx
		waitStartTime := time.Now()
		_, err = bind.WaitMined(ctx, conn, signedTx)
		if err != nil {
			glog.Errorf("tx mining error:%v\n", err)
			b.ResultData.Locker.Lock()
			b.ResultData.PendingTx = b.ResultData.PendingTx - 1
			b.ResultData.Locker.Unlock()
			return
		}
		waitTime := time.Now().Sub(waitStartTime).Seconds()
		// end of tx

		// update data
		b.ResultData.Locker.Lock()
		if waitTime > b.ResultData.MaxWaitingTime {
			b.ResultData.MaxWaitingTime = waitTime
		}
		b.ResultData.PendingTx = b.ResultData.PendingTx - 1
		b.ResultData.FinishedTx = b.ResultData.FinishedTx + 1
		b.ResultData.FinishedTxP = b.ResultData.FinishedTxP + 1
		b.ResultData.Locker.Unlock()

		// cal another
		go func() {
			b.distributeEthereum(conn, toKey, keyChannel)
		}()
	}
}

func (b *Benchmark) runTwo()  {
	// open grpc connection
	conn, err := ethclient.Dial(b.JsonrpcEndpoint)
	if err != nil {
		glog.Errorf("Failed to dial, url: %s, err: %s\n", b.JsonrpcEndpoint, err)
		return
	}
	networkId, err := conn.NetworkID(context.Background())
	if err != nil {
		glog.Errorf("Failed to get networkId, err: %s\n", err)
		return
	}
	glog.V(1).Infof("Connect to Ethereum success, networkId: %d\n", networkId)

	ctx := context.Background()
	// from
	fromKey, _ := crypto.HexToECDSA(b.MainKey)
	fromAddress := crypto.PubkeyToAddress(fromKey.PublicKey)
	// to
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), crand.Reader)
	if err != nil {
		glog.Errorf("ecdsa.GenerateKey failed: %v\n", err)
	}
	toAddress := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)
	// start nonce
	nonce, _ := conn.NonceAt(ctx, fromAddress, nil)

	for {
		tx := types.NewTransaction(nonce, toAddress, big.NewInt(1), uint64(b.GasLmit), big.NewInt(b.GasPrice), nil)
		signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, fromKey)
		if err != nil {
			glog.Errorf("Failed to SignTx, err: %s\n", err)
		}
		// txData, _ := signedTx.MarshalJSON()
		// glog.V(1).Infof("TX: %s\n", string(txData))
		err = conn.SendTransaction(ctx, signedTx)
		nonce++
	}

}