package main

import (
	"flag"

	"tcp-tunnel/lan"
	"tcp-tunnel/wan"
)

func main() {

	flag.Parse()
	args := flag.Args()

	cmd := args[0]

	if cmd == "lan" {
		lan.Start()
	}

	if cmd == "wan" {
		wan.Start()
	}

}
