package network

import "fmt"

var (
	prefix = "dev-"
	suffix = ""

	major = 0
	minor = 0
	patch = 1

	APIVersion = fmt.Sprintf("%s%d.%d.%d%s", prefix, major, minor, patch, suffix)
)
