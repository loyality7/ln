// Command ln draws from the command line: 3D scenes to video or pictures, 2D drawings, glowing light, mesh tools,
// and any video redrawn as line art. Run `ln` for the list of commands, and `ln help <command>` for each.
package main

import (
	"os"

	"github.com/loyality7/ln/cli"
)

func main() { os.Exit(cli.Main(os.Args[1:])) }
