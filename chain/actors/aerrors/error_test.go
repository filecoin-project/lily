package aerrors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-state-types/exitcode"
<<<<<<< HEAD
	. "github.com/filecoin-project/lily/chain/actors/aerrors"
=======
	"github.com/filecoin-project/lily/chain/actors/aerrors"
>>>>>>> master
)

func TestFatalError(t *testing.T) {
	e1 := xerrors.New("out of disk space")
	e2 := fmt.Errorf("could not put node: %w", e1)
	e3 := fmt.Errorf("could not save head: %w", e2)
	ae := aerrors.Escalate(e3, "failed to save the head")
	aw1 := aerrors.Wrap(ae, "saving head of new miner actor")
	aw2 := aerrors.Absorb(aw1, 1, "try to absorb fatal error")
	aw3 := aerrors.Wrap(aw2, "initializing actor")
	aw4 := aerrors.Wrap(aw3, "creating miner in storage market")
	t.Logf("Verbose error: %+v", aw4)
	t.Logf("Normal error: %v", aw4)
	assert.True(t, aerrors.IsFatal(aw4), "should be fatal")
}
func TestAbsorbeError(t *testing.T) {
	e1 := xerrors.New("EOF")
	e2 := fmt.Errorf("could not decode: %w", e1)
	ae := aerrors.Absorb(e2, 35, "failed to decode CBOR")
	aw1 := aerrors.Wrap(ae, "saving head of new miner actor")
	aw2 := aerrors.Wrap(aw1, "initializing actor")
	aw3 := aerrors.Wrap(aw2, "creating miner in storage market")
	t.Logf("Verbose error: %+v", aw3)
	t.Logf("Normal error: %v", aw3)
	assert.Equal(t, exitcode.ExitCode(35), aerrors.RetCode(aw3))
}
