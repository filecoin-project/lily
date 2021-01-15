package ffi

import (
	"io"
	"os"

	abi "github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/ipfs/go-cid"
)

func GetGPUDevices() ([]string, error) {
	return nil, nil
}

func GenerateUnsealedCID(_ abi.RegisteredSealProof, _ []abi.PieceInfo) (cid.Cid, error) {
	return cid.Undef, nil
}

func GeneratePieceCIDFromFile(_ abi.RegisteredSealProof, _ io.Reader, _ abi.UnpaddedPieceSize) (cid.Cid, error) {
	return cid.Undef, nil
}

func UnsealRange(_ abi.RegisteredSealProof, _ string, _, _ *os.File, _ abi.SectorNumber, _ abi.ActorID, _ abi.SealRandomness, _ cid.Cid, _, _ uint64) error {
	return nil
}

func SealPreCommitPhase1(_ abi.RegisteredSealProof, _, _, _ string, _ abi.SectorNumber, _ abi.ActorID, _ abi.SealRandomness, _ []abi.PieceInfo) ([]byte, error) {
	return nil, nil
}

func SealPreCommitPhase2(_ []byte, _, _ string) (cid.Cid, cid.Cid, error) {
	return cid.Undef, cid.Undef, nil
}

func SealCommitPhase1(abi.RegisteredSealProof, cid.Cid, cid.Cid, string, string, abi.SectorNumber, abi.ActorID, abi.SealRandomness, abi.InteractiveSealRandomness, []abi.PieceInfo) ([]byte, error) {
	return nil, nil
}

func SealCommitPhase2([]byte, abi.SectorNumber, abi.ActorID) ([]byte, error) {
	return nil, nil
}

func ClearCache(uint64, string) error {
	return nil
}

func GenerateWinningPoSt(abi.ActorID, SortedPrivateSectorInfo, abi.PoStRandomness) ([]proof.PoStProof, error) {
	return nil, nil
}

func GenerateWindowPoSt(abi.ActorID, SortedPrivateSectorInfo, abi.PoStRandomness) ([]proof.PoStProof, []abi.SectorNumber, error) {
	return nil, nil, nil
}

type (
	SortedPrivateSectorInfo struct{ abi.SectorNumber }
	PrivateSectorInfo       struct {
		abi.SectorNumber
		CacheDirPath     string
		PoStProofType    abi.RegisteredPoStProof
		SealedSectorPath string
		SectorInfo       proof.SectorInfo
	}
)

func NewSortedPrivateSectorInfo(...PrivateSectorInfo) SortedPrivateSectorInfo {
	return SortedPrivateSectorInfo{}
}

func VerifySeal(proof.SealVerifyInfo) (bool, error) {
	return true, nil
}

func VerifyWinningPoSt(proof.WinningPoStVerifyInfo) (bool, error) {
	return true, nil
}

func VerifyWindowPoSt(proof.WindowPoStVerifyInfo) (bool, error) {
	return true, nil
}

func GenerateWinningPoStSectorChallenge(abi.RegisteredPoStProof, abi.ActorID, abi.PoStRandomness, uint64) ([]uint64, error) {
	return nil, nil
}

func FauxRep(abi.RegisteredSealProof, string, string) (cid.Cid, error) {
	return cid.Undef, nil
}

type FallbackChallenges struct {
	Sectors    []abi.SectorNumber
	Challenges map[abi.SectorNumber][]uint64
}

func GeneratePoStFallbackSectorChallenges(proofType abi.RegisteredPoStProof, minerID abi.ActorID, randomness abi.PoStRandomness, sectorIds []abi.SectorNumber) (*FallbackChallenges, error) {
	return nil, nil
}

func GenerateSingleVanillaProof(replica PrivateSectorInfo, challange []uint64) ([]byte, error) {
	return nil, nil
}
