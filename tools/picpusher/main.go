package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/yutopp/go-flv/tag"
	"github.com/zhonglinghang/grs/av"
	"github.com/zhonglinghang/grs/protocol/rtmp/rtmprelay"
)

var (
	outRtmp = flag.String("o", "", "target rtmp address")
)

func init() {
	flag.Parse()
}

func main() {
	img1, _ := LoadImage("/Users/hangzhongling/IMG_0222.JPG")
	img2, _ := LoadImage("/Users/hangzhongling/IMG_0223.JPG")

	log.Infof("img1 %d, img2 %d", len(img1), len(img2))

	pusher := rtmprelay.NewStaticPush(*outRtmp)
	err := pusher.Start()

	if err != nil {
		log.Errorf("Failed to start pusher: %+v", err)
		return
	}

	var ts uint32 = 0
	for {
		pusher.WriteAvPacket(ConstructFlvPackage(img1, ts))
		time.Sleep(time.Second)
		pusher.WriteAvPacket(ConstructFlvPackage(img2, ts))
		time.Sleep(time.Second)
		ts += 1000
		pusher.SendStreamPingRequestCmd()
	}
}

func LoadImage(file string) ([]byte, error) {
	fd, err := os.Open(file)
	if err != nil {
		log.Errorf("Failed to openfile: %+v", err)
	}

	return ioutil.ReadAll(fd)
}

func ConstructFlvPackage(imgs []byte, timestamp uint32) *av.Packet {
	packet := av.Packet{}
	packet.TimeStamp = timestamp
	packet.IsVideo = true

	buf := bytes.NewBuffer([]byte{})
	bufHeader := make([]byte, 5)
	bufHeader[0] |= byte(tag.FrameTypeKeyFrame<<4) & 0xf0 // 0b11110000
	bufHeader[0] |= byte(tag.CodecIDJPEG) & 0x0f          // 0b00001111

	bufHeader[1] = byte(0) // JPEG Packet type unused
	bufHeader[2] = byte(0) // JPEG CompositionTime unused
	bufHeader[3] = byte(0) // JPEG CompositionTime unused
	bufHeader[4] = byte(0) // JPEG CompositionTime unused

	buf.Write(bufHeader)
	buf.Write(imgs)

	packet.Data = buf.Bytes()
	log.Infof("data size %d", len(packet.Data))
	return &packet
}
