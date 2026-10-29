// main.go - kwtsms-cli entry point.
// Delegates immediately to cmd.Execute() following cobra conventions.
// Related files: cmd/root.go
package main

import "github.com/kwtsms/kwtsms-cli/cmd"

func main() {
	cmd.Execute()
}
