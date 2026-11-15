package maths

import (
	"fmt"
	"math"
)

// Portion is a part of a whole, said one of two ways: Percent(50) for
// fifty percent, or Share(0.5) for the same thing as a fraction. It is
// made inline, so it carries any complaint privately; the constructor
// that receives it surfaces the complaint through Share.
type Portion struct {
	share     float64
	complaint error
}

// Percent is a portion said as a number out of a hundred: Percent(50)
// is half, and Percent(99.9) is 99.9 percent. Zero, and any amount from
// one to a hundred, is a percent. An amount above zero and below one,
// such as 0.999, reads as a fraction someone meant to write as a
// percent, and is refused; Share says fractions. Below zero, past a
// hundred, or no number at all is no percent either.
func Percent(amount float64) Portion {
	switch {
	case math.IsNaN(amount) || math.IsInf(amount, 0):
		return Portion{complaint: fmt.Errorf(
			"maths: Percent(%v) is no number", amount)}
	case amount > 0 && amount < 1:
		return Portion{complaint: fmt.Errorf(
			"maths: Percent(%v) reads as a fraction: write "+
				"Percent(%v) for the percent, or Share(%v) "+
				"for the fraction", amount, amount*100, amount)}
	case amount < 0 || amount > 100:
		return Portion{complaint: fmt.Errorf(
			"maths: Percent(%v) is outside nought to a hundred",
			amount)}
	}
	return Portion{share: amount / 100}
}

// Share is a portion said as a fraction of one: Share(0.5) is half, and
// small shares like Share(0.006) are at home here. Outside nought to
// one, or no number at all, is no share.
func Share(amount float64) Portion {
	if math.IsNaN(amount) || amount < 0 || amount > 1 {
		return Portion{complaint: fmt.Errorf(
			"maths: Share(%v) is outside nought to one", amount)}
	}
	return Portion{share: amount}
}

// Share is the portion as a fraction of one, or the complaint its
// making carried: a constructor taking a Portion surfaces this.
func (portion Portion) Share() (float64, error) {
	return portion.share, portion.complaint
}
