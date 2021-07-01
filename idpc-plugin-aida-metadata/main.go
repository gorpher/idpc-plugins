//+build windows

package main

import (
	lib "github.com/gorpher/idpc-plugins/idpc-plugin-aida-metadata/lib"
)

var (
	Revision = "untracked"
	Version  = "v1.0.0"
)

func main() {
	lib.Revision = Revision
	lib.Version = Version
	lib.Do()
}
