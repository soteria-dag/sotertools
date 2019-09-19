sendcoin
===

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

The `sendcoin` command demonstrates generating a transaction and sending it to a soterd network, without needing to run a full [soterwallet](https://github.com/soteria-dag/soterwallet) service or going through a more [manual process of creating transactions](http://www.righto.com/2014/02/bitcoins-hard-way-using-raw-bitcoin.html).

Coins from transactions matching the provided wallet (`-w`) are used for the new transaction. `-amt` SOTER of them are sent to the address specified by the `-dest` parameter. 

```bash
$ sendcoin -h
Usage of sendcoin:
  -amt float
    	Amount of coin to transfer (SOTER)
  -dest string
    	Destination address of funds
  -fee float
    	Fee for transfer (SOTER)
  -mainnet
    	Use mainnet params for wallet
  -priv string
    	Password to use, for unlocking address manager (for private keys and info)
  -pub string
    	Password to use, for opening address manager
  -rpccert string
    	Soterd RPC server cert chain
  -rpcpass string
    	Soterd RPC server password to use
  -rpcserver string
    	Soterd RPC server to send transaction to (ip:port)
  -rpcuser string
    	Soterd RPC server username to use
  -simnet
    	Use simnet params for wallet
  -source string
    	Source address of funds
  -testnet
    	Use testnet params for wallet
  -w string
    	Source wallet file name
```

#### Example usage
In this example, `sendcoin`
* Opens `mining_wallet.db`
* Connects to soterd node `127.0.0.1:5071`
* Looks in the dag for transactions matching `-source` address in the wallet
* Creates a new transaction
    * Transaction will use a matching output tx to send `10 SOTER` to address `SS9YzH3XSqovULiisvHp6oKsXQD1aprE3f`
    * For higher `-amt` values, multiple transaction outputs may be used up as inputs to the new transaction.
    * A fee of `1 SOTER` is included 
* Signs it using the wallet
* Sends the transaction to the network
* You may need to trigger additional block-generation at this point if the soterd network isn't mining on its own. 
```bash
sendcoin -simnet -rpccert /home/cedric/.soterd/rpc.cert -rpcserver 127.0.0.1:5071 -rpcuser USER -rpcpass PASS -w /home/cedric/simnet_wallet.db -priv password -pub public -source SQoJvhmt6QkK7itCgy4S12JN2CkVMoqNf5 -dest SS9YzH3XSqovULiisvHp6oKsXQD1aprE3f -amt 10 -fee 1
```

## Full example

In this example we'll spin up two soterd nodes for the purpose of demonstrating creating and sending a transaction to the network.

1. Use `genwallet` from the [soterd repository](https://github.com/soteria-dag/soterd) to generate a wallet, containing an address which will be assigned the coin reward for our mined blocks.

    ```bash
    genwallet -simnet -priv password -pub public -w /tmp/mining_wallet.db

    Created wallet: /tmp/mining_wallet.db
    Opened wallet /tmp/mining_wallet.db
    Accounts:
            name: default   number: 0       balance: 0 SOTER
                    address: SVmU9LrW1Ga7W7ufHeT6gfUiCjTttYMqcH
            name: imported  number: 2147483647      balance: 0 SOTER
    ```

    Here we can see that a new address was created under the `default` account. We'll use this address as the `--miningaddr` parameter when starting our mining node.

2. Use `genwallet` to generate another wallet. We'll use the address in this wallet as the destination address for our new transaction when calling `sendcoin` later on.

    ```bash
    genwallet -simnet -priv password -pub public -w /tmp/lucky_wallet.db
    
    Created wallet: /tmp/lucky_wallet.db
    Opened wallet /tmp/lucky_wallet.db
    Accounts:
            name: default   number: 0       balance: 0 SOTER
                    address: SMqDGyjfbT4TemzGYHFddmFR13rEjmNyp6
            name: imported  number: 2147483647      balance: 0 SOTER
    
    ```

3. Spin up a miner node, using the address from step 1.

    ```bash
    soterd --simnet --datadir=/tmp/soterd_node_a/data --logdir=/tmp/soterd_node_a/logs --listen=127.0.0.1:5070 --rpclisten=127.0.0.1:5071 --rpcuser=USER --rpcpass=PASS --connect=127.0.0.1:6070 --miningaddr=SVmU9LrW1Ga7W7ufHeT6gfUiCjTttYMqcH
    ```

4. Spin up another soterd node. This node exists so that mining can happen (miners need someone to send blocks to)

    ```bash
    soterd --simnet --datadir=/tmp/soterd_node_b/data --logdir=/tmp/soterd_node_b/logs --listen=127.0.0.1:6070 --rpclisten=127.0.0.1:6071 --rpcuser=USER --rpcpass=PASS
    ```

5. Generate at least `100 + <SOTER amount you want to spend / 50>` blocks

    The `100` value is from the `simnet` `CoinbaseMaturity` value, which says that this many blocks need to exist before you can spend the coins from a transaction with your address as the output.
    
    The `50` value is from the default `simnet` SOTER reward for mining a block.
    
    We'll generate `105` blocks on the mining node.
    
    ```bash
    soterctl --simnet --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:5071 --skipverify generate 105
    ```
    
6. Generate a transaction to spend some coins that were sent to your mining address during step 5.

    ```bash
    sendcoin -simnet -w /tmp/mining_wallet.db -priv password -pub public -rpcserver "127.0.0.1:5071" -rpcuser USER -rpcpass PASS -source SVmU9LrW1Ga7W7ufHeT6gfUiCjTttYMqcH -dest SMqDGyjfbT4TemzGYHFddmFR13rEjmNyp6 -amt 57 -fee 1
    ```

    Here we are generating a transaction that spends `57 SOTER` from transactions with our mining address as their output, and sending it to the address of our _lucky wallet_.

7. (Optional) Generate more blocks, to have new transaction included in them

    In the output of `sendcoin` it'll let us know that the transaction has been sent to the network. If the network isn't mining blocks on its own (which is the case in our example), we'll want to generate more blocks in order to ensure that our transaction gets included.
    
    ```bash
    soterctl --rpcuser=USER --rpcpass=PASS --rpcserver=127.0.0.1:5071 --simnet --skipverify generate 5
    ```
    
### `sendcoin` output

In our full example, step 6 generated a lot of output. In this section we'll review it.

```
Opened wallet /tmp/mining_wallet.db
Transactions matching wallet addresses:
block 308eb12955d157b844366a5bdb27ae438722a77160344c9083f6ae3edad533a6  height 1        tx 51b2d48194ddb84ee66b3ccb1821d147c414aa1b70569dbff8701bcd78e08897      outputNum 0     value 50 SOTER    matching wallet addr SVmU9LrW1Ga7W7ufHeT6gfUiCjTttYMqcH
block 121f4d2387ecd72c9692e972b09b985fa24ae51e46157511fff74f4a08733a8d  height 2        tx 5f6800248ee8b48af0fb29072a22dc6e62eb6628fbfd2cadffef9909a19877b5      outputNum 0     value 50 SOTER    matching wallet addr SVmU9LrW1Ga7W7ufHeT6gfUiCjTttYMqcH
. . .
```
* `sendcoin` opened the mining wallet
* Read addresses from accounts in the wallet
* Connected to the soterd node, and examined its dag from height 0 to its tips, looking for transactions whose output addresses match our specified wallet addresses.
* Output the matching addresses, and where it found them.

 ```
Creating a transaction for 57 SOTER, to SMqDGyjfbT4TemzGYHFddmFR13rEjmNyp6
```
* A new transaction is being created :)
* `SMqDGyjfbT4TemzGYHFddmFR13rEjmNyp6` is our _lucky wallet_ address.

```
Sent transaction with hash c8a606a72df43ad2944891592331fb9d2242c0a496c3d4cbf4a34b6ab0c1857c
```
* Our new transaction is signed with the mining wallet
* Transaction is sent to the network
