soter-tools
===

This repository contains tools that use Soteria DAG projects to demonstrate functionality. Tools are located in the [cmd](cmd) directory.

You can build and install all tools with the following command:
```bash
go install ./cmd/...
```

## gentx

The [gentx](cmd/gentx/README.md) command demonstrates generating a transaction and sending it to a soterd network, without needing to run a full [soterwallet](https://github.com/soteria-dag/soterwallet) service or going through a more [manual process of creating transactions](http://www.righto.com/2014/02/bitcoins-hard-way-using-raw-bitcoin.html).


## genwallet

The [genwallet](cmd/genwallet/README.md) command can create an offline wallet, without needing to run a full [soterwallet](https://github.com/soteria-dag/soterwallet) service.