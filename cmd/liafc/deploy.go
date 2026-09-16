package main

import (
	"context"
	"flag"
	"fmt"
	"liaf/pkg/deploy"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
func runDeploy(args []string) {
	if len(args) == 0 || args[0] != "caddy-bind" {
		fatal(fmt.Errorf("usage: liafc deploy caddy-bind --domain HOST --upstream HOST:PORT [--admin URL] [--server srv0]"))
	}
	f := flag.NewFlagSet("caddy-bind", flag.ExitOnError)
	domain := f.String("domain", "", "public host")
	upstream := f.String("upstream", "", "host:port")
	admin := f.String("admin", "http://localhost:2019", "admin URL")
	server := f.String("server", "srv0", "Caddy server name")
	f.Parse(args[1:])
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := deploy.BindCaddy(ctx, &http.Client{Timeout: 15 * time.Second}, *admin, *server, *domain, *upstream); err != nil {
		fatal(err)
	}
	fmt.Println("Caddy route configured")
}
func installService(args []string) {
	if len(args) == 0 || args[0] != "install" {
		fatal(fmt.Errorf("usage: liafc service install --name NAME --bin FILE [--workdir DIR] [--dry-run]"))
	}
	f := flag.NewFlagSet("service install", flag.ExitOnError)
	name := f.String("name", "", "service name")
	bin := f.String("bin", "", "executable")
	dir := f.String("workdir", "", "working directory")
	dry := f.Bool("dry-run", false, "print unit without installing")
	f.Parse(args[1:])
	if *bin == "" {
		fatal(fmt.Errorf("--bin is required"))
	}
	absolute, err := filepath.Abs(*bin)
	if err != nil {
		fatal(err)
	}
	if *dir == "" {
		*dir = filepath.Dir(absolute)
	}
	*dir, err = filepath.Abs(*dir)
	if err != nil {
		fatal(err)
	}
	unit, err := deploy.Unit(*name, absolute, *dir)
	if err != nil {
		fatal(err)
	}
	if *dry {
		fmt.Print(unit)
		return
	}
	if runtime.GOOS != "linux" {
		fatal(fmt.Errorf("systemd installation requires Linux; use --dry-run to inspect"))
	}
	if _, err = os.Stat(absolute); err != nil {
		fatal(err)
	}
	if err = os.WriteFile("/etc/systemd/system/"+*name+".service", []byte(unit), 0644); err != nil {
		fatal(err)
	}
	for _, a := range [][]string{{"daemon-reload"}, {"enable", "--now", *name + ".service"}} {
		cmd := exec.Command("systemctl", a...)
		if output, err := cmd.CombinedOutput(); err != nil {
			fatal(fmt.Errorf("systemctl: %w: %s", err, output))
		}
	}
	fmt.Println("Service installed and started:", *name)
}
