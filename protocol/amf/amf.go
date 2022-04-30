package amf

import (
	"fmt"
	"io"
)

func (d *Decoder) DecodeBatch(r io.Reader, ver Version) (ret []interface{}, err error) {
	var v interface{}
	for {
		v, err = d.Decode(r, ver)
		if err != nil {
			break
		}
		ret = append(ret, v)
	}
	return
}

func (d *Decoder) Decode(r io.Reader, ver Version) (interface{}, error) {
	switch ver {
	case 0:
		return d.DecodeAmf0(r)
	case 3:
		return d.DecodeAmf3(r)
	}

	return nil, fmt.Errorf("decode amf: unsupported version %d", ver)
}

func (e *Encoder) EncodeBatch(w io.Writer, ver Version, val ...interface{}) (int, error) {
	for _, v := range val {
		if _, err := e.Encode(w, v, ver); err != nil {
			return 0, err
		}
	}
	return 0, nil
}

func (e *Encoder) Encode(w io.Writer, val interface{}, ver Version) (int, error) {
	switch ver {
	case AMF0:
		return e.EncodeAmf0(w, val)
	case AMF3:
		return e.EncodeAmf3(w, val)
	}

	return 0, fmt.Errorf("encode amf: unsupported version %d", ver)
}

func (e *Encoder) EncodeString(w io.Writer, val string, ver Version) (int, error) {
	switch ver {
	case AMF0:
		return e.EncodeAmf0StringValue(w, val)
	case AMF3:
		return e.EncodeAmf3StringValue(w, val)
	}
	return 0, fmt.Errorf("encode amf: unsupported version %d", ver)
}

func (e *Encoder) EncodeInt(w io.Writer, val int, ver Version) (int, error) {
	switch ver {
	case AMF0:
		return e.EncodeAmf0IntValue(w, int64(val))
	case AMF3:
		return e.EncodeAmf3IntValue(w, int64(val))
	}
	return 0, fmt.Errorf("encode amf: unsupported version %d", ver)
}

func (e *Encoder) EncodeObject(w io.Writer, val Object, ver Version) (int, error) {
	switch ver {
	case AMF0:
		return e.EncodeAmf0ObjectValue(w, val)
	case AMF3:
		return e.EncodeAmf3ObjectValue(w, val)
	}
	return 0, fmt.Errorf("encode amf: unsupported version %d", ver)
}
