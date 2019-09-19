balance
===

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

The balance utility iterates through the dag, determining the SOTER coin balance of a given address.
```bash
$ balance -h
Usage of balance:
  -address string
    	Address to check balance of
  -json
    	Output in JSON format
  -mainnet
    	Use mainnet params for rpc calls
  -rpccert string
    	Soterd RPC server cert chain
  -rpcpass string
    	Soterd RPC server password to use
  -rpcserver string
    	Soterd RPC server to scan for transactions
  -rpcuser string
    	Soterd RPC server username to use
  -simnet
    	Use simnet params for rpc calls
  -testnet
    	Use testnet params for rpc calls
```

### Example usage
```
balance -simnet -rpccert /home/cedric/.soterd/rpc.cert -rpcserver 127.0.0.1:5071 -rpcuser USER -rpcpass PASS -address SQoJvhmt6QkK7itCgy4S12JN2CkVMoqNf5
```