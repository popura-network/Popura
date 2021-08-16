# Popura ポプラ

Popura, an alternative Yggdrasil network client

*Yggdrasil Network* is a peer-to-peer IPv6 network with link-local peer discovery, 
automatic end-to-end encryption, distributed IP address allocation, and DHT-based routing information exchange.

Popura uses the same Yggdrasil core API internally, but adds some useful 
experimental features which the original client lacks.

By default, it works just like the original yggdrasil client, all features must be enabled manually. 
Popura adds new command line flags and config file sections to control those features.

## Features

- [Autopeering](https://github.com/popura-network/Popura/wiki/Autopeering) over the Internet
- Built-in decentralized DNS system [meshname](https://github.com/popura-network/Popura/wiki/Meshname)

## Installing

- [Debian](https://github.com/popura-network/popura-debian-repo)
- [Arch Linux](https://aur.archlinux.org/packages/popura-git/)
- [OpenWRT](https://github.com/popura-network/hypermodem-packages)
- [Windows](https://github.com/popura-network/Popura/releases)

## Building from source

1. Install Go
2. Clone this repository
3. Run `./build`

## Information

[Wiki](https://github.com/popura-network/Popura/wiki)

[Blog](https://popura-network.github.io)

[Telegram channel](https://t.me/PopuraChan)

[Yggdrasil documentation](https://yggdrasil-network.github.io/)
