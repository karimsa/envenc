package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/karimsa/secrets"
	"github.com/karimsa/secrets/internal/encrypt"
	"github.com/karimsa/secrets/internal/logger"
	"github.com/urfave/cli/v2"
)

var (
	inFlag = &cli.PathFlag{
		Name:      "in",
		Aliases:   []string{"i"},
		Usage:     "Path to the input file",
		Required:  true,
		TakesFile: true,
	}
	outFlag = &cli.PathFlag{
		Name:      "out",
		Aliases:   []string{"o"},
		Usage:     "Path to the output file",
		Required:  true,
		TakesFile: true,
	}
	formatFlag = &cli.StringFlag{
		Name:    "format",
		Aliases: []string{"f"},
		Usage:   "Format of the input and output files (json, yaml, dotenv)",
		Value:   "",
	}
	strategyFlag = &cli.StringFlag{
		Name:    "strategy",
		Aliases: []string{"s"},
		Usage:   "Encryption/decryption type (symmetric, asymmetric, or keyring)",
		Value:   "symmetric",
	}
	passphraseFlag = &cli.StringFlag{
		Name:    "unsafe-passphrase",
		Usage:   "Unsafely pass the passphrase for symmetric encryption",
		Value:   "",
		EnvVars: []string{"PASSPHRASE"},
	}
	keyFlag = &cli.StringSliceFlag{
		Name:     "key",
		Aliases:  []string{"k"},
		Usage:    "Target key path to find secure value",
		Required: true,
	}
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

func getCipher(ctx *cli.Context) (secrets.SimpleCipher, error) {
	strategy := ctx.String("strategy")

	if strategy == "symmetric" {
		// 1) Read from flags + 2) Will read from 'ENVENC_PASSPHRASE' env variable
		if pass := ctx.String("unsafe-passphrase"); len(pass) != 0 {
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
		Name:  "secrets",
		Usage: "Manage secrets in config files.",
		Commands: []*cli.Command{
			cmdEncrypt,
			cmdDecrypt,
			cmdEdit,
		},
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Karim Alibhai",
				Email: "karim@alibhai.co",
			},
		},
		Copyright: "(C) 2020-present Karim Alibhai",
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
