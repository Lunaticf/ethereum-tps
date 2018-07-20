# Ethereum TPS Test

```shell
Ethereum TPS test Tool.

Usage:
  ethereum-tps [flags]

Flags:
      --alsologtostderr                  log to standard error as well as files
      --balance-limit int                When reaching this limit, account will not distribute eth. (default 10000000000000000)
      --gas-limit int                    Global gasLimit for tx. (default 61569)
      --gas-price int                    Global gasPrice for tx. (default 18000000000)
  -h, --help                             help for ethereum-tps
      --jsonrpc-endpoint string          JsonRPC endpoint for Ethereum. (default "http://127.0.0.1:8545")
      --log-backtrace-at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log-dir string                   If non-empty, write log files in this directory
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --logtostderr                      log to standard error instead of files (default true)
      --main-key string                  The main eth account's private key. (default "abc")
      --pending-tx-limit int             The num of pending tx limit. (default 400)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```