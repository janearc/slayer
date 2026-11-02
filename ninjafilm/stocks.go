package ninjafilm

import (
	"github.com/janearc/slayer/ninjafilm/data"
	"github.com/janearc/slayer/ninjafilm/roll"
	"github.com/janearc/slayer/ninjafilm/shelf"
)

// Stocks is the name of every stock on the shelf, in order. Roll is
// asked for a name.
//
//ninja:vow practice slayer lists the shelf here; no caller in this tree
func Stocks() ([]string, error) {
	return shelf.Of(data.StocksTable(), data.RecordsTable(),
		data.GaugesTable()).Names()
}

// Roll is a roll of the stock of a name, exposed at its rating. The
// seed says which roll. The same seed is the same roll, grain for
// grain. Names not on the shelf are refused, and the refusal lists the
// names that are.
//
//ninja:vow practice slayer loads each roll here; no caller in this tree
func Roll(name string, seed uint64) (roll.Roll, error) {
	tables := shelf.Of(data.StocksTable(), data.RecordsTable(),
		data.GaugesTable())
	return roll.Load(tables, name, seed)
}
