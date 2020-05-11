// +build tools

package build

// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

import (
	_ "golang.org/x/tools/cmd/goimports"
)
