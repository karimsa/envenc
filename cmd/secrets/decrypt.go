package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/karimsa/secrets"
	"github.com/urfave/cli/v2"
)

var decOutFlag = &*outFlag

func init() {
	decOutFlag.Required = false
}

var cmdDecrypt = &cli.Command{
	Name:    "decrypt",
	Aliases: []string{"dec"},
	Usage:   "Decrypt values from a config file",
	Flags: []cli.Flag{
		inFlag,
		decOutFlag,
		formatFlag,
		strategyFlag,
		passphraseFlag,
		keyFlag,
		keyFileFlag,
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		format := ctx.String("format")
		inPath := ctx.String("in")
		outPath := ctx.String("out")

		securePaths, err := getInputPaths(ctx)
		if err != nil {
			return err
		}

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

		envFile, err := secrets.Open(secrets.OpenEnvOptions{
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

		switch outPath {
		case "":
			fallthrough
		case "/dev/stdout":
			fmt.Printf("%s\n", string(buff))
		case "/dev/stderr":
			fmt.Fprintf(os.Stderr, "%s\n", string(buff))
		default:
			return ioutil.WriteFile(outPath, buff, 0600)
		}

		return nil
	},
}
