package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yutopp/go-flv"
	"github.com/yutopp/go-flv/tag"
	"github.com/zhonglinghang/grs/av"
	"github.com/zhonglinghang/grs/protocol/rtmp/rtmprelay"
)

var (
	inFilePathOpt = flag.String("i", "", "Input file")
	outRtmp       = flag.String("o", "", "target rtmp address")
)

func init() {
	flag.Parse()
}

func main() {
	log.Infof("Open file: %s", *inFilePathOpt)

	file, err := os.Open(*inFilePathOpt)
	if err != nil {
		log.Fatalf("Failed to open file: %+v", err)
	}
	defer file.Close()

	dec, err := flv.NewDecoder(file)
	if err != nil {
		log.Fatalf("Failed to create decoder: %+v", err)
	}

	pusher := rtmprelay.NewStaticPush(*outRtmp)
	err = pusher.Start()

	if err != nil {
		log.Errorf("Failed to start pusher: %+v", err)
		return
	}

	var startTimeStamp int64 = -1
	startWallClock := time.Now()
	for {
		var flvTag tag.FlvTag
		if err := dec.Decode(&flvTag); err != nil {
			if err == io.EOF {
				log.Infof("EOF: %+v", err)
				pusher.SendStreamEOFCmd()
				break
			}
			log.Warnf("Failed to decode: %+v", err)
		}

		packet := av.Packet{}
		packet.TimeStamp = flvTag.Timestamp

		if startTimeStamp < 0 {
			startTimeStamp = int64(packet.TimeStamp)
		}

		switch flvTag.TagType {
		case 8:
			packet.IsAudio = true
			ad := flvTag.Data.(*tag.AudioData)
			buf := bytes.NewBuffer([]byte{})
			tag.EncodeAudioData(buf, ad)
			packet.Data = buf.Bytes()
			packet.StreamID = flvTag.StreamID
		case 9:
			packet.IsVideo = true
			vd := flvTag.Data.(*tag.VideoData)
			buf := bytes.NewBuffer([]byte{})
			err = tag.EncodeVideoData(buf, vd)
			if err != nil {
				log.Errorf("encode error: %+v", err)
			}
			packet.Data = buf.Bytes()
			packet.StreamID = flvTag.StreamID
		case 18:
			packet.IsMetaData = true
			sd := flvTag.Data.(*tag.ScriptData)
			buf := bytes.NewBuffer([]byte{})
			tag.EncodeScriptData(buf, sd)
			packet.Data = buf.Bytes()
			packet.StreamID = flvTag.StreamID
		}
		pusher.WriteAvPacket(&packet)

		// control send speed
		if flvTag.TagType == 8 {
			elapse := time.Since(startWallClock)
			if time.Duration(int64(flvTag.Timestamp)-startTimeStamp)*time.Millisecond > elapse+time.Millisecond*5 {
				time.Sleep(time.Duration(int64(flvTag.Timestamp)-startTimeStamp)*time.Millisecond - elapse)
			}
		}

		flvTag.Close() // discard unread buffers
	}
}
