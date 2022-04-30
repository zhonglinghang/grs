package rtmp

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zhonglinghang/grs/av"
	"github.com/zhonglinghang/grs/configure"
	"github.com/zhonglinghang/grs/container/flv"
	"github.com/zhonglinghang/grs/protocol/rtmp/cache"
	"github.com/zhonglinghang/grs/protocol/rtmp/core"
)

var (
	EmptyID = ""
)

type RtmpStream struct {
	streams *sync.Map
}

func NewRtmpStream() *RtmpStream {
	ret := &RtmpStream{
		streams: &sync.Map{},
	}
	go ret.CheckAlive()
	return ret
}

func (rs *RtmpStream) HandleReader(r av.ReaderCloser) {
	info := r.Info()
	log.Debugf("HandleReader: info[%v]", info)

	var stream *Stream
	i, ok := rs.streams.Load(info.Key)
	if stream, ok = i.(*Stream); ok {
		stream.TransStop()
		id := stream.ID()
		if id != EmptyID && id != info.UID {
			ns := NewStream()
			stream.Copy(ns)
			stream = ns
			rs.streams.Store(info.Key, ns)
		}
	} else {
		stream = NewStream()
		rs.streams.Store(info.Key, stream)
		stream.info = info
	}

	stream.AddReader(r)
}

func (rs *RtmpStream) HandleWriter(w av.WriterCloser) {
	info := w.Info()
	log.Debugf("HandleWriter: info[%v]", info)

	var s *Stream
	item, ok := rs.streams.Load(info.Key)
	if !ok {
		log.Infof("no stream record %s, app=%s, stream=%s", info.Key, info.App, info.Stream)
		if configure.GlobalConfig.AutoPullEnable {
			host, found := rs.GetOrigStream(info.App, info.Stream)
			if found {
				s = NewStream()
				rs.streams.Store(info.Key, s)
				s.info = info
				s.SetAutoPullHost(host)
				s.AddWriter(w)
				rs.AutoPullStream(s, host, info.App, info.Stream)
			}
		} else {
			log.Infof("auto pull, but stream not found, close writer %s, app=%s, stream=%s", info.Key, info.App, info.Stream)
			w.Close(fmt.Errorf("stream not found"))
		}
	} else {
		log.Infof("has stream record %s, app=%s, stream=%s", info.Key, info.App, info.Stream)
		if item.(*Stream).isAutoPullExit && configure.GlobalConfig.AutoPullEnable {
			log.Infof("has stream record, try to restart pull %s, app=%s, stream=%s", info.Key, info.App, info.Stream)
			host, found := rs.GetOrigStream(info.App, info.Stream)
			if found {
				s = NewStream()
				rs.streams.Store(info.Key, s)
				s.info = info
				s.SetAutoPullHost(host)
				s.AddWriter(w)
				rs.AutoPullStream(s, host, info.App, info.Stream)
			} else {
				log.Infof("auto pull, but stream not found, fail to restart, close writer %s, app=%s, stream=%s", info.Key, info.App, info.Stream)
				w.Close(fmt.Errorf("stream not found"))
				rs.streams.Delete(info.Key)
			}
		} else {
			s = item.(*Stream)
			s.AddWriter(w)
		}
	}
}

func (rs *RtmpStream) GetOrigStream(app string, stream string) (string, bool) {
	// for now: get remote host from configure
	host := configure.GlobalConfig.AutoPullRemoteHost
	return host, true
}

func (rs *RtmpStream) AutoPullStream(s *Stream, origHost string, app string, stream string) {
	log.Info("start auto pull")

	playurl := fmt.Sprintf("rtmp://%s/%s/%s", origHost, app, stream)
	pullRtmprelay := NewRtmpAutoPuller(&playurl, s)
	log.Infof("rtmprelay start push from %s", playurl)
	err := pullRtmprelay.Start()
	if err != nil {
		log.Errorf("auto pull stream fail %v", err)
	}
}

func (rs *RtmpStream) GetStreams() *sync.Map {
	return rs.streams
}

func (rs *RtmpStream) CheckAlive() {
	for {
		<-time.After(5 * time.Second)
		rs.streams.Range(func(key, val interface{}) bool {
			v := val.(*Stream)
			if v.CheckAlive() == 0 {
				rs.streams.Delete(key)
			}
			return true
		})
	}
}

type Stream struct {
	isStart        bool
	isAutoPull     bool
	isAutoPullExit bool
	autoPullHost   string
	cache          *cache.Cache
	r              av.ReaderCloser
	ws             *sync.Map
	info           av.Info
	demuxer        *flv.Demuxer
}

type PackWriterCloser struct {
	init bool
	w    av.WriterCloser
}

func (p *PackWriterCloser) GetWriter() av.WriterCloser {
	return p.w
}

func NewStream() *Stream {
	return &Stream{
		cache: cache.NewCache(),
		ws:    &sync.Map{},
	}
}

func (s *Stream) ID() string {
	if s.r != nil {
		return s.r.Info().UID
	}
	return EmptyID
}

func (s *Stream) GetInfo() av.Info {
	return s.info
}

func (s *Stream) SetAutoPullHost(host string) {
	s.autoPullHost = host
}

func (s *Stream) GetAutoPullHost() string {
	return s.autoPullHost
}

func (s *Stream) SetAutoPullExit() *Stream {
	s.isAutoPullExit = true
	return s
}

func (s *Stream) IsAutoPullExit() bool {
	return s.isAutoPullExit
}

func (s *Stream) IsAutoPull() bool {
	return s.isAutoPull
}

func (s *Stream) GetReader() av.ReaderCloser {
	return s.r
}

func (s *Stream) GetWs() *sync.Map {
	return s.ws
}

func (s *Stream) Copy(dst *Stream) {
	dst.info = s.info
	s.ws.Range(func(key, val interface{}) bool {
		v := val.(*PackWriterCloser)
		s.ws.Delete(key)
		v.w.CalcBaseTimestamp()
		dst.AddWriter(v.w)
		return true
	})
}

func (s *Stream) AddReader(r av.ReaderCloser) {
	s.r = r
	go s.TransStart()
}

func (s *Stream) AddWriter(w av.WriterCloser) {
	info := w.Info()
	pw := &PackWriterCloser{w: w}
	s.ws.Store(info.UID, pw)
}

func (s *Stream) TransStart() {
	s.isStart = true
	var p av.Packet
	log.Debugf("TransStart: %v", s.info)

	for {
		if !s.isStart {
			s.closeInter()
			return
		}
		err := s.r.Read(&p)
		if err != nil {
			s.closeInter()
			s.isStart = false
			return
		}

		s.cache.Write(p)

		s.ws.Range(func(key, val interface{}) bool {
			v := val.(*PackWriterCloser)
			if !v.init {
				if err = s.cache.Send(v.w); err != nil {
					log.Debugf("[%s] send cache packet error: %v, remove", v.w.Info(), err)
					s.ws.Delete(key)
					return true
				}
				v.init = true
			} else {
				newPacket := p
				if err = v.w.Write(&newPacket); err != nil {
					log.Debugf("[%s] write packet error: %v, remove", v.w.Info(), err)
					s.ws.Delete(key)
				}
			}
			return true
		})
	}
}

func (s *Stream) closeInter() {
	s.ws.Range(func(key, val interface{}) bool {
		v := val.(*PackWriterCloser)
		if v.w != nil {
			v.w.Close(fmt.Errorf("closed"))
			if v.w.Info().IsInterval() {
				s.ws.Delete(key)
				log.Debugf("[%v] player closed and remove\n", v.w.Info())
			}
		}
		return true
	})

	if s.r != nil {
		s.r.Close(fmt.Errorf("read exit..."))
	}
}

func (s *Stream) TransStop() {
	log.Debugf("TransStop: %s", s.info.Key)

	if s.isStart && s.r != nil {
		s.r.Close(fmt.Errorf("stop old"))
	}

	s.isStart = false
}

func (s *Stream) CheckAlive() (n int) {
	if s.r != nil && s.isStart {
		if s.r.Alive() {
			n++
		} else {
			s.r.Close(fmt.Errorf("read timeout"))
		}
	}

	if s.isAutoPull && !s.isAutoPullExit {
		n++ //play count
	}

	s.ws.Range(func(key, val interface{}) bool {
		v := val.(*PackWriterCloser)
		if v.w != nil {
			if !v.w.Alive() {
				log.Infof("write timeout remove")
				s.ws.Delete(key)
				v.w.Close(fmt.Errorf("write timeout"))
				return true
			}
			n++
		}
		return true
	})
	return
}

func (s *Stream) Hold() {
	if !configure.GlobalConfig.BlockMode {
		return
	}

	for {
		length := 0
		s.ws.Range(func(_, _ interface{}) bool {
			length++
			return false
		})

		if length > 0 {
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}

func (s *Stream) PublishToLocal(cs *core.ChunkStream) {
	s.isStart = true
	s.isAutoPull = true
	var p av.Packet

	if cs.TypeID != av.TAG_AUDIO &&
		cs.TypeID != av.TAG_VIDEO &&
		cs.TypeID != av.TAG_SCRIPTDATAAMF0 &&
		cs.TypeID != av.TAG_SCRIPTDATAAMF3 &&
		cs.TypeID != av.TAG_USER_CONTROL_MSG {
		return
	}

	p.IsAudio = cs.TypeID == av.TAG_AUDIO
	p.IsVideo = cs.TypeID == av.TAG_VIDEO
	p.IsMetaData = cs.TypeID == av.TAG_SCRIPTDATAAMF0 || cs.TypeID == av.TAG_SCRIPTDATAAMF3
	p.IsCtrlMsg = cs.TypeID == av.TAG_USER_CONTROL_MSG
	p.StreamID = cs.StreamID
	p.Data = cs.Data
	p.TimeStamp = cs.TimeStamp

	s.demuxer.DemuxH(&p)
	s.Hold()
	s.cache.Write(p)

	s.ws.Range(func(key, val interface{}) bool {
		v := val.(PackWriterCloser)
		if !v.init {
			if err := s.cache.Send(v.w); err != nil {
				log.Debugf("[%s] send cache packet err: %v, remove", v.w.Info(), err)
				s.ws.Delete(key)
				return true
			}
			v.init = true
		} else {
			newPacket := p
			if err := v.w.Write(&newPacket); err != nil {
				log.Debugf("[%s] write packet err: %v, remove", v.w.Info(), err)
				s.ws.Delete(key)
			}
		}
		return true
	})

}
