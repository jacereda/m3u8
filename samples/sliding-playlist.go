package main

import (
	"github.com/jacereda/m3u8"
)

func main() {
	s := m3u8.NewSlidingPlaylist(4)
	for i := 0; i < 10; i++ {
		s.AddSegment(m3u8.Segment{URI: "sample.ts", Duration: 5.0})
		if i%5 == 1 {
			print(s.Buffer().String())
			print("\n")
		}
	}
	print(s.Buffer().String())
	print("\n")
	print(s.Buffer().String())
	print("\n")
}
