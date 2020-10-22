package main

import (
	"fmt"
	"os"

	"github.com/karimsa/envenc"
	"github.com/urfave/cli"
)

var cmdDecrypt = cli.Command{
	Name:  "decrypt",
	Usage: "Decrypt values from a config file",
	Flags: []cli.Flag{
		inFlag,
		formatFlag,
		strategyFlag,
		passphraseFlag,
		keyFlag,
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		format := ctx.String("format")
		inPath := ctx.String("in")
		inputPaths := ctx.StringSlice("key")

		if format == "" {
			format = getFormatFromPath(inPath)
		}

		inFile, err := os.OpenFile(inPath, os.O_RDONLY, 0)
		if err != nil {
			return err
		}

		cipher, err := getCipher(ctx)
		if err != nil {
			return err
		}

		securePaths := make(map[string]bool, len(inputPaths))
		for _, path := range inputPaths {
			securePaths[path] = true
		}

		logLevel, err := getLogLevel(ctx)
		if err != nil {
			return err
		}

		envFile, err := envenc.Open(envenc.OpenEnvOptions{
			Format:      format,
			Reader:      inFile,
			Cipher:      cipher,
			SecurePaths: securePaths,
			LogLevel:    logLevel,
		})
		if err != nil {
			return err
		}

		buff, err := envFile.UnsafeRawExport(format)
		if err != nil {
			return err
		}

		// For decryption, always write to stderr
		fmt.Fprintf(os.Stderr, string(buff))

		return nil
	},
}
