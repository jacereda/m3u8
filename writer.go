package m3u8

/*
 Part of M3U8 parser & generator library.

 Copyleft Alexander I.Grafov aka Axel <grafov@gmail.com>
 Library licensed under GPLv3

 ॐ तारे तुत्तारे तुरे स्व
*/

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
)

func version(ver *uint8, newver uint8) {
	if *ver < newver {
		*ver = newver
	}
}

func strver(ver uint8) string {
	return strconv.FormatUint(uint64(ver), 10)
}

func NewFixedPlaylist() *FixedPlaylist {
	p := new(FixedPlaylist)
	p.ver = minver
	p.TargetDuration = 0
	return p
}

func NewFixedIFramesPlaylist() *FixedPlaylist {
	p := NewFixedPlaylist()
	p.iframes = true
	return p
}

func (p *FixedPlaylist) AddSegment(segment Segment) {
	p.Segments = append(p.Segments, segment)
	if segment.Offset != 0 {
		version(&p.ver, 4)
	}
	if segment.Key != nil { // due section 7
		version(&p.ver, 5)
	}
	if p.TargetDuration < segment.Duration {
		p.TargetDuration = segment.Duration
	}
}

func (p *FixedPlaylist) Buffer() *bytes.Buffer {
	var buf bytes.Buffer

	buf.WriteString("#EXTM3U\n")
	buf.WriteString("#EXT-X-TARGETDURATION:")
	buf.WriteString(strconv.FormatInt(int64(math.Ceil(p.TargetDuration)), 10))
	buf.WriteRune('\n')
	buf.WriteString("#EXT-X-VERSION:")
	buf.WriteString(strver(p.ver))
	buf.WriteRune('\n')
	//buf.WriteString("#EXT-X-ALLOW-CACHE:YES\n")
	buf.WriteString("#EXT-X-MEDIA-SEQUENCE:0\n")
	buf.WriteString("#EXT-X-PLAYLIST-TYPE:VOD\n")
	if p.iframes {
		buf.WriteString("#EXT-X-I-FRAMES-ONLY\n")
	}

	for _, s := range p.Segments {
		if s.Key != nil {
			buf.WriteString("#EXT-X-KEY:")
			buf.WriteString("METHOD=")
			buf.WriteString(s.Key.Method)
			buf.WriteString(",URI=")
			buf.WriteString(s.Key.URI)
			if s.Key.IV != "" {
				buf.WriteString(",IV=")
				buf.WriteString(s.Key.IV)
			}
			buf.WriteRune('\n')
		}
		buf.WriteString("#EXTINF:")
		buf.WriteString(strconv.FormatFloat(s.Duration, 'f', 3, 64))
		buf.WriteString(",\t\n")
		if s.Size != 0 {
			buf.WriteString(fmt.Sprintf("#EXT-X-BYTERANGE:%v@%v\n", s.Size, s.Offset))
		}
		buf.WriteString(s.URI)
		if p.SID != "" {
			buf.WriteRune('?')
			buf.WriteString(p.SID)
		}
		buf.WriteString("\n")
	}

	buf.WriteString("#EXT-X-ENDLIST\n")

	return &buf
}

func NewVariantPlaylist() *VariantPlaylist {
	p := new(VariantPlaylist)
	p.ver = minver
	return p
}

func (p *VariantPlaylist) AddVariant(variant Variant) {
	p.Variants = append(p.Variants, variant)
}

func (p *VariantPlaylist) Buffer() *bytes.Buffer {
	var buf bytes.Buffer

	buf.WriteString("#EXTM3U\n#EXT-X-VERSION:")
	buf.WriteString(strver(p.ver))
	buf.WriteRune('\n')

	for _, pl := range p.Variants {
		buf.WriteString("#EXT-X-STREAM-INF:PROGRAM-ID=")
		buf.WriteString(strconv.FormatUint(uint64(pl.ProgramId), 10))
		buf.WriteString(",BANDWIDTH=")
		buf.WriteString(strconv.FormatUint(uint64(pl.Bandwidth), 10))
		if pl.Codecs != "" {
			buf.WriteString(",CODECS=")
			buf.WriteString(pl.Codecs)
		}
		if pl.Resolution != "" {
			buf.WriteString(",RESOLUTION=\"")
			buf.WriteString(pl.Resolution)
			buf.WriteRune('"')
		}
		buf.WriteRune('\n')
		buf.WriteString(pl.URI)
		if p.SID != "" {
			buf.WriteRune('?')
			buf.WriteString(p.SID)
		}
		buf.WriteRune('\n')
	}

	return &buf
}

func NewSlidingPlaylist(winsize uint16) *SlidingPlaylist {
	p := new(SlidingPlaylist)
	p.ver = minver
	p.TargetDuration = 0
	p.SeqNo = 0
	p.winsize = winsize
	p.Segments = make(chan Segment, winsize*2) // TODO множитель в конфиг
	return p
}

func (p *SlidingPlaylist) AddSegment(segment Segment) error {
	if uint16(len(p.Segments)) >= p.winsize*2-1 {
		return errors.New("segments channel is full")
	}
	p.Segments <- segment
	if segment.Key != nil && segment.Key.Method != "" { // due section 7
		version(&p.ver, 5)
	}
	if p.TargetDuration < segment.Duration {
		p.TargetDuration = segment.Duration
	}
	return nil
}

func (p *SlidingPlaylist) Buffer() *bytes.Buffer {
	var buf bytes.Buffer
	var key *Key

	if len(p.Segments) == 0 && p.cache.Len() > 0 {
		return &p.cache
	}

	buf.WriteString("#EXTM3U\n#EXT-X-VERSION:")
	buf.WriteString(strver(p.ver))
	buf.WriteRune('\n')
	buf.WriteString("#EXT-X-ALLOW-CACHE:NO\n")
	buf.WriteString("#EXT-X-TARGETDURATION:")
	buf.WriteString(strconv.FormatFloat(p.TargetDuration, 'f', 2, 64))
	buf.WriteRune('\n')
	buf.WriteString("#EXT-X-MEDIA-SEQUENCE:")
	buf.WriteString(strconv.FormatUint(p.SeqNo, 10))
	buf.WriteRune('\n')
	p.SeqNo++

	for i := 0; i <= len(p.Segments); i++ {
		select {
		case seg := <-p.Segments:
			key = nil
			if seg.Key != nil {
				key = seg.Key
			} else {
				if p.key != nil {
					key = p.key
				}
			}
			if key != nil {
				buf.WriteString("#EXT-X-KEY:")
				buf.WriteString("METHOD=")
				buf.WriteString(key.Method)
				buf.WriteString(",URI=")
				buf.WriteString(key.URI)
				if key.IV != "" {
					buf.WriteString(",IV=")
					buf.WriteString(key.IV)
				}
				buf.WriteRune('\n')
			}
			if p.wv != nil {
				if p.wv.CypherVersion != "" {
					buf.WriteString("#WV-CYPHER-VERSION:")
					buf.WriteString(p.wv.CypherVersion)
					buf.WriteRune('\n')
				}
				if p.wv.ECM != "" {
					buf.WriteString("#WV-ECM:")
					buf.WriteString(p.wv.ECM)
					buf.WriteRune('\n')
				}
			}
			buf.WriteString("#EXTINF:")
			buf.WriteString(strconv.FormatFloat(seg.Duration, 'f', 3, 64))
			buf.WriteString(",\t\n")
			buf.WriteString(seg.URI)
			if p.SID != "" {
				buf.WriteRune('?')
				buf.WriteString(p.SID)
			}
			buf.WriteString("\n")
			// TODO key
		default:
		}
	}
	p.cache = buf
	return &buf
}

func (p *SlidingPlaylist) BufferEnd() *bytes.Buffer {
	var buf bytes.Buffer

	buf.WriteString("#EXT-X-ENDLIST\n")

	return &buf
}

func (p *SlidingPlaylist) SetKey(key *Key) {
	p.key = key
}

func (p *SlidingPlaylist) SetWV(wv *WV) {
	p.wv = wv
}
