package main

import (
	"fmt"
	"os"

	"github.com/daimatz/gojvm/pkg/classfile"
	"github.com/daimatz/gojvm/pkg/vm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: gojvm <classfile>\n")
		os.Exit(1)
	}

	filename := os.Args[1]

	f, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	cf, err := classfile.Parse(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing class file: %v\n", err)
		os.Exit(1)
	}

	v := vm.NewVM(cf)

	if err := v.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing: %v\n", err)
		os.Exit(1)
	}
}
