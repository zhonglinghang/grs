package rtmp

import (
	log "github.com/sirupsen/logrus"
	"github.com/zhonglinghang/grs/av"
	"github.com/zhonglinghang/grs/protocol/rtmp/core"
)

type RtmpClient struct {
	handler av.Handler
	getter  av.GetWriter
}

func NewRtmpClient(h av.Handler, getter av.GetWriter) *RtmpClient {
	return &RtmpClient{
		handler: h,
		getter:  getter,
	}
}

func (c *RtmpClient) Dial(url string, method string) error {
	connClient := core.NewConnClient()
	if err := connClient.Start(url, method); err != nil {
		return err
	}
	if method == av.PUBLISH {
		writer := NewVirWriter(connClient)
		log.Debugf("client Dial call NewVirWriter url=%s, method=%s", url, method)
		c.handler.HandleWriter(writer)
	} else if method == av.PLAY {
		reader := NewVirReader(connClient)
		log.Debugf("client Dial call NewVirReader url=%s, method=%s", url, method)
		c.handler.HandleReader(reader)
		if c.getter != nil {
			writer := c.getter.GetWriter(reader.Info())
			c.handler.HandleWriter(writer)
		}
	}
	return nil
}

func (c *RtmpClient) GetHandle() av.Handler {
	return c.handler
}
