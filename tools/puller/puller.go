package main

import (
	"bytes"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/zhonglinghang/grs/protocol/amf"
	"github.com/zhonglinghang/grs/protocol/rtmp/core"
)

var (
	STOP_CTRL = "RTMPRELAY_STOP"
)

type RtmpDumper struct {
	PlayUrl           string
	cs_chan           chan core.ChunkStream
	sndctrl_chan      chan string
	connectPlayClient *core.ConnClient
	startflag         bool
	packetHandler     func(*core.ChunkStream) error
}

func NewRtmpDumper(playurl *string) *RtmpDumper {
	return &RtmpDumper{
		PlayUrl:           *playurl,
		cs_chan:           make(chan core.ChunkStream, 500),
		sndctrl_chan:      make(chan string),
		connectPlayClient: nil,
		startflag:         false,
	}
}

func (self *RtmpDumper) SetPacketHandler(f func(*core.ChunkStream) error) {
	self.packetHandler = f
}

func (self *RtmpDumper) rcvPlayChunkStream() {
	log.Info("rcvPlayRtmpMediaPacket connectClient.Read...")
	for {
		var rc core.ChunkStream

		if self.startflag == false {
			self.connectPlayClient.Close(nil)
			log.Infof("rcvPlayChunkStream close: playurl=%s, publish url = %s", self.PlayUrl)
			break
		}
		err := self.connectPlayClient.Read(&rc)

		if err != nil && err == io.EOF {
			self.Stop()
			break
		}

		switch rc.TypeID {
		case 20, 17:
			r := bytes.NewReader(rc.Data)
			vs, err := self.connectPlayClient.DecodeBatch(r, amf.AMF0)

			log.Infof("rcvPlayRtmpMediaPacket: vs=%v, error:%v", vs, err)
		case 18:
			log.Info("rcvPlayRtmpMediaPacket: metadata...")
		case 8, 9, 4:
			self.cs_chan <- rc
		}
	}
}

func (self *RtmpDumper) dumpChunkStream() {
	for {
		select {
		case rc := <-self.cs_chan:
			log.Infof("dumpChunkStream: rc.TypeID=%v length=%d, timestamp=%d", rc.TypeID, len(rc.Data), rc.TimeStamp)
			if self.packetHandler != nil {
				self.packetHandler(&rc)
			}
		case ctrlcmd := <-self.sndctrl_chan:
			if ctrlcmd == STOP_CTRL {
				log.Infof("dumpChunkStream close: playurl=%s, publishurl=%s", self.PlayUrl)
				return
			}
		}

	}
}

func (self *RtmpDumper) Start() error {
	if self.startflag {
		return fmt.Errorf("The rtmpdumper already started, playerurl: %s", self.PlayUrl)
	}

	self.connectPlayClient = core.NewConnClient()

	log.Infof("play server addr:%v starting....", self.PlayUrl)
	err := self.connectPlayClient.Start(self.PlayUrl, "play")
	if err != nil {
		log.Info("connectPlayClient.Start url: %v error", self.PlayUrl)
		return err
	}

	self.startflag = true
	go self.rcvPlayChunkStream()
	go self.dumpChunkStream()
	return nil
}

func (self *RtmpDumper) Stop() {
	if !self.startflag {
		log.Infof("The rtmpdumper already stopped, playurl = %s", self.PlayUrl)
		return
	}

	self.startflag = false
	self.sndctrl_chan <- STOP_CTRL
}
