package rtmp

import (
	"net"
	"reflect"

	"github.com/zhonglinghang/grs/av"
	"github.com/zhonglinghang/grs/configure"
	"github.com/zhonglinghang/grs/container/flv"
	"github.com/zhonglinghang/grs/protocol/rtmp/core"

	log "github.com/sirupsen/logrus"
)

const (
	maxQueueNum           = 1024
	SAVE_STATICS_INTERVAL = 5000
)

type RtmpServer struct {
	handler av.Handler
	getter  av.GetWriter
}

type GetInFo interface {
	GetInfo() (string, string, string)
}

type StreamReadWriteCloser interface {
	GetInFo
	Close(error)
	Write(core.ChunkStream) error
	Read(c *core.ChunkStream) error
	Flush() error
}

type StaticsBW struct {
	StreamId               uint32
	VideoDatainBytes       uint64
	LastVideoDatainBytes   uint64
	VideoSpeedInBytesperMS uint64

	AudioDatainBytes       uint64
	LastAudioDatainBytes   uint64
	AudioSpeedInBytesperMS uint64

	LastTimestamp int64
}

func NewRtmpServer(h av.Handler, getter av.GetWriter) *RtmpServer {
	return &RtmpServer{
		handler: h,
		getter:  getter,
	}
}

func (s *RtmpServer) Serve(listener net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("rtmp serve panic: ", r)
		}
	}()

	for {
		var netconn net.Conn
		netconn, err = listener.Accept()
		if err != nil {
			return
		}
		conn := core.NewConn(netconn, 4*1024)
		log.Debug("new client, connect remote: ", conn.RemoteAddr().String(),
			"local:", conn.LocalAddr().String())
		go s.handleConn(conn)
	}
}

func (s *RtmpServer) handleConn(conn *core.Conn) error {
	if err := conn.HandshakeServer(); err != nil {
		conn.Close()
		log.Error("handleConn HandshakeServer err: ", err)
		return err
	}
	connServer := core.NewConnServer(conn)

	if err := connServer.ReadMsg(); err != nil {
		conn.Close()
		log.Error("handleConn read msg err: ", err)
		return err
	}

	appname, name, _ := connServer.GetInfo()

	log.Debugf("handleConn: IsPublisher=%v", connServer.IsPublisher())
	if connServer.IsPublisher() {

		connServer.PublishInfo.Name = name
		if pushlist, ret := configure.GetStaticPushUrlList(appname); ret && (pushlist != nil) {
			log.Debugf("GetStaticPushUrlList: %v", pushlist)
		}
		reader := NewVirReader(connServer)
		s.handler.HandleReader(reader)
		log.Debugf("new publisher: %+v", reader.Info())

		if s.getter != nil {
			writeType := reflect.TypeOf(s.getter)
			log.Debugf("handleConn:writeType=%v", writeType)
			writer := s.getter.GetWriter(reader.Info())
			s.handler.HandleWriter(writer)
		}
		if configure.Config.GetBool("flv_archive") {
			flvWriter := new(flv.FlvDvr)
			s.handler.HandleWriter(flvWriter.GetWriter(reader.Info()))
		}
	} else {
		writer := NewVirWriter(connServer)
		log.Debugf("new player: %+v", writer.Info())
		s.handler.HandleWriter(writer)
	}

	return nil
}
