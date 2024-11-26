package util

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/filecoin-project/lotus/blockstore"
	blockservice "github.com/ipfs/go-blockservice"
	"github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	"github.com/ipld/go-car"
)

func ReadCSVAsByteSlices(filePath string) ([][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read() // Read headers
	if err != nil {
		return nil, err
	}

	var bs [][]byte
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		rowData := make(map[string]string)
		for i, value := range record {
			rowData[headers[i]] = value
		}
		jsonData, err := json.Marshal(rowData)
		if err != nil {
			return nil, err
		}
		bs = append(bs, jsonData)
	}
	return bs, nil
}

func MakeCar(name string, bs [][]byte, mhType uint64) ([]byte, error) {
	mbs := blockstore.NewMemory()
	bsv := blockservice.New(mbs, offline.Exchange(mbs))
	ds := merkledag.NewDAGService(bsv)

	pn := merkledag.NodeWithData(unixfs.FolderPBData())

	for i, bytes := range bs {
		nd, err := merkledag.NewRawNodeWPrefix(bytes, cid.V1Builder{Codec: cid.Raw, MhType: mhType})
		if err != nil {
			return nil, err
		}
		if err := ds.Add(context.TODO(), nd); err != nil {
			return nil, err
		}
		if err := pn.AddNodeLink(fmt.Sprintf("%s-%d", name, i), nd); err != nil {
			return nil, err
		}
	}

	if err := ds.Add(context.TODO(), pn); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if err := car.WriteCar(context.TODO(), ds, []cid.Cid{pn.Cid()}, &out); err != nil {
		return nil, err
	}

	// Optionally log the size
	fmt.Printf("CAR file size: %d bytes\n", out.Len())

	return out.Bytes(), nil
}
