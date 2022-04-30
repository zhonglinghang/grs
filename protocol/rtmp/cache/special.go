package cache

import (
	"bytes"
	"log"

	"github.com/zhonglinghang/grs/av"
	"github.com/zhonglinghang/grs/protocol/amf"
)

const (
	SetDataFrame string = "@setDataFrame"
	OnMetaData   string = "onMetaData"
)

var setFrameFrame []byte

func init() {
	b := bytes.NewBuffer(nil)
	encoder := &amf.Encoder{}
	if _, err := encoder.Encode(b, SetDataFrame, amf.AMF0); err != nil {
		log.Fatal(err)
	}
	setFrameFrame = b.Bytes()
}

type SpecialCache struct {
	full bool
	p    *av.Packet
}

func NewSpecialCache() *SpecialCache {
	return &SpecialCache{}
}

func (specialCache *SpecialCache) Write(p *av.Packet) {
	specialCache.p = p
	specialCache.full = true
}

func (specialCache *SpecialCache) Send(w av.WriterCloser) error {
	if !specialCache.full {
		return nil
	}

	// demux in hls will change p.Data, only send a copy here
	newPacket := *specialCache.p
	return w.Write(&newPacket)
}
