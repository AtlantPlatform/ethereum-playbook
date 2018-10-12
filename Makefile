all:

init:
	mkdir -p var
	geth --datadir=var/chain/ init var/genesis.json

node:
	geth --datadir=var/chain/ --rpc

attach:
	geth attach var/chain/geth.ipc

install:
	go install github.com/AtlantPlatform/ethereum-playbook

# balances:
# ethereum-playbook
