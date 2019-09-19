soter-tools
===

This repository contains tools that use Soteria DAG projects to demonstrate functionality. Tools are located in the [cmd](cmd) directory.

You can build and install all tools with the following command:
```bash
go install ./cmd/...
```

## balance

The [balance](cmd/balance/README.md) command iterates through a dag, determining the SOTER coin balance of a given address.

## sendcoin

The `sendcoin` command demonstrates generating a transaction and sending it to a soterd network, without needing to run a full [soterwallet](https://github.com/soteria-dag/soterwallet) service.

## genwallet

The [genwallet](cmd/genwallet/README.md) command can create an offline wallet, without needing to run a full [soterwallet](https://github.com/soteria-dag/soterwallet) service.

## walletweb

The [walletweb](cmd/walletweb/README.md) utility provides a web ui for retrieving wallet address balance and sending coin to the soter network.