GOARCH := $(GOARCH)
GOOS := $(GOOS)
FLAGS := -ldflags "-s -w"
PEER_LIST := "assets/peers.txt"
peers.go := "src/autopeering/peers.go"

all: peers_asset
	GOARCH=$$GOARCH GOOS=$$GOOS go build $(FLAGS) ./cmd/yggdrasil
	GOARCH=$$GOARCH GOOS=$$GOOS go build $(FLAGS) ./cmd/yggdrasilctl

clean:
	$(RM) yggdrasil yggdrasil.exe yggdrasilctl yggdrasilctl.exe

peers_asset:
	echo "package autopeering\n" > $(peers.go)
	echo -n "var HighCapPeers = []string{" >> $(peers.go)
	for line in `grep -E "^h" $(PEER_LIST) | cut -d "," -f2 | head -n -1`
	do 
		echo "\t\"$$line\"," >> $(peers.go)
	done
	echo "\t\"`grep -E "^h" $(PEER_LIST) | cut -d "," -f2 | tail -n 1`\"}" >> $(peers.go)
	echo -n "var LowCapPeers = []string{" >> $(peers.go)
	for line in `grep -E "^l" $(PEER_LIST) | cut -d "," -f2 | head -n -1`
	do 
		echo "\t\"$$line\"," >> $(peers.go)
	done
	echo "\t\"`grep -E "^l" $(PEER_LIST) | cut -d "," -f2 | tail -n 1`\"}" >> $(peers.go)

.PHONY: peers_asset all clean
.SILENT: peers_asset
.ONESHELL: peers_asset
