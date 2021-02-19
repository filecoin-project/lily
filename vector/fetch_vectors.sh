#!/bin/bash

set -e

while read hash; do
  lines=$(find ./data -name "${hash}*" | wc -l)
	if [ $lines -eq 0 ]; then
    echo "Fetching $hash"
    curl https://ipfs.io/ipfs/"$hash" -o tmp
    filename=$(cat ./tmp | jq ".metadata.description" | sed 's/ /_/g' | sed -e 's/^"//' -e 's/"$//')
    mv ./tmp ./data/"$hash"_"$filename".json
	fi
done <./VECTOR_MANIFEST
