## ethereum-playbook ![](https://img.shields.io/badge/version-1.0.0-blue.svg)

<img src="https://cl.ly/9d02d217b423/download/playbook.png" alt="ethereum-playbook" width="600px" />

**ethereum-playbook** is a simple tool that configures and deploys Ethereum DApp infrastructures using a static specification.

Ethereum is a decentralized platform that runs "persistent scripts" called smart contracts. These sentinels resemble the usual microservices in a web application architecture, each having an API and dependencies. With growing amount of services in the distributed application, the fuss with deployment and data management becomes very significant. The problem is being solved by leaders such as [Truffle Framework](http://truffleframework.com), however with great power comes a great mental overhead.

Ethereum-playbook has been designed to be simple, with lowest possible mental overhead and absoulte no learning curve. It has been designed to be more declarative, rather than imperative: the playbook acts like a catalogue of named state transitions, you can invoke them by running a single command or a chain of commands — target. Everything that is declared in the playbook: the inventory, wallets, contracts, commands is statically validate upon start. The playbook doesn't have a state, except the YAML specification with the inital state.

### Features
    
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
    - Sticky sessions for load balancing (hashmap)
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
    - Detect token symbol in value expression based known contract instances
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

Everyting is packed into nice and clear YAML synax! 🔥

### License

[MIT](/LICENSE)
