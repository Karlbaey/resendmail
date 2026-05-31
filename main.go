package main

import (
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], defaultRuntimeDeps(os.Stdout, os.Stderr)))
}
