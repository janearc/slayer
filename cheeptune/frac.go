package cheeptune

import (
	"fmt"
	"strconv"
	"strings"
)

// frac is a length in beats, held exactly.
//
// A bar has to add up to its meter, and three triplet eighths and a dotted
// quarter do that only in fractions. In floats they add up to 3.9999999 and a
// bar that is right gets refused, or a tolerance lets through one that is not.
type frac struct{ n, d int64 }

// mkfrac is n over d in lowest terms, with the sign on top.
func mkfrac(n, d int64) frac {
	if d < 0 {
		n, d = -n, -d
	}
	g := gcd(abs64(n), d)
	if g == 0 {
		return frac{0, 1}
	}
	return frac{n / g, d / g}
}

// add is the sum.
func (a frac) add(b frac) frac { return mkfrac(a.n*b.d+b.n*a.d, a.d*b.d) }

// float is the nearest float, for when exactness is done with.
func (a frac) float() float64 { return float64(a.n) / float64(a.d) }

// String is n, or n/d when it is not whole.
func (a frac) String() string {
	if a.d == 1 {
		return strconv.FormatInt(a.n, 10)
	}
	return fmt.Sprintf("%d/%d", a.n, a.d)
}

// parseFrac reads "3" or "3/4". Zero and negative lengths are refused here, so
// nothing downstream has to wonder about a note that ends before it starts.
func parseFrac(s string) (frac, error) {
	num, den, slash := strings.Cut(s, "/")
	n, err := strconv.ParseInt(num, 10, 64)
	if err != nil {
		return frac{}, fmt.Errorf("length %q is not a number or n/d", s)
	}
	d := int64(1)
	if slash {
		d, err = strconv.ParseInt(den, 10, 64)
		if err != nil || d == 0 {
			return frac{}, fmt.Errorf(
				"length %q is not a number or n/d", s)
		}
	}
	f := mkfrac(n, d)
	if f.n <= 0 {
		return frac{}, fmt.Errorf("length %q is not above nought", s)
	}
	return f, nil
}

// gcd is the greatest common divisor, by Euclid.
func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// abs64 is the magnitude.
func abs64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}
