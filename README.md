# Ethereum TPS Test

```shell
Ethereum TPS Test.

Usage:
  ethereum-tps [flags]

Flags:
      --alsologtostderr                  log to standard error as well as files
      --balance-limit int                 (default 10000000000000000)
      --gas-limit int                     (default 61569)
      --gas-price int                     (default 18000000000)
  -h, --help                             help for ethereum-tps
      --jsonrpc-endpoint string          JsonRPC endpoint for Ethereum (default "http://127.0.0.1:8545")
      --log-backtrace-at traceLocation   when logging hits line file:N, emit a stack trace (default :0)
      --log-dir string                   If non-empty, write log files in this directory
      --log-flush-frequency duration     Maximum number of seconds between log flushes (default 5s)
      --logtostderr                      log to standard error instead of files (default true)
      --main-key string                   (default "http://127.0.0.1:8545")
      --pending-tx-limit int              (default 400)
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
  -v, --v Level                          log level for V logs (default 4)
      --vmodule moduleSpec               comma-separated list of pattern=N settings for file-filtered logging
```