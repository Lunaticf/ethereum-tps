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
		Short: "Ethereum TPS test Tool.",
		Run: func(cmd *cobra.Command, args []string) {
			jsonrpcEndpoint, _ := cmd.Flags().GetString("jsonrpc-endpoint")
			mainKey, _ := cmd.Flags().GetString("main-key")

			balanceLimit, _ := cmd.Flags().GetInt64("balance-limit")
			gasLimit, _ := cmd.Flags().GetInt64("gas-limit")
			gasPrice, _ := cmd.Flags().GetInt64("gas-price")

			benchmarkIndex, _ := cmd.Flags().GetInt64("benchmark-index")
			b := benchmark.NewBenchmark(jsonrpcEndpoint, mainKey, balanceLimit, gasLimit, gasPrice)
			b.Run(benchmarkIndex)
		},
	}
	// flag
	cmd.PersistentFlags().Bool("profile", false, "Open pprof.")

	cmd.PersistentFlags().String("jsonrpc-endpoint", "http://127.0.0.1:8545", "JsonRPC endpoint for Ethereum.")
	cmd.PersistentFlags().String("main-key", "abc", "The main eth account's private key.")

	cmd.PersistentFlags().Int64("balance-limit", 10000000000000000, "When reaching this limit, account will not distribute eth.")
	cmd.PersistentFlags().Int64("gas-limit", 61569, "Global gasLimit for tx.")
	cmd.PersistentFlags().Int64("gas-price", 18000000000, "Global gasPrice for tx.")

	cmd.PersistentFlags().Int64("benchmark-index", 1, "The index of benchmark.")

	goflag.Set("v", "4")
	// TODO: once we switch everything over to Cobra commands, we can go back to calling
	// utilflag.InitFlags() (by removing its pflag.Parse() call). For now, we have to set the
	// normalize func and add the go flag set by hand.
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	goflag.CommandLine.Parse([]string{})
	logs.InitLogs()
	defer logs.FlushLogs()

	profile, _ := cmd.Flags().GetBool("profile")
	if profile {
		go func() {
			glog.Error(http.ListenAndServe(":6060", nil))
		}()
	}

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
