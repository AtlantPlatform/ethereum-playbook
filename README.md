# ethereum-playbook ![](https://img.shields.io/badge/version-1.0.0-blue.svg) ![](https://img.shields.io/badge/lines-2500-silver.svg)

<img src="https://cl.ly/9d02d217b423/download/playbook.png" alt="ethereum-playbook" width="600px" />

**ethereum-playbook** is a simple tool that configures and deploys Ethereum DApp infrastructures using a static specification.

Ethereum is a decentralized platform that runs "persistent scripts" called smart contracts. These sentinels resemble the usual microservices in a web application architecture, each having an API and dependencies. With growing amount of services in the distributed application, the fuss with deployment and data management becomes very significant. The problem is being solved by leaders such as [Truffle Framework](http://truffleframework.com), however with great power comes a great mental overhead.

Ethereum-playbook has been designed to be simple, with lowest possible mental overhead and absoulte no learning curve. It has been designed to be more declarative, rather than imperative: the playbook acts like a catalogue of named state transitions, you can invoke them by running a single command or a chain of commands. Everything that is declared in a playbook: inventory, wallets, contracts, commands and targets is statically validated upon start. The playbook doesn't have a state, except the YAML specification with the inital state. It has been inspired by [Ansible Playbooks](https://docs.ansible.com/ansible/devel/user_guide/playbooks.html) however has a different terminology and approach.

Ethereum-playbook is not a part of the ATLANT product platform and is licensed under MIT license. It is used internally for Ethereum smart contract testing and validation purposes. Use in production is not recommended, only on your own risk.

## Features
    
* Geth nodes inventory with healthcheck
    - JSON-RPC endpoints
    - IPC sockets
* Wallet management
    - Load accounts by JSON keyfile
    - Keyfile auto-locate in keystore
    - Load using private key
    - Password-protected keys
    - Run commands for wallets matching Regexp
    - Run commands with balancing among wallets
    - Sticky sessions for load balancing (hashring)
* Contracts management
    - Solidity ABI/BIN compilation using `solc`
    - Instance deployment
    - Instance binding
    - Token symbol autodiscovery
* Calls
    - Call any JSON-RPC method, with params
    - Wallet address placeholders
    - Argument placeholder from CLI
    - Math evaluation, can use math expressions
    - Field refrences — reference certain wallet properties such as password
    - Runs for each wallet by regexp
* Contract View
    - Call view methods of bound contract instances
    - Run for each wallet by regexp
* Contract Transactions
    - Invoke write transactions, such as contract deployment
    - Auto-binding after contract deployment
    - Call write methods of bound contract instances
    - Math expressions and field references in the value
* Ether Transactions
    - Send ether between accounts
    - Math expressions and field references in the value
    - Ether denominators - wei, gwei, ether
* Token Transactions
    - Works as Ether Transactions
    - Detect token symbol in value expression based on the known contract instances
    - Invokes target contract's transfer method
    - Math expressions and field references in the value
    - Load-balancing among different wallets, sticky sessions
* Targets
    - Run all listed commands in a batch
    - All transactions are synced, i.e. wait each other
    - Mark certain transactions async to run in background
* CLI
    - Command Line Interface autogeneration
    - Static validation of command arguments (count, types, math)

Everyting is packed into nice and clean YAML synax! 🔥

## Demo

[![token-demo](https://cl.ly/31ddd7850b51/download/token-demo.gif)](https://asciinema.org/a/mwGXhJ6p9hAGmiI12Dw7hxNoy?autoplay=1)

Demo is based on the flow defined in [examples/tokens.yml](/examples/tokens.yml).

## Installation

Using `brew` (for macOS):

```bash
$ brew tap AtlantPlatform/Apps
$ brew install ethereum-playbook
```

### All platforms

Grab a release executable for your system in our [Releases](https://github.com/AtlantPlatform/ethereum-playbook/releases) section.

To compile the tool manually, you need to install the [Go programming language](https://golang.org/dl/) compiler first. This is out of scope of this document. Once it is installed, there is a single command:

```bash
$ go get -u github.com/AtlantPlatform/ethereum-playbook
```

Make sure the binary is placed into one of your `$PATH` prefixes to be available in the shell.

## Usage

When you start the tool without specifying anything, it will print the basic help. All other commands and their arguments are being **auto-generated** after specifying an YAML spec.

```bash
$ ethereum-playbook -f examples/tokens.yml help

Usage: ethereum-playbook [OPTIONS] COMMAND [arg...]

Ethereum contracts deployment and management tool.

Options:
  -f                      Custom path to playbook.yml spec file. (default "playbook.yml")
  -s                      Name or path of Solidity compiler (solc, not solcjs). (default "solc")
  -g                      Inventory group name, corresponding to Geth nodes. (default "genesis")
  -l, --log-level         Sets the log level (default: info) (default 4)

Commands:
  make-transfers          Target with 5 commands, accepts 0 args
  eth-balances            Generic CALL command, accepts 0 args
  txinfo                  Generic CALL command, accepts 1 args
  txreceipt               Generic CALL command, accepts 1 args
  get-owner               Generic VIEW command, accepts 0 args
  token-balances          Generic VIEW command, accepts 0 args
  deploy-property-token   Generic WRITE command, accepts 0 args
  mint-100-tokens         Generic WRITE command, accepts 0 args
  send-1-eth              Generic WRITE command, accepts 0 args
  send-100-gwei           Generic WRITE command, accepts 0 args
  send-100-wei            Generic WRITE command, accepts 0 args
  send-25-tokens          Generic WRITE command, accepts 0 args
  send-wei                Generic WRITE command, accepts 1 args
  transfer-50-tokens      Generic WRITE command, accepts 0 args

Run 'ethereum-playbook COMMAND --help' for more information on a command.
```

The CLI interface above has been generated from [examples/tokens.yml](/examples/tokens.yml).

Calling the tool without specifying any command will validate the spec:

```bash
$ ethereum-playbook -f examples/tokens.yml

INFO[0000] loaded address from keyfile   address=0xa480763627636ff8b8ce97d0d6608e99fddb1062 section=Wallets wallet=bob
INFO[0000] loaded address from privkey   address=0xddb987896df947ee5aeb2bbb5d387008ed9dceef section=Wallets wallet=alice
INFO[0004] spec validated
```

## A Deep Dive Into the Spec

The spec is an YAML file with sections. Each section defines various properties of the spec, most of them are optional. The whole structure can be seen as this:

```yaml
---

INVENTORY:
  name:
    # list of Geth nodes

WALLETS:
  name:
    # wallet specification
    # credentials

CONTRACTS:
  name:
    # contract code specification
    # contract instances specification

CALL:
  name:
    # command specification
    # params specification

VIEW:
  name:
    # command specification
    # params specification

WRITE:
  name:
    # command specification
    # params specification

TARGETS:
  name:
    # list of commands

CONFIG:
  name: # config value
```

We will walk through each section and explain how it should look like.

### Geth Inventory

```yaml
INVENTORY:
  testnet:
    - http://localhost:8545
  genesis:
    - var/chain/geth.ipc
```

You can specify Geth node groups in the inventory section. By default, the playbook tries to load `genesis` group, as it usually corresponds to a private test chain, ran by some local Geth nodes. The list of nodes should be in a form of `JSON-RPC` endpoints or IPC socket file paths. Nodes are checked for liveness when the specification is being validated upon startup, at least one node in the specified inventory group must be alive.

### Wallet Management

```yaml
WALLETS:
  alice:
    privkey: "41022453C949BAB4821358D2FA5B93CA6B046EFFA7B7A19765ACF8FD6AE8FA9B"

  bob:
    keyfile: "examples/keystore/bob.json"
    password: "1234"

  foo3:
    address: 0x3b47427740b5dedf1bfae36862a78d7134609607
    keystore: "var/chain/keystore"
    password: "1234"
```

There are multiple ways to specify the account credentials. The section is called wallets, each wallet has a name, and the corresponding specification on how to obtain private key for transaction signing.

With `privkey` field it is possible to have an unprotected private key (generated with `crypto.SaveECDSA()`) for an account, the address will be derived from it. The most simple way is to specify the `keystore` prefix path, where protected keys are stored, usually it's within the `--datadir` of the local Geth node. That allows to make lookups for keyfiles by an account `address`. You can specify the path to a `keyfile` explicitly, relatively to the `keystore` path. You need to supply the password to unlock the keys.

Absolute paths are supported, however we discourage using absolute paths in the specification, as this will affect cross-platform use cases.

Wallets keep some properties that can be fetched dynamically, for example, an ETH balance can be fetched, so it can be used in commands, also user can reference one wallet's password, more about field references later (see [Params](#params)).

### Contracts Management

```yaml
CONTRACTS:
  property-token:
    name: PropertyToken
    sol: contracts/PropertyToken.sol
    instances:
      - &PTO123
        contract: property-token
        address: 0xecc5c5b61f3833af29dcf5f1597f20ca0e6d4fa3
      - &PTO124
        contract: property-token
        address: 0x0
```

After the inventory and wallets are set, it's time to add some smart contracts. The contracts section allows to add Solidity sources that will be compiled with `solc` to validate, and specify the instances, if there is any. As we can see from the example above, it will load `contracts/PropertyToken.sol` as root source file, and will pick-up "PropertyToken" ABI and BIN, will bind that contract the the instance located at `0xecc5c5b61f3833af29dcf5f1597f20ca0e6d4fa3`.

There is no names for instances, to reference one from the commands, you should use a combination of contract source name and the instance address. So it's recommended to just leverage [Anchor & Alias Nodes](http://yaml.org/spec/1.2/spec.html#id2786196) from YAML. We mark a block with `&PTO123` and then use it as an alias, example:

```yaml
VIEW:
  get-owner:
    instance: *PTO123
    method: owner
```

We could simply copy-pasted the two fields (`contract`, `address`), but anchor-alias approach is superior for DRY and keeping the contract specs in one place.

When no address is specified, or the address is `0x0`, the contract is meant to be deployed. Playbook can deploy contracts, more on this later (see [Contract Transactions](#contract-transactions)). However, when the new contract address is generated, it's user's responsibility to add that address into the instance spec. Because the specification is not dynamic, and is evaluated on the start only, with exception to some wallet properties such as balances.

### Calls

```yaml
CALL:
  block:
    method: eth_blockNumber
  new-account:
    method: personal_newAccount
    params:
      - {type: string, reference: @alice.password}
  eth-balances:
    wallet: .
    method: eth_getBalance
    params:
      - {type: address, value: @@}
      - latest
```

Commands are divided into three main categories: `CALL`, `VIEW` and `WRITE`. In the `CALL` section user should place any JSON-RPC commands that are not interacting with smart contracts or signing transactions. There you can retrieve various info about the Ethereum network, use personal API (if allowed by Geth instance), start or stop the local miner. It is possible to manually invoke low-level `eth_*` methods such as `eth_sendRawTransaction`. 

The `wallet` field is a filter, if not specified, the command runs without context about wallets. It is a regexp string, so having `.` there means that the command will run in a context of an array of all possible wallets. Example: the spec has five wallets, and `eth-balances` has `wallet: .`, so it will run the `method` five times, against each wallet. To use the current wallet address in the method params, you must write `@@` as a placeholder.

### Params

All commands have `params` specification that is an ordered array of arguments for the used `method`. By default, the param is a string, and cannot have any field references or placeholders, or math expressions. All Ethereum types are supported in params:

```yaml
- Just a plain string, yo!
- {type: address, value: 0xecc5c5b61f3833af29dcf5f1597f20ca0e6d4fa3}
- {type: string, value: Hello World}
- {type: int, value: -50 * 1e18}
- {type: int8, value: 0xFF}
- {type: int16, value: 1337}
- {type: int32, value: -1337}
- {type: int64, value: 1 << 12}
- {type: int128, value: 1337}
- {type: int256, value: -50 * 1e18}
- {type: uint, value: 50 * 1e18}
- {type: uint8, value: 0xFF}
- {type: uint16, value: 1337}
- {type: uint32, value: -1337}
- {type: uint64, value: 1 << 12}
- {type: uint128, value: 1337}
- {type: uint256, value: 50 * 1e18}
- {type: bool, value: true}
- {type: byte, value: 0xED}
- {type: bytes, value: 0xdeadbeef}
```

Notice that all params here are values. And if the type is numeric, math expressions are allowed too. There is more on top for the flexibility of params: you can reference wallet fields or arguments from CLI:

```yaml
- {type: address, reference: $1}
- {type: string, reference: @alice.password}
- {type: uint, reference: @alice.balance}
- {type: uint, reference: $2}
```

The `@` symbol is specific to wallet field references (by field name), while `$` is specific to CLI arguments (by offset). `$0` is always the name of the current command, to reflect UNIX philosophy a little bit. All the argument placeholders are parsed statically by ethereum-playbook, to generate the apropriate CLI specification.

```
Commands:
  done              Target with 1 commands, accepts 0 args
  run               Target with 5 commands, accepts 0 args
  view              Target with 2 commands, accepts 1 args
  new-account       Generic CALL command, accepts 0 args
  block             Generic CALL command, accepts 0 args
  txinfo            Generic CALL command, accepts 1 args
  txreceipt         Generic CALL command, accepts 1 args

$ ethereum-playbook -f examples/targets.yml txinfo -h

Usage: ethereum-playbook txinfo ARG1

Generic CALL command, accepts 1 args

Arguments:
  ARG1         Command argument $1
```

### Contract View

```yaml
VIEW:
  get-owner:
    instance: *PTO123
    method: owner

  token-balances:
    wallet: .
    instance: *PTO123
    method: balanceOf
    params:
      - {type: address, value: @@}
```


In this `VIEW` section we have the same logic as for `CALL`, however, a deployed contract instance is required, as this section is for commands, that will call the contract view-only instance methods.

It's recommended to leverage [Anchor & Alias Nodes](http://yaml.org/spec/1.2/spec.html#id2786196) from YAML, to reference the contract instances from their section. Otherwise, specify the instance's `contract` name and the `address`:

```yaml
VIEW:
  token-balances:
    wallet: .
    instance:
        contract: property-token
        address: 0x0
```

Same as with `CALL` commands, the wallet spec is a regexp, so the method will be called multiple times, for each matching wallet address. By using a placeholder, we can get a summary of results for all wallets:

```bash
$ ethereum-playbook -f examples/tokens.yml token-balances

0xddb987896df947ee5aeb2bbb5d387008ed9dceef (@alice): "75000000000000000000"
0xa480763627636ff8b8ce97d0d6608e99fddb1062 (@bob): "25000000000000000000"
```

### Send Ether

```yaml
WRITE:
  send-100-wei:
    wallet: alice
    to: bob
    value: 100
```

And the most important section `WRITE` specifies the commands that alter the blockchain state by signing and sending Ethereum transactions. Simple as that, we can specify `wallet` to use for signing, and `to` recipient, to send any amount of ether. The difference from `CALL` and `VIEW` sections is that it uses only one matching wallet. It uses hashring balancing algorithm with sticky sessions (`sticky: "someinfo"`) to pick one wallet from a set of all matched wallets.

The `value` field is really smart here. It supports math expressions, as we have used in params, but it also supports value denominators. There are few base denominators:

```yaml
value: 100 # empty
value: 100 wei
value: 100 eth # same as wei
value: 100 gwei # 1e9 wei
value: 1 ether # 1e18 wei
```

It will convert the value from any custom denominator to the base Wei before sending the transaction. Moreover, it also supports references and argument placeholders!

```yaml
value: @alice.balance - (40 * 1e9 * 21000) # reference wallet's field
value: $1 * 5 ether # use first arg from CLI
```

It is important to have the denominator at the end of the string only, as the whole value should have only one total denominator. And it should be separated by space from the math expression.

### Send Tokens

Another feature that is possible by contract instance discovery — you can use a token symbol in value expression, to invoke the transfer method of the corresponding contract instance. The contract instance's token symbol is being detected automatically.

```yaml
WRITE:
  send-25-tokens:
    wallet: bob
    to: alice
    value: 25 * 1e18 PTO123
```

The spec above will internally match `PTO123` symbol name with one of the known contract instances and will send a write transaction to its `transfer` method. This allows to send tokens without care about contract methods, as simply as sending ethers between addresses.

### Contract Transactions

```yaml
WRITE:
  deploy-property-token:
    wallet: bob
    instance:
      contract: property-token
    params:
      - Atlant Property Token 123
      - PTO123
      - {type: uint, value: 50 * 1e6 * 1e18}
```

If there is no `to` address specified and the contract is not deployed, the spec above will sign and send a contract deploy transaction with provided params, paying for the gas using Bob's wallet.

```yaml
WRITE:
  mint-100-tokens:
    wallet: bob
    instance: *PTO123
    method: mint
    params:
      - {type: address, value: @bob}
      - {type: uint, value: 100 * 1e18}
```

While Bob is an owner of the newly contract instance, he can invoke contract methods that are available only for him, for example mint some tokens for his address. The instance must be specified beforehand in the corresponding section:

```yaml
CONTRACTS:
  property-token:
    name: PropertyToken
    sol: contracts/PropertyToken.sol
    instances:
      - &PTO123
        contract: property-token
        address: 0xecc5c5b61f3833af29dcf5f1597f20ca0e6d4fa3
```

So, the playbook will sign a transaction using Bob's private key and send it to `0xecc5c5b61f3833af29dcf5f1597f20ca0e6d4fa3` contract, calling its `mint` method using the ABI from `contracts/PropertyToken.sol`. In a few lines! 😱

### Targets 

```yaml
TARGETS:
  run:
    - miner-rebase
    - miner-start
    - balance
    - burn-all
    - balance
  view:
    - txinfo
    - txreceipt
  done:
    - miner-stop
```

Targets allow to run all commands from the list in a batch mode. Instead of invoking the commands one-by-one and validating the spec each time, the ethereum-playbook can run once and execute multiple commands. All transactions (write commands) are synced between calls, i.e. will wait each other. It is possible to invoke commands in background with bash-like syntax using `&`:

```yaml
TARGETS:
  run:
    - send-to-alice &
    - send-to-bob &
    - send-to-others
    - balances
```

So `send-to-alice`, `send-to-bob` and `send-to-others` will be signed and executed simultaneously, while `balances` will wait for the latest command without amp: `send-to-others`. The three transactions will be sent from different wallets, if there is at least three wallets matching the regexp, also if no `sticky` marker is set in the commands.

```yaml
TARGETS:
  make-transfers:
    - token-balances
    - mint-100-tokens
    - transfer-50-tokens
    - send-25-tokens
    - token-balances
```

Will be invoked in a sequence, results will be printed once available from each command:

```bash
$ ethereum-playbook -f examples/tokens.yml make-transfers

INFO[0000] loaded address from privkey    address=0xddb987896df947ee5aeb2bbb5d387008ed9dceef section=Wallets wallet=alice
INFO[0000] loaded address from keyfile    address=0xa480763627636ff8b8ce97d0d6608e99fddb1062 section=Wallets wallet=bob

token-balances:
    0xddb987896df947ee5aeb2bbb5d387008ed9dceef (@alice): "0"
    0xa480763627636ff8b8ce97d0d6608e99fddb1062 (@bob): "0"

mint-100-tokens:
    "0x80c8b1eca7fce7f227782853a0ed8f8acc979de1d371aa6bbf0e7269b7dc7081"

transfer-50-tokens:
    "0x768baa938f383c8943f84d4385a6439c3a3e3b262b3f99568ed44d654fd711f2"

send-25-tokens:
    "0xeb7e2245c6f7e24da7553d3b7fa5bb6444dbd788a911f3244b14eb5cc78421aa"

token-balances:
    0xddb987896df947ee5aeb2bbb5d387008ed9dceef (@alice): "75000000000000000000"
    0xa480763627636ff8b8ce97d0d6608e99fddb1062 (@bob): "25000000000000000000"
```

### Config

And the last, but not the least, the config section with some global parameters. Defaults are:

```yaml
CONFIG:
  gasPrice: 10000000000 # 10 gwei
  gasLimit: 10000000 # hard limit
  chainID: 1 # https://eips.ethereum.org/EIPS/eip-155
  awaitTimeout: 10m # when executing target
```

## Example Specs

* [examples/tokens.yml](/examples/tokens.yml) — a spec that shows how to deploy contracts and manage ERC20 tokens;
* [examples/targets.yml](/examples/targets.yml) — a spec showing how to use targets effectively, also shows some personal JSON-RPC usage cases.

## Code quality [![Go Report Card](https://goreportcard.com/badge/github.com/AtlantPlatform/ethereum-playbook)](https://goreportcard.com/report/github.com/AtlantPlatform/ethereum-playbook)

All linter checks have been passed. All exceptional cases in the code are covered and logged.

```bash
# linters:
#   enable-all: true
#   disable:
#     - gocyclo
#     - goimports

$ golangci-lint run
$
```

We would like to undergo a 3rd-party security audit for the code and logic, if you're providing such services for free, please contact us or raise an issue.

## License

[MIT](/LICENSE)
