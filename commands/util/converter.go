package util

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"

	blockservice "github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/boxo/ipld/unixfs"
	"github.com/ipfs/go-cid"
	offline "github.com/ipfs/go-ipfs-exchange-offline"
	"github.com/ipld/go-car"

	commcid "github.com/filecoin-project/go-fil-commcid"
	commp "github.com/filecoin-project/go-fil-commp-hashhash"

	"github.com/filecoin-project/lotus/blockstore"
)

func ReadTargetFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func ReadCSVAsByteSlices(filePath string) ([][]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil {
			// Handle or log the error from file.Close
			fmt.Printf("Error closing file: %v", cerr)
		}
	}()

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

func MakeCar(name string, b []byte, mhType uint64) ([]byte, error) {
	mbs := blockstore.NewMemory()
	bsv := blockservice.New(mbs, offline.Exchange(mbs))
	ds := merkledag.NewDAGService(bsv)

	pn := merkledag.NodeWithData(unixfs.FolderPBData())

	nd, err := merkledag.NewRawNodeWPrefix(b, cid.V1Builder{Codec: cid.Raw, MhType: mhType})
	if err != nil {
		return nil, err
	}
	if err := ds.Add(context.TODO(), nd); err != nil {
		return nil, err
	}
	if err := pn.AddNodeLink(fmt.Sprintf("%s", name), nd); err != nil {
		return nil, err
	}

	if err := ds.Add(context.TODO(), pn); err != nil {
		return nil, err
	}

	var carBuffer bytes.Buffer

	cp := new(commp.Calc)

	multiWriter := io.MultiWriter(&carBuffer, cp)

	if err := car.WriteCar(context.TODO(), ds, []cid.Cid{pn.Cid()}, multiWriter); err != nil {
		return nil, err
	}

	// Optionally log the size
	fmt.Printf("CAR file size: %d bytes\n", carBuffer.Len())

	// Calculate piece-size and piece-cid
	rawCommP, pieceSize, err := cp.Digest()
	if err != nil {
		return nil, err
	}

	pieceCID, err := commcid.DataCommitmentV1ToCID(rawCommP)
	if err != nil {
		return nil, err
	}

	payloadCID := pn.Cid()
	fmt.Printf("piece size: %v\n", pieceSize)
	fmt.Printf("payload cid: %v\n", payloadCID)

	fmt.Printf("piece cid: %v\n", pieceCID)

	return carBuffer.Bytes(), nil
}
