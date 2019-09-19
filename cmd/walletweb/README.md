walletweb
===

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

The walletweb utility provides a web ui for retrieving wallet address balance and sending coin to the soter network. The `-w` flag is used for the `sendcoin` feature of the ui.

```bash
$ walletweb -h
Usage of walletweb:
  -l string
    	Which [ip]:port to listen on (default ":5077")
  -mainnet
    	Use mainnet params for rpc connections
  -priv string
    	Password to use, for unlocking address manager (for private keys and info)
  -pub string
    	Password to use, for opening address manager
  -rpccert string
    	Soterd RPC server cert chain
  -rpcpass string
    	Soterd RPC server password to use
  -rpcserver string
    	Soterd RPC server connect to (ip:port)
  -rpcuser string
    	Soterd RPC server username to use
  -simnet
    	Use simnet params for rpc connections
  -testnet
    	Use testnet params for rpc connections
  -w string
    	Wallet file name (for sending coin)
```

### Example usage
```
walletweb -simnet -priv password -pub public -w /home/cedric/simnet_wallet.db -rpccert /home/cedric/.soterd/rpc.cert -rpcserver 127.0.0.1:5072 -rpcuser USER -rpcpass PASS
```