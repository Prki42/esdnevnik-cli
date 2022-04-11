package main

import (
	"os"

	"github.com/Prki42/esdnevnik-cli/esdnevnik"
)

func main() {
	os.Exit(esdnevnik.CLI(os.Args[1:]))
}
