package codegen

import (
	"os"
	"testing"
)

// Os testes de execucao sobem os binarios gerados como servidores. Cada um e
// um executavel novo numa pasta temporaria, e escutar em todas as interfaces
// faria o firewall do Windows perguntar a cada rodada. Os processos filhos
// herdam LIAF_HOST e escutam so no loopback, que e onde os testes conectam.
func TestMain(m *testing.M) {
	os.Setenv("LIAF_HOST", "127.0.0.1")
	os.Exit(m.Run())
}
