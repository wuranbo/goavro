package goavro

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

type IpcReader struct {
	*Reader
	msgname string
	buflen  int
	datalen int
}

type ErrReadIpcMessageName struct {
	Err error
}

func (e *ErrReadIpcMessageName) Error() string {
	return "ErrReadIpcMessageName:" + e.Err.Error()
}

type ErrReadIpcBufferLen struct {
	Err error
}

func (e *ErrReadIpcBufferLen) Error() string {
	return "ErrReadIpcBufferLen: " + e.Err.Error()
}

func NewIpcRequestReader(requestSchema string, setters ...ReaderSetter) (*Reader, error) {
	var err error
	fr := &IpcReader{Reader: &Reader{}}
	for _, setter := range setters {
		err = setter(fr.Reader)
		if err != nil {
			return nil, newReaderInitError(err)
		}
	}
	if fr.r == nil {
		return nil, newReaderInitError("must specify io.Reader")
	}
	buflen, _, err := decodeBufferLength(fr.r)
	if err != nil {
		return nil, newReaderInitError("cannot read buff length", err)
	}
	fr.datalen = buflen

	_, err = decodeHeaderMetadata(fr.r)
	if err != nil {
		return nil, newReaderInitError("cannot read header metadata", err)
	}
	fr.datalen -= 1 // 对应py内meta为空，长度是一个byte, 看ipc.py中bug描述

	name, namelen, err := decodeIpcMessageName(fr.r)
	if err != nil {
		return nil, newReaderInitError("cannot decode ip message name", err)
	}
	fr.msgname = name
	fr.datalen -= namelen

	fr.CompressionCodec = "null" // python ipc 不加密
	fr.DataSchema = requestSchema
	if fr.dataCodec, err = NewCodec(fr.DataSchema); err != nil {
		return nil, newReaderInitError("cannot compile schema", err)
	}
	fr.Sync = make([]byte, 0) // 无sync
	// setup reading pipeline
	toDecompress := make(chan *readerBlock)
	toDecode := make(chan *readerBlock)
	fr.deblocked = make(chan Datum)
	go frameRead(fr, toDecompress)
	go decompress(fr.Reader, toDecompress, toDecode)
	go decode(fr.Reader, toDecode)
	return fr.Reader, nil
}

func decodeIpcMessageName(r io.Reader) (string, int, error) {
	name, err := stringCodec.Decode(r)
	if err != nil {
		if ed, ok := err.(*ErrDecoder); ok && ed.Err.Error() == "EOF" {
			return "", 0, nil // we're done
		}
		return "", 0, &ErrReadIpcMessageName{err}
	}
	bb := new(bytes.Buffer)
	err = stringCodec.Encode(bb, name)
	if err != nil {
		return "", 0, &ErrReadIpcMessageName{err}
	}
	return name.(string), bb.Len(), nil
}

// 跟py里buffer_length一致
func decodeBufferLength(r io.Reader) (int, int, error) {
	var bl uint32
	err := binary.Read(r, binary.BigEndian, &bl)
	if err != nil {
		if ed, ok := err.(*ErrDecoder); ok && ed.Err.Error() == "EOF" {
			return 0, 0, nil // we're done
		}
		return 0, 0, &ErrReadIpcBufferLen{err}
	}
	return int(bl), 4, nil
}

// 去掉sync
func frameRead(fr *IpcReader, toDecompress chan<- *readerBlock) {
	lr := io.LimitReader(fr.Reader.r, int64(fr.datalen))
	bits, err := ioutil.ReadAll(lr)
	if err != nil {
		err = newReaderError("cannot read block", err)
	}
	toDecompress <- &readerBlock{datumCount: 1, r: bytes.NewReader(bits)}
	close(toDecompress)
}
