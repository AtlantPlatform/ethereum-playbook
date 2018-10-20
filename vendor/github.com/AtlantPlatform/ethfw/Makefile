all:

init:
	mkdir -p var
	geth --datadir=var/chain/ init var/genesis.json

node:
	geth --datadir=var/chain/ --rpc

attach:
	geth attach var/chain/geth.ipc --preload "scripts/allbalances.js"

