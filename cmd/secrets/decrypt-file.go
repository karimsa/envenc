package main

import (
	"fmt"
	"os"
	"io/fs"
	"io/ioutil"

	"github.com/urfave/cli/v2"
)

var cmdDecryptFile = &cli.Command{
	Name:    "decrypt-file",
	Usage:   "Decrypt entire file",
	Flags: []cli.Flag{
		inFlag,
		outFlag,
		strategyFlag,
		passphraseFlag,
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		inPath := ctx.String("in")

		inFile, err := os.OpenFile(inPath, os.O_RDONLY, 0)
		if err != nil {
			return err
		}

		cipher, err := getCipher(ctx)
		if err != nil {
			return err
		}

		rawInput, err := ioutil.ReadAll(inFile)
		if err != nil {
			return err
		}

		decryptedData, err := cipher.Decrypt(string(rawInput))
		if err != nil {
			return err
		}

		outPath := ctx.String("out")
		switch outPath {
		case "/dev/stdout":
			fmt.Printf(decryptedData)
			return nil
		case "/dev/stderr":
			fmt.Fprintf(os.Stderr, decryptedData)
			return nil
		}

		// For in-place edits, overwrite the file
		outFileMode := os.O_EXCL
		if outPath == inPath {
			outFileMode = os.O_WRONLY | os.O_TRUNC
		}
		return ioutil.WriteFile(outPath, []byte(decryptedData), fs.FileMode(outFileMode))
	},
}
