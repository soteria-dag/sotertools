genwallet
===

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

The `genwallet` command can create an offline wallet, without needing to run a full [soterwallet](https://github.com/soteria-dag/soterwallet) service.

If there aren't any addresses found in the wallet, `genwallet` will also create and display one.
```
$ genwallet -h
Usage of genwallet:
  -mainnet
        Use mainnet params for wallet
  -priv string
        Password to use, for unlocking address manager (for private keys and info)
  -pub string
        Password to use, for opening address manager
  -simnet
        Use simnet params for wallet
  -testnet
        Use testnet params for wallet
  -w string
        Wallet file name

```

#### Example usage
```
genwallet -simnet -priv password -pub public -w /tmp/mining_wallet.db
```