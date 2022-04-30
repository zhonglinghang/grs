package rtmp

import (
	"bytes"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/zhonglinghang/grs/protocol/amf"
	"github.com/zhonglinghang/grs/protocol/rtmp/core"
)

var (
	STOP_CTRL = "RTMPRELAY_STOP"
)

type RtmpAutoPuller struct {
	PlayUrl           string
	PubStream         *Stream
	cs_chan           chan core.ChunkStream
	sndctrl_chan      chan string
	connectPlayClient *core.ConnClient
	startflag         bool
}

func NewRtmpAutoPuller(playUrl *string, pubStream *Stream) *RtmpAutoPuller {
	return &RtmpAutoPuller{
		PlayUrl:           *playUrl,
		PubStream:         pubStream,
		cs_chan:           make(chan core.ChunkStream, 500),
		sndctrl_chan:      make(chan string),
		connectPlayClient: nil,
		startflag:         false,
	}
}

func (self *RtmpAutoPuller) Start() error {
	if self.startflag {
		return fmt.Errorf("The auto pull already started, playurl=%s", self.PlayUrl)
	}

	self.connectPlayClient = core.NewConnClient()

	log.Debugf("play server addr: %v starting....", self.PlayUrl)
	err := self.connectPlayClient.Start(self.PlayUrl, "play")
	if err != nil {
		log.Debugf("connectPlayClient.Start url=%v error", self.PlayUrl)
		return err
	}
	self.startflag = true
	go self.rcvPlayChunkStream()
	go self.sendPublishChunkStream()

	return nil
}

func (self *RtmpAutoPuller) Stop() {
	if !self.startflag {
		log.Debugf("The auto pull already stopped, playurl=%s", self.PlayUrl)
		return
	}

	self.startflag = false
	self.sndctrl_chan <- STOP_CTRL
}

func (self *RtmpAutoPuller) rcvPlayChunkStream() {
	log.Debug("autoPull rcvPlayRtmpMediaPacket connectClient read...")
	for {
		var rc core.ChunkStream

		if self.startflag == false {
			self.connectPlayClient.Close(nil)
			log.Debugf("autoPull rcvPlayChunkStream close: playurl=%s", self.PlayUrl)
			break
		}
		err := self.connectPlayClient.Read(&rc)

		if err != nil {
			//TODO: Stop publish
			log.Infof("stop auto pull")
			self.Stop()
			break
		}

		switch rc.TypeID {
		case 20, 17:
			r := bytes.NewReader(rc.Data)
			vs, err := self.connectPlayClient.DecodeBatch(r, amf.AMF0)
			log.Debugf("autoPull rcvPlayRtmpMediaPacket: vs=%v, err=%v", vs, err)
		case 18:
			log.Debug("autoPull rcvPlayRtmpMediaPacket: metadata skip")
		case 8, 9, 4:
			self.cs_chan <- rc
		}
	}
}

func (self *RtmpAutoPuller) sendPublishChunkStream() {
	for {
		select {
		case rc := <-self.cs_chan:
			self.PubStream.PublishToLocal(&rc)
		case ctrlmcd := <-self.sndctrl_chan:
			if ctrlmcd == STOP_CTRL {
				self.PubStream.SetAutoPullExit()
				// close all the players
				self.PubStream.closeInter()
				log.Debugf("autoPull sendPublishChunkStream close: playurl=%s", self.PlayUrl)
				return
			}
		}
	}
	log.Infof("local relay stop")
}
