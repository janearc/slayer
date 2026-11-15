package gaps

import "testing"

func TestAnswered(t *testing.T) {}

// anti-test of TestAnswered
// anti-test of TestAlsoAnswered
func TestRefusesBoth(t *testing.T) {}

func TestAlsoAnswered(t *testing.T) {}

func TestAlone(t *testing.T) {}

// anti-test of TestAlone with words after it
func TestLooseMarker(t *testing.T) {}

func helper(t *testing.T) {}

func TestlowerIsNotATest(t *testing.T) {}
