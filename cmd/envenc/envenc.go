package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/karimsa/envenc"
	"github.com/karimsa/envenc/internal/encrypt"
	"github.com/urfave/cli"
)

func getFormatFromPath(path string) string {
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

func main() {
	app := &cli.App{
		Name:  "envenc",
		Usage: "Manage secrets in config files.",
		Commands: []cli.Command{
			cmdEncrypt,
			cmdDecrypt,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
