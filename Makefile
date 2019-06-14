all:

init:
	mkdir -p var
	geth --datadir=var/chain/ init var/genesis.json

node:
	geth --datadir=var/chain/ --keystore examples/keystore --rpc

attach:
	geth attach var/chain/geth.ipc

install:
	go install github.com/AtlantPlatform/ethereum-playbook

lint:
	golangci-lint run

demo:
	asciicast2gif -t solarized-light https://asciinema.org/a/mwGXhJ6p9hAGmiI12Dw7hxNoy.json token-demo.gif

xgo:
	xgo -out bin/ethereum-playbook \
		--targets=windows/386,windows/amd64,darwin/amd64,linux/amd64,linux/arm-7,linux/386 \
		github.com/AtlantPlatform/ethereum-playbook

release:
	cp a b
