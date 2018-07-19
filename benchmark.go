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
)

type Benchmark struct {
	JsonrpcEndpoint string
	MainKey         string
	BalanceLimit    int64
	GasLmit         int64
	GasPrice        int64
	PendingTxLimit  int64
	StartTime       time.Time
	StartTimeP      time.Time
	FinishedTx      int64
	FinishedTxP     int64
	MaxTPS          float64
	MaxPendingTx    int64
	MaxWaitingTime  float64
	PendingTx       int64
	Locker          sync.Mutex
}

func NewBenchmark(jsonrpcEndpoint, mainKey string, balanceLimit, gasLimit, gasPrice, pendingTxLimit int64) *Benchmark {
	return &Benchmark{
		JsonrpcEndpoint: jsonrpcEndpoint,
		MainKey:         mainKey,
		BalanceLimit:    balanceLimit,
		GasLmit:         gasLimit,
		GasPrice:        gasPrice,
		PendingTxLimit:  pendingTxLimit,
		FinishedTx:      0,
		FinishedTxP:     0,
		PendingTx:       0,
	}
}

func (b *Benchmark) Run() {
	// open grpc connection
	conn, err := b.openConnection(b.JsonrpcEndpoint)
	if err != nil {
		glog.Errorf("Failed to openConnection, err: ", err)
	}

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

	fromKey, _ := crypto.HexToECDSA(b.MainKey)

	b.StartTime = time.Now()
	b.StartTimeP = time.Now()
	go func() {
		b.distributeEthereum(conn, fromKey, keyChannel)
	}()
	select {}
}

func (b *Benchmark) distributeEthereum(conn *ethclient.Client, fromKey *ecdsa.PrivateKey, keyChannel chan *ecdsa.PrivateKey) {

	fromAddress := crypto.PubkeyToAddress(fromKey.PublicKey)
	ctx := context.Background()
	balance, err := conn.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		glog.Errorf("conn.BalanceAt error:%v\n", err)
		return
	}
	limit := big.NewInt(b.BalanceLimit) // 10^16 = 0.01eth
	if balance.Cmp(limit) < 0 {
		glog.V(4).Infof("Account %s reach balance limit.\n", fromAddress.Hex())
		return
	}

	count := 2
	limit.Neg(limit)
	balance.Add(balance, limit)
	balance.Div(balance, big.NewInt(int64(count)))

	for i := 0; i < count; i++ {
		toKey := <-keyChannel
		toAddress := crypto.PubkeyToAddress(toKey.PublicKey)
		nonce, _ := conn.NonceAt(ctx, fromAddress, nil)
		tx := types.NewTransaction(nonce, toAddress, balance, uint64(b.GasLmit), big.NewInt(b.GasPrice), nil)
		signedTx, err := types.SignTx(tx, types.HomesteadSigner{}, fromKey)
		err = conn.SendTransaction(ctx, signedTx)
		if err != nil {
			glog.Errorf("SendTransaction error:%v\n", err)
			return
		}

		b.Locker.Lock()
		// print log
		var tpsP float64
		if time.Now().Sub(b.StartTimeP).Seconds() > 60 {
			tpsP = float64(b.FinishedTxP) / time.Now().Sub(b.StartTimeP).Seconds()
			if tpsP > b.MaxTPS {
				b.MaxTPS = tpsP
			}

			// reset period
			b.StartTimeP = time.Now()
			b.FinishedTxP = 0
		}
		tps := float64(b.FinishedTx) / time.Now().Sub(b.StartTime).Seconds()
		if tps > b.MaxTPS {
			b.MaxTPS = tps
		}
		glog.V(1).Infof("MaxTPS: %f; MaxPendingTx: %d; MaxWaitTime: %f; Period TPS: %f; Average TPS: %f; FinishedTx: %d; PendingTx: %d;\n",
			b.MaxTPS,
			b.MaxPendingTx,
			b.MaxWaitingTime,
			tpsP,
			tps,
			b.FinishedTx,
			b.PendingTx)
		b.PendingTx = b.PendingTx + 1
		if b.PendingTx > b.MaxPendingTx {
			b.MaxPendingTx = b.PendingTx
		}
		b.Locker.Unlock()

		// begin of tx
		waitStartTime := time.Now()
		_, err = bind.WaitMined(ctx, conn, signedTx)
		if err != nil {
			glog.Errorf("tx mining error:%v\n", err)
			b.Locker.Lock()
			b.PendingTx = b.PendingTx - 1
			b.Locker.Unlock()
			return
		}
		waitTime := time.Now().Sub(waitStartTime).Seconds()
		// end of tx

		b.Locker.Lock()
		if waitTime > b.MaxWaitingTime {
			b.MaxWaitingTime = waitTime
		}
		b.PendingTx = b.PendingTx - 1
		b.FinishedTx = b.FinishedTx + 1
		b.FinishedTxP = b.FinishedTxP + 1
		b.Locker.Unlock()

		b.Locker.Lock()
		if b.PendingTx > b.PendingTxLimit {
			glog.V(4).Infof("PendingTx > %d\n; return.\n", b.PendingTxLimit)
			b.Locker.Unlock()
			return
		}
		b.Locker.Unlock()
		go func() {
			b.distributeEthereum(conn, toKey, keyChannel)
		}()
	}
}

func (b *Benchmark) openConnection(jsonrpcEndpoint string) (*ethclient.Client, error) {
	conn, err := ethclient.Dial(jsonrpcEndpoint)
	if err != nil {
		glog.Errorf("Failed to dial, url: ", jsonrpcEndpoint, ", err: ", err)
		return nil, err
	}
	networkId, err := conn.NetworkID(context.Background())
	if err != nil {
		glog.Errorf("Failed to get networkId, err: ", err)
		return nil, err
	}
	glog.V(1).Infof("Connect to Ethereum success, networkId: %d\n", networkId)
	return conn, nil
}
