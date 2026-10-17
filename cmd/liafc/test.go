package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	rt "liaf/pkg/runtime"
)

func runTest(args []string) {
	target := "."
	asJSON := false

	var filteredArgs []string
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		} else {
			filteredArgs = append(filteredArgs, a)
		}
	}
	if len(filteredArgs) > 0 {
		target = filteredArgs[0]
	}

	var testFiles []string
	info, err := os.Stat(target)
	if err != nil {
		if asJSON {
			fmt.Printf(`{"status":"error","error":%q}`+"\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "Erro ao acessar '%s': %v\n", target, err)
		}
		os.Exit(1)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(target)
		if err != nil {
			if asJSON {
				fmt.Printf(`{"status":"error","error":%q}`+"\n", err.Error())
			} else {
				fmt.Fprintf(os.Stderr, "Erro ao ler diretório '%s': %v\n", target, err)
			}
			os.Exit(1)
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".liaf") {
				if strings.HasSuffix(e.Name(), "_test.liaf") || strings.HasPrefix(e.Name(), "test_") {
					testFiles = append(testFiles, filepath.Join(target, e.Name()))
				}
			}
		}
		if len(testFiles) == 0 {
			// Se não houver _test.liaf específico, roda os .liaf da pasta
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".liaf") {
					testFiles = append(testFiles, filepath.Join(target, e.Name()))
				}
			}
		}
	} else {
		testFiles = append(testFiles, target)
	}

	if len(testFiles) == 0 {
		if asJSON {
			fmt.Println(`{"passed":0,"failed":0,"total":0,"duration_ms":0,"tests":[]}`)
		} else {
			fmt.Println("Nenhum arquivo de teste LIAF (.liaf) encontrado.")
		}
		return
	}

	if !asJSON {
		fmt.Printf("🧪 Executando suíte de testes LIAF (%d arquivo(s))...\n\n", len(testFiles))
	}

	totalPassed := 0
	totalFailed := 0
	startAll := time.Now()

	var allReportItems []rt.TestReportItem

	for _, file := range testFiles {
		start := time.Now()
		_, allErrors, generatedCode := parseAndCheck(file)
		if len(allErrors) > 0 {
			var errMsgs []string
			for _, err := range allErrors {
				errMsgs = append(errMsgs, err.String())
			}
			fullErrMsg := strings.Join(errMsgs, "; ")
			if !asJSON {
				fmt.Printf("❌ FAIL %s (erros de tipo/sintaxe):\n", filepath.Base(file))
				for _, m := range errMsgs {
					fmt.Printf("  %s\n", m)
				}
			}
			totalFailed++
			allReportItems = append(allReportItems, rt.TestReportItem{
				Name:       filepath.Base(file),
				Status:     "fail",
				Error:      fullErrMsg,
				DurationMs: float64(time.Since(start).Microseconds()) / 1000.0,
			})
			continue
		}

		tmpDir, err := os.MkdirTemp("", "liaf_test_*")
		if err != nil {
			if !asJSON {
				fmt.Printf("❌ FAIL %s (falha ao criar pasta temporária): %v\n", filepath.Base(file), err)
			}
			totalFailed++
			allReportItems = append(allReportItems, rt.TestReportItem{
				Name:       filepath.Base(file),
				Status:     "fail",
				Error:      err.Error(),
				DurationMs: float64(time.Since(start).Microseconds()) / 1000.0,
			})
			continue
		}

		binPath := filepath.Join(tmpDir, "test_bin.exe")
		if err := compileGo(generatedCode, "", binPath, file); err != nil {
			os.RemoveAll(tmpDir)
			if !asJSON {
				fmt.Printf("❌ FAIL %s (compilação falhou): %v\n", filepath.Base(file), err)
			}
			totalFailed++
			allReportItems = append(allReportItems, rt.TestReportItem{
				Name:       filepath.Base(file),
				Status:     "fail",
				Error:      "compilação falhou: " + err.Error(),
				DurationMs: float64(time.Since(start).Microseconds()) / 1000.0,
			})
			continue
		}

		cmd := exec.Command(binPath)
		cmd.Env = append(os.Environ(), "LIAF_TEST_MODE=1")
		if asJSON {
			cmd.Env = append(cmd.Env, "LIAF_TEST_JSON=1")
		}
		out, runErr := cmd.CombinedOutput()
		dur := time.Since(start)
		os.RemoveAll(tmpDir)

		if asJSON {
			var rep rt.TestReport
			if jsonErr := json.Unmarshal(out, &rep); jsonErr == nil {
				totalPassed += rep.Passed
				totalFailed += rep.Failed
				for _, ti := range rep.Tests {
					ti.Name = fmt.Sprintf("[%s] %s", filepath.Base(file), ti.Name)
					allReportItems = append(allReportItems, ti)
				}
			} else {
				if runErr != nil {
					totalFailed++
					allReportItems = append(allReportItems, rt.TestReportItem{
						Name:       filepath.Base(file),
						Status:     "fail",
						Error:      string(out),
						DurationMs: float64(dur.Microseconds()) / 1000.0,
					})
				} else {
					totalPassed++
					allReportItems = append(allReportItems, rt.TestReportItem{
						Name:       filepath.Base(file),
						Status:     "pass",
						DurationMs: float64(dur.Microseconds()) / 1000.0,
					})
				}
			}
		} else {
			if runErr != nil {
				fmt.Printf("❌ FAIL %s (%.2fs)\n", filepath.Base(file), dur.Seconds())
				if len(out) > 0 {
					fmt.Println(string(out))
				}
				totalFailed++
			} else {
				fmt.Printf("✓ PASS %s (%.2fs)\n", filepath.Base(file), dur.Seconds())
				if len(out) > 0 {
					fmt.Print(string(out))
				}
				totalPassed++
			}
		}
	}

	if asJSON {
		finalReport := rt.TestReport{
			Passed:     totalPassed,
			Failed:     totalFailed,
			Total:      totalPassed + totalFailed,
			DurationMs: float64(time.Since(startAll).Microseconds()) / 1000.0,
			Tests:      allReportItems,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(finalReport)
	} else {
		fmt.Printf("\nResultado da suíte: %d passaram, %d falharam (Total: %d, Tempo: %.2fs)\n",
			totalPassed, totalFailed, totalPassed+totalFailed, time.Since(startAll).Seconds())
	}

	if totalFailed > 0 {
		os.Exit(1)
	}
}
