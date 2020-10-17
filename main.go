package main

import (
	sdk "github.com/hashicorp/waypoint-plugin-sdk"
	"github.com/tcnksm/waypoint-plugin-kustomize/platform"
)

func main() {
	sdk.Main(sdk.WithComponents(
		&platform.Platform{},
	))
}
