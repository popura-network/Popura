#!/bin/sh

echo "package autopeering\n"

echo -n "var HighCapPeers = []string{"
for line in `grep -E "^h" assets/peers.txt | cut -d "," -f2 | head -n -1`
do
	echo "\t\"$line\","
done
echo "\t\"`grep -E "^h" assets/peers.txt | cut -d "," -f2 | tail -n 1`\"}"

echo -n "var LowCapPeers = []string{"
for line in `grep -E "^l" assets/peers.txt | cut -d "," -f2 | head -n -1`
do
	echo "\t\"$line\","
done
echo "\t\"`grep -E "^l" assets/peers.txt | cut -d "," -f2 | tail -n 1`\"}"
