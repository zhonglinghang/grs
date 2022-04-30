package main

import (
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/zhonglinghang/grs/configure"
	"github.com/zhonglinghang/grs/protocol/api"
	"github.com/zhonglinghang/grs/protocol/hls"
	"github.com/zhonglinghang/grs/protocol/rtmp"

	log "github.com/sirupsen/logrus"
)

var VERSION = "master"

func startAPI(stream *rtmp.RtmpStream) {
	apiAddr := configure.Config.GetString("api_addr")
	rtmpAddr := configure.Config.GetString("rtmp_addr")

	if apiAddr != "" {
		opListen, err := net.Listen("tcp", apiAddr)
		if err != nil {
			log.Fatal(err)
		}
		opServer := api.NewServer(stream, rtmpAddr)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Error("HTTP-API server panic: ", r)
				}
			}()
			log.Info("HTTP-API listen On ", apiAddr)
			opServer.Serve(opListen)
		}()
	}
}

func startRtmp(stream *rtmp.RtmpStream, hlsServer *hls.Server) {
	rtmpAddr := configure.Config.GetString("rtmp_addr")

	rtmpListen, err := net.Listen("tcp", rtmpAddr)
	if err != nil {
		log.Fatal(err)
	}

	var rtmpServer *rtmp.RtmpServer

	if hlsServer == nil {
		rtmpServer = rtmp.NewRtmpServer(stream, nil)
		log.Info("HLS server disable....")
	} else {
		rtmpServer = rtmp.NewRtmpServer(stream, hlsServer)
		log.Info("HLS server enable....")
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error("RTMP server panic: ", r)
		}
	}()
	log.Info("RTMP Listen On ", rtmpAddr)
	rtmpServer.Serve(rtmpListen)
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("grs panic: ", r)
			time.Sleep(1 * time.Second)
		}
	}()

	configure.InitCfg()
	runtime.GOMAXPROCS(16)
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	stream := rtmp.NewRtmpStream()
	startAPI(stream)
	startRtmp(stream, nil)
}
