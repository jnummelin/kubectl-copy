package main

import (
	"os"

	"github.com/jnummelin/kubectl-copy/internal/cmd"
	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func main() {
	streams := genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	if err := cmd.NewCopyCommand(streams).Execute(); err != nil {
		os.Exit(1)
	}
}
