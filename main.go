// Command localsend is a LocalSend client on NSGo: nearby devices found
// over multicast, files sent and received over the LocalSend protocol,
// and one app on macOS, Windows, Linux, iOS and Android.
package main

import (
	"github.com/oneo-me/nsgo"
	"github.com/oneo-me/nsgo/core"
)

func main() {
	nsgo.Run(Root,
		nsgo.WindowSize(1000, 720),
		nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 520, MinHeight: 560}),
	)
}
