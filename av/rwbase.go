package av

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zhonglinghang/grs/configure"
)

type RWBaser struct {
	lock               sync.Mutex
	timeout            time.Duration
	PreTime            time.Time
	BaseTimestamp      uint32
	LastVideoTimestamp uint32
	LastAudioTimestamp uint32
}

func NewRWBaser(duration time.Duration) RWBaser {
	return RWBaser{
		timeout: duration,
		PreTime: time.Now(),
	}
}

func (rw *RWBaser) BaseTimeStamp() uint32 {
	return rw.BaseTimestamp
}

func (rw *RWBaser) CalcBaseTimestamp() {
	if rw.LastAudioTimestamp > rw.LastVideoTimestamp {
		rw.BaseTimestamp = rw.LastAudioTimestamp
	} else {
		rw.BaseTimestamp = rw.LastVideoTimestamp
	}
}

func (rw *RWBaser) RecTimeStamp(timestamp, typeID uint32) {
	if typeID == TAG_VIDEO {
		rw.LastVideoTimestamp = timestamp
	} else if typeID == TAG_AUDIO {
		rw.LastAudioTimestamp = timestamp
	}
}

func (rw *RWBaser) SetPreTime() {
	rw.lock.Lock()
	rw.PreTime = time.Now()
	rw.lock.Unlock()
}

func (rw *RWBaser) Alive() bool {
	// do not mark it dead in the block mode
	if configure.GlobalConfig.BlockMode {
		return true
	}
	rw.lock.Lock()
	b := !(time.Now().Sub(rw.PreTime) >= rw.timeout)
	if !b {
		log.Warningf("rwbase not alive time: %v now: %v, timeout: %d", rw.PreTime, time.Now(), rw.timeout)
	}
	rw.lock.Unlock()
	return b
}
