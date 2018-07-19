package main

import (
	goflag "flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilflag "k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"

	"github.com/hello2mao/ethereum-tps"
)

func main() {
	cmd := &cobra.Command{
		Use:   "ethereum-tps",
		Short: "Ethereum TPS Test.",
		Run: func(cmd *cobra.Command, args []string) {
			jsonrpcEndpoint, err := cmd.Flags().GetString("jsonrpc-endpoint")
			if err != nil {
				glog.Errorf("Failed to get jsonrpc-endpoint, err: ", err)
			}
			mainKey, err := cmd.Flags().GetString("main-key")
			if err != nil {
				glog.Errorf("Failed to get main-key, err: ", err)
			}
			balanceLimit, err := cmd.Flags().GetInt64("balance-limit")
			if err != nil {
				glog.Errorf("Failed to get balance-limit, err: ", err)
			}
			gasLimit, err := cmd.Flags().GetInt64("gas-limit")
			if err != nil {
				glog.Errorf("Failed to get gas-limit, err: ", err)
			}
			gasPrice, err := cmd.Flags().GetInt64("gas-price")
			if err != nil {
				glog.Errorf("Failed to get gas-price, err: ", err)
			}
			pendingTxLimit, err := cmd.Flags().GetInt64("pending-tx-limit")
			if err != nil {
				glog.Errorf("Failed to get pending-tx-limit, err: ", err)
			}
			b := benchmark.NewBenchmark(jsonrpcEndpoint, mainKey, balanceLimit, gasLimit, gasPrice, pendingTxLimit)
			b.Run()
		},
	}
	// flag
	cmd.PersistentFlags().String("jsonrpc-endpoint", "http://127.0.0.1:8545", "JsonRPC endpoint for Ethereum")
	cmd.PersistentFlags().String("main-key", "http://127.0.0.1:8545", "")
	cmd.PersistentFlags().Int64("balance-limit", 10000000000000000, "")
	cmd.PersistentFlags().Int64("gas-limit", 61569, "")
	cmd.PersistentFlags().Int64("gas-price", 18000000000, "")
	cmd.PersistentFlags().Int64("pending-tx-limit", 400, "")

	goflag.Set("v", "4")
	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	goflag.CommandLine.Parse([]string{})
	logs.InitLogs()
	defer logs.FlushLogs()

	go func() {
		glog.Error(http.ListenAndServe(":6060", nil))
	}()

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
