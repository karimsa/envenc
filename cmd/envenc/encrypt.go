package main

import (
	"fmt"
	"os"

	"github.com/karimsa/envenc"
	"github.com/urfave/cli"
)

var cmdEncrypt = cli.Command{
	Name:  "encrypt",
	Usage: "Encrypt values in a given file",
	Flags: []cli.Flag{
		inFlag,
		outFlag,
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

		logLevel, err := getLogLevel(ctx)
		if err != nil {
			return err
		}

		securePaths := make(map[string]bool, len(inputPaths))
		for _, path := range inputPaths {
			securePaths[path] = true
		}

		envFile, err := envenc.New(envenc.NewEnvOptions{
			Format:      format,
			Reader:      inFile,
			Cipher:      cipher,
			LogLevel:    logLevel,
			SecurePaths: securePaths,
		})
		if err != nil {
			return err
		}

		buff, err := envFile.Export(format)
		if err != nil {
			return err
		}

		outPath := ctx.String("out")
		switch outPath {
		case "/dev/stdout":
			fmt.Printf(string(buff))
			return nil
		case "/dev/stderr":
			fmt.Fprintf(os.Stderr, string(buff))
			return nil
		}

		// For in-place edits, overwrite the file
		outFileMode := os.O_EXCL
		if outPath == inPath {
			outFileMode = os.O_WRONLY | os.O_TRUNC
		}
		return envFile.ExportFile(format, outPath, outFileMode)
	},
}
