package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/karimsa/envenc"
	"github.com/karimsa/envenc/internal/encrypt"
	"github.com/karimsa/envenc/internal/logger"
	"github.com/urfave/cli"
)

var (
	flagLogLevel = &cli.StringFlag{
		Name:  "log-level",
		Usage: "Increase logging verbosity (none, info, debug)",
		Value: "none",
	}
)

func getFormatFromPath(path string) string {
	if path[0] == '.' {
		return "dotenv"
	}

	ext := path[strings.LastIndexByte(path, '.'):]
	return ext
}

func getCipher(ctx *cli.Context) (envenc.SimpleCipher, error) {
	strategy := ctx.String("strategy")

	if strategy == "symmetric" {
		// 1) Read from flags
		if pass := ctx.String("unsafe-passphrase"); len(pass) != 0 {
			return encrypt.NewSymmetricCipher([]byte(pass)), nil
		}

		// 2) Read from env
		if pass := os.Getenv("ENVENC_PASSPHRASE"); len(pass) != 0 {
			return encrypt.NewSymmetricCipher([]byte(pass)), nil
		}

		// 3) Read from stdin
		fmt.Fprintf(os.Stderr, "Passphrase: ")
		pass, err := gopass.GetPasswdMasked()
		if err != nil {
			return nil, err
		}
		return encrypt.NewSymmetricCipher(pass), nil
	}

	return nil, fmt.Errorf("Unsupported strategy: %s", strategy)
}

func getLogLevel(ctx *cli.Context) (logger.LogLevel, error) {
	level := ctx.String("log-level")
	switch level {
	case "":
		fallthrough
	case "none":
		return logger.LevelNone, nil
	case "info":
		return logger.LevelInfo, nil
	case "debug":
		return logger.LevelDebug, nil
	default:
		return logger.LogLevel(-1), fmt.Errorf("Unrecognized log level: %s", level)
	}
}

func main() {
	app := &cli.App{
		Name:  "envenc",
		Usage: "Manage secrets in config files.",
		Commands: []cli.Command{
			cmdEncrypt,
			cmdDecrypt,
			cmdEdit,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
