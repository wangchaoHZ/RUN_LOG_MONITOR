package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Telnet struct {
		IP   string `yaml:"ip"`
		Port int    `yaml:"port"`
	} `yaml:"telnet"`
	Log struct {
		Dir      string `yaml:"dir"`
		KeepDays int    `yaml:"keep_days"`
	} `yaml:"log"`
}

func loadConfig(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}
	return &cfg
}

func logFileName(dir string, t time.Time) string {
	return filepath.Join(dir, fmt.Sprintf("runlog_%s.log", t.Format("20060102")))
}

func cleanOldLogs(dir string, keepDays int) {
	files, _ := filepath.Glob(filepath.Join(dir, "runlog_*.log"))
	cutoff := time.Now().AddDate(0, 0, -keepDays)

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(f)
		}
	}
}

func runTelnetMonitor(ctx context.Context, cfg *Config) {
	addr := fmt.Sprintf("%s:%d", cfg.Telnet.IP, cfg.Telnet.Port)
	_ = os.MkdirAll(cfg.Log.Dir, 0755)

	var logFile *os.File
	var currentDay = time.Now().Day()

	openLog := func() {
		if logFile != nil {
			logFile.Close()
		}
		name := logFileName(cfg.Log.Dir, time.Now())
		f, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		logFile = f
		cleanOldLogs(cfg.Log.Dir, cfg.Log.KeepDays)
	}

	openLog()

	for {
		select {
		case <-ctx.Done():
			if logFile != nil {
				logFile.Close()
			}
			fmt.Println("Telnet monitor exit.")
			return
		default:
		}

		fmt.Println("Connecting to", addr)
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			fmt.Println("Connect failed, retry in 3s:", err)
			time.Sleep(3 * time.Second)
			continue
		}

		fmt.Println("Connected.")
		reader := bufio.NewReader(conn)

		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), "use of closed") {
					fmt.Println("Disconnected, reconnecting...")
					conn.Close()
					break
				}
				fmt.Println("Read error:", err)
				conn.Close()
				break
			}

			now := time.Now()
			if now.Day() != currentDay {
				currentDay = now.Day()
				openLog()
			}

			line = strings.TrimRight(line, "\r\n")
			out := fmt.Sprintf("[%s] %s\n", now.Format("2006-01-02 15:04:05"), line)
			fmt.Print(out)
			logFile.WriteString(out)
		}

		time.Sleep(3 * time.Second)
	}
}

func main() {
	cfg := loadConfig("config.yaml")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runTelnetMonitor(ctx, cfg)
}
