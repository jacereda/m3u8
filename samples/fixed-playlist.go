package main

import (
	"github.com/jacereda/m3u8"
)

func main() {
	p := m3u8.NewFixedPlaylist()
	p.AddSegment(m3u8.Segment{URI: "test02.ts?111", Duration: 5.0})
	p.AddSegment(m3u8.Segment{URI: "test03.ts?111", Duration: 6.1})
	print(p.Buffer().String())
	print("\n")
}
