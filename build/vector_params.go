package build

import _ "embed"

//go:embed test-vectors/vectors.json
var vectors []byte

func VectorsJSON() []byte {
	return vectors
}
