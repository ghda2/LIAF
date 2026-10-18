package runtime

import (
	"fmt"
	"math"
	"os"
)

func IntFromFloat(x float64) int64 {
	return int64(x)
}

func AddInt(a, b int64) int64 {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		fmt.Fprintln(os.Stderr, "liaf: overflow na adicao de inteiros")
		os.Exit(1)
	}
	return a + b
}

func SubInt(a, b int64) int64 {
	if (b < 0 && a > math.MaxInt64+b) || (b > 0 && a < math.MinInt64+b) {
		fmt.Fprintln(os.Stderr, "liaf: overflow na subtracao de inteiros")
		os.Exit(1)
	}
	return a - b
}

func MulInt(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	if (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
		fmt.Fprintln(os.Stderr, "liaf: overflow na multiplicacao de inteiros")
		os.Exit(1)
	}
	res := a * b
	if res/b != a {
		fmt.Fprintln(os.Stderr, "liaf: overflow na multiplicacao de inteiros")
		os.Exit(1)
	}
	return res
}

func NegInt(a int64) int64 {
	if a == math.MinInt64 {
		fmt.Fprintln(os.Stderr, "liaf: overflow na negacao de inteiros")
		os.Exit(1)
	}
	return -a
}

func AbsInt(a int64) int64 {
	if a == math.MinInt64 {
		fmt.Fprintln(os.Stderr, "liaf: overflow no abs de inteiros")
		os.Exit(1)
	}
	if a < 0 {
		return -a
	}
	return a
}

func AbsFloat(a float64) float64 {
	return math.Abs(a)
}

func DivInt(a, b int64) Result[int64, string] {
	if b == 0 {
		return Err[int64, string]("div: divisao por zero")
	}
	if a == math.MinInt64 && b == -1 {
		fmt.Fprintln(os.Stderr, "liaf: overflow na divisao de inteiros")
		os.Exit(1)
	}
	return Ok[int64, string](a / b)
}

func ModInt(a, b int64) Result[int64, string] {
	if b == 0 {
		return Err[int64, string]("mod: divisao por zero")
	}
	if a == math.MinInt64 && b == -1 {
		return Ok[int64, string](0)
	}
	return Ok[int64, string](a % b)
}

func MinInt(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func MinFloat(a, b float64) float64 {
	return math.Min(a, b)
}

func MaxFloat(a, b float64) float64 {
	return math.Max(a, b)
}

func PowInt(base, exp int64) int64 {
	if exp < 0 {
		fmt.Fprintln(os.Stderr, "liaf: expoente negativo em pow")
		os.Exit(1)
	}
	res := int64(1)
	for exp > 0 {
		if exp&1 == 1 {
			res = MulInt(res, base)
		}
		exp >>= 1
		if exp > 0 {
			base = MulInt(base, base)
		}
	}
	return res
}

func PowFloat(base, exp float64) float64 {
	return math.Pow(base, exp)
}

func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

func Floor(x float64) float64 {
	return math.Floor(x)
}

func Ceil(x float64) float64 {
	return math.Ceil(x)
}

func Round(x float64) float64 {
	return math.Round(x)
}
