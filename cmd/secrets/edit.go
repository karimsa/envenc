package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/karimsa/secrets"
	"github.com/urfave/cli/v2"
)

var cmdEdit = &cli.Command{
	Name:  "edit",
	Usage: "Edit a file with encrypted values",
	Flags: []cli.Flag{
		inFlag,
		formatFlag,
		strategyFlag,
		passphraseFlag,
		keyFlag,
		keyFileFlag,
		&cli.StringFlag{
			Name:    "editor",
			Usage:   "Text editor to open for temporary file",
			EnvVars: []string{"EDITOR"},
			Value:   "vi",
		},
		flagLogLevel,
	},
	Action: func(ctx *cli.Context) error {
		format := ctx.String("format")
		inPath := ctx.String("in")
		editor := ctx.String("editor")

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

		envFile, err := secrets.Open(secrets.OpenEnvOptions{
			Format:      format,
			Reader:      inFile,
			Cipher:      cipher,
			SecurePaths: securePaths,
		})
		if err != nil {
			return err
		}

		// Create temporary version for user edits
		tmp, err := ioutil.TempFile("/tmp", "*")
		if err != nil {
			return fmt.Errorf("failed to create temporary file for editing: %s", err)
		}
		defer os.Remove(tmp.Name())

		buff, err := envFile.UnsafeRawExport(format)
		if err != nil {
			return err
		}

		_, err = tmp.Write(buff)
		if err != nil {
			return err
		}

		err = tmp.Sync()
		if err != nil {
			return err
		}

		cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s %s", editor, tmp.Name()))
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return err
		}

		editFile, err := os.OpenFile(tmp.Name(), os.O_RDONLY, 0)
		if err != nil {
			return err
		}

		err = envFile.UpdateFrom(format, editFile)
		if err != nil {
			return err
		}

		return envFile.ExportFile(format, inPath, os.O_RDWR)
	},
}
