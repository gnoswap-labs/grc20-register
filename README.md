<h2 align="center">
⚛️ GRC20 Register ⚛️

## Overview

`grc20-register` is a tool designed to register Gno's GRC20 token automatically whenever new grc20 token has been successfully deployed to gno.

## Key Features

- **Automatic GRC20 Register**: Automatically registers grc20 token to target register contract.
- **Concurrent Chain Indexing**: Utilizes asynchronous workers for fast and efficient indexing. Data is available for serving as soon as it is fetched from the remote chain.
  > feature came from [tx-indexer](https://github.com/gnolang/tx-indexer)
- **Embedded Database**: Features PebbleDB for quick on-disk data access and migration.
  > feature came from [tx-indexer](https://github.com/gnolang/tx-indexer)

## Getting Started

This section guides you through setting up and running the `grc20-register.

1. **Clone the Repository**

```shell
git clone github.com/gnoswap-labs/grc20-register
```

2. **Copy `.env.example` in `/addpkg` to `.env` at same location and change variables**

3. **Change register template `addpkg/template.txt`**

   > Default template is for gnoswap [pool](https://github.com/gnoswap-labs/gnoswap/blob/7d008486ba7be6ba82b469d0f8c0c30e7e022e6b/pool/token_register.gno), [router](https://github.com/gnoswap-labs/gnoswap/blob/7d008486ba7be6ba82b469d0f8c0c30e7e022e6b/router/token_register.gno), [staker](https://github.com/gnoswap-labs/gnoswap/blob/7d008486ba7be6ba82b469d0f8c0c30e7e022e6b/staker/token_register.gno) register. Use your own realm's register.

4. **Build the binary**

```shell
make build
```

5. **Run the grc20-register**

```shell
./build/grc20-register start
```

It should print something like below if automatic register ${\bf\color{#009900}SUCCEDED}$

```shell
2024-03-20T17:57:20.907+0900    INFO    fetcher fetch/fetch.go:248      Registered grc20 token  {"pkgPath": "gno.land/r/gnoswap/test_foo"}
```

It should print something like below if automatic register ${\bf\color{#ff0000}FAILED}$

```shell
2024-03-20T17:59:16.908+0900    ERROR   fetcher fetch/fetch.go:246      Failed to register grc20 token  {"pkgPath": "gno.land/r/demo/gns", "error": "transaction failed during execution, invalid package path"}
```
