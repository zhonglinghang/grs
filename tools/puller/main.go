package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/jpeg"

	"github.com/hajimehoshi/ebiten"
	"github.com/zhonglinghang/grs/protocol/rtmp/core"
)

var (
	inRtmpAddress = flag.String("i", "", "Input rtmp address")
	isPic         = flag.Bool("p", false, "is jpeg stream")
)

func init() {
	flag.Parse()
}

func main() {
	puller := NewRtmpDumper(inRtmpAddress)

	if *isPic {
		puller.SetPacketHandler(func(cs *core.ChunkStream) error {
			packet2Img(cs)
			return nil
		})
	}

	puller.Start()

	if *isPic {
		initDisplay()
	}

	for {
		select {}
	}
}

var gimage *ebiten.Image

func update(screen *ebiten.Image) error {
	op := &ebiten.DrawImageOptions{}
	// op.GeoM.Rotate(0.234)      // random non 45-degree angle
	op.GeoM.Scale(1, 1)             // just for a fine pic
	op.Filter = ebiten.FilterLinear // non DefaultFilter here needed
	screen.DrawImage(gimage, op)
	return nil
}

func initDisplay() {
	gimage, _ = ebiten.NewImage(800, 800, ebiten.FilterDefault)
	ebiten.Run(update, 800, 800, 1, "minitest")
}

func isJpeg(data []byte) bool {
	codecID := data[0] & 0x0f
	return codecID == 1
}

func packet2Img(cs *core.ChunkStream) {
	if cs.TypeID != 9 || !isJpeg(cs.Data) {
		fmt.Println("not jpeg frame skip")
		return
	}

	buf := bytes.NewBuffer(cs.Data[5:])
	i, err := jpeg.Decode(buf)
	if err != nil {
		fmt.Printf("decode fail err=%v\n", err)
		return
	}

	gimage, err = ebiten.NewImageFromImage(i, ebiten.FilterDefault)
	if err != nil {
		fmt.Printf("create image fail err=%v\n", err)
		return
	}
}
