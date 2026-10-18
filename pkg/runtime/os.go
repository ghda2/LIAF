package runtime

import (
	"bytes"
	"encoding/csv"
	"os"
	"os/exec"
	"strings"
)

// CliArgs devolve a lista de argumentos passados via linha de comando (excluindo o nome do binario).
func CliArgs() *List[string] {
	args := os.Args
	if len(args) <= 1 {
		return &List[string]{Items: []string{}}
	}
	items := make([]string, len(args)-1)
	copy(items, args[1:])
	return &List[string]{Items: items}
}

// ExitProcess termina o processo imediatamente com o codigo fornecido.
func ExitProcess(code int64) {
	os.Exit(int(code))
}

// FsMkdirAll cria a arvore de diretorios fornecida.
func FsMkdirAll(path string) Result[bool, string] {
	err := os.MkdirAll(path, 0755)
	if err != nil {
		return Err[bool, string](err.Error())
	}
	return Ok[bool, string](true)
}

// FsRemoveAll remove o arquivo ou diretorio recursivamente.
func FsRemoveAll(path string) Result[bool, string] {
	err := os.RemoveAll(path)
	if err != nil {
		return Err[bool, string](err.Error())
	}
	return Ok[bool, string](true)
}

// FsListDir lista os nomes dos arquivos/pastas dentro de um diretorio.
func FsListDir(dirPath string) Result[*List[string], string] {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return Err[*List[string], string](err.Error())
	}
	items := make([]string, 0, len(entries))
	for _, e := range entries {
		items = append(items, e.Name())
	}
	return Ok[*List[string], string](&List[string]{Items: items})
}

// FsIsDir verifica se o caminho especificado e um diretorio.
func FsIsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FsFileSize devolve o tamanho em bytes do arquivo.
func FsFileSize(path string) Result[int64, string] {
	info, err := os.Stat(path)
	if err != nil {
		return Err[int64, string](err.Error())
	}
	return Ok[int64, string](info.Size())
}

// OsExec executa um comando externo com argumentos e captura sua saida padrao.
func OsExec(cmdName string, args *List[string]) Result[string, string] {
	var a []string
	if args != nil {
		a = args.Items
	}
	cmd := exec.Command(cmdName, a...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr == "" {
			errStr = err.Error()
		}
		return Err[string, string](errStr)
	}
	return Ok[string, string](stdout.String())
}

// OsExecStatus executa um comando e devolve o status/codigo de saida numerico.
func OsExecStatus(cmdName string, args *List[string]) int64 {
	var a []string
	if args != nil {
		a = args.Items
	}
	cmd := exec.Command(cmdName, a...)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return int64(exitErr.ExitCode())
		}
		return -1
	}
	return 0
}

// CsvParse analisa uma string CSV e devolve uma lista de linhas com campos.
func CsvParse(content string) Result[*List[*List[string]], string] {
	r := csv.NewReader(strings.NewReader(content))
	r.FieldsPerRecord = -1 // campos variaveis permitidos
	records, err := r.ReadAll()
	if err != nil {
		return Err[*List[*List[string]], string](err.Error())
	}
	rows := make([]*List[string], 0, len(records))
	for _, rec := range records {
		rows = append(rows, &List[string]{Items: rec})
	}
	return Ok[*List[*List[string]], string](&List[*List[string]]{Items: rows})
}

// CsvEncode serializa uma lista de linhas em formato CSV.
func CsvEncode(rows *List[*List[string]]) string {
	if rows == nil || len(rows.Items) == 0 {
		return ""
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, row := range rows.Items {
		if row != nil {
			_ = w.Write(row.Items)
		}
	}
	w.Flush()
	return buf.String()
}
