package config

import (
	"hands/define"
	"slices"
)

var Config *define.Config

func IsValidInterface(ifName string) bool {
	return slices.Contains(Config.AvailableInterfaces, ifName)
}
