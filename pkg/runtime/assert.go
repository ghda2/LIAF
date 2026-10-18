package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"
)

// Assert falha com panico se a condicao for falsa.
func Assert(cond bool, msg string) {
	if !cond {
		if msg == "" {
			msg = "assertion failed"
		}
		panic("ASSERTION_ERROR: " + msg)
	}
}

// AssertTrue atalho para assert(cond, "expected true, got false").
func AssertTrue(cond bool) {
	Assert(cond, "expected true, got false")
}

// AssertFalse atalho para assert(!cond, "expected false, got true").
func AssertFalse(cond bool) {
	Assert(!cond, "expected false, got true")
}

// AssertEq compara dois valores com igualdade estrutural profunda e equivalência numérica.
func AssertEq(a, b any, msg string) {
	if !equalValues(a, b) {
		detail := fmt.Sprintf("expected %v, got %v", b, a)
		if msg != "" {
			detail = msg + ": " + detail
		}
		panic("ASSERTION_ERROR: " + detail)
	}
}

func equalValues(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if !va.IsValid() || !vb.IsValid() {
		return false
	}
	if isIntKind(va.Kind()) && isIntKind(vb.Kind()) {
		return va.Int() == vb.Int()
	}
	if isUintKind(va.Kind()) && isUintKind(vb.Kind()) {
		return va.Uint() == vb.Uint()
	}
	if isIntKind(va.Kind()) && isUintKind(vb.Kind()) && va.Int() >= 0 {
		return uint64(va.Int()) == vb.Uint()
	}
	if isUintKind(va.Kind()) && isIntKind(vb.Kind()) && vb.Int() >= 0 {
		return va.Uint() == uint64(vb.Int())
	}
	if isFloatKind(va.Kind()) && isFloatKind(vb.Kind()) {
		return va.Float() == vb.Float()
	}
	return false
}

func isIntKind(k reflect.Kind) bool {
	return k >= reflect.Int && k <= reflect.Int64
}

func isUintKind(k reflect.Kind) bool {
	return k >= reflect.Uint && k <= reflect.Uint64
}

func isFloatKind(k reflect.Kind) bool {
	return k == reflect.Float32 || k == reflect.Float64
}

// TestCase representa um bloco de teste LIAF compilado.
type TestCase struct {
	Name string
	Fn   func()
}

type TestReportItem struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"` // "pass" ou "fail"
	Error      string  `json:"error,omitempty"`
	DurationMs float64 `json:"duration_ms"`
}

type TestReport struct {
	Passed     int              `json:"passed"`
	Failed     int              `json:"failed"`
	Total      int              `json:"total"`
	DurationMs float64          `json:"duration_ms"`
	Tests      []TestReportItem `json:"tests"`
}

// RunTests executa todos os blocos de teste registrados.
func RunTests(tests []TestCase) {
	report := TestReport{
		Total: len(tests),
		Tests: make([]TestReportItem, 0, len(tests)),
	}

	startAll := time.Now()
	asJSON := os.Getenv("LIAF_TEST_JSON") == "1"

	for _, tc := range tests {
		start := time.Now()
		var testErr error

		func() {
			defer func() {
				if r := recover(); r != nil {
					testErr = fmt.Errorf("%v", r)
				}
			}()
			tc.Fn()
		}()

		durMs := float64(time.Since(start).Microseconds()) / 1000.0

		if testErr != nil {
			report.Failed++
			report.Tests = append(report.Tests, TestReportItem{
				Name:       tc.Name,
				Status:     "fail",
				Error:      testErr.Error(),
				DurationMs: durMs,
			})
			if !asJSON {
				fmt.Printf("  ❌ FAIL: %s (%.2fms)\n     %s\n", tc.Name, durMs, testErr.Error())
			}
		} else {
			report.Passed++
			report.Tests = append(report.Tests, TestReportItem{
				Name:       tc.Name,
				Status:     "pass",
				DurationMs: durMs,
			})
			if !asJSON {
				fmt.Printf("  ✓ PASS: %s (%.2fms)\n", tc.Name, durMs)
			}
		}
	}

	report.DurationMs = float64(time.Since(startAll).Microseconds()) / 1000.0

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		fmt.Printf("\nResultado dos testes: %d passaram, %d falharam (tempo: %.2fms)\n",
			report.Passed, report.Failed, report.DurationMs)
	}

	if report.Failed > 0 {
		os.Exit(1)
	}
}
