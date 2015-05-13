package goavro

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

type RpcReader struct {
	*Reader
	MsgName string
	Buflen  int
	Datalen int
}

type ErrReadRpcMessageName struct {
	Err error
}

func (e *ErrReadRpcMessageName) Error() string {
	return "ErrReadRpcMessageName:" + e.Err.Error()
}

type ErrReadRpcBufferLen struct {
	Err error
}

func (e *ErrReadRpcBufferLen) Error() string {
	return "ErrReadRpcBufferLen: " + e.Err.Error()
}

func NewRpcRequestReader(requestSchema string, setters ...ReaderSetter) (*RpcReader, error) {
	var err error
	fr := &RpcReader{Reader: &Reader{}}
	for _, setter := range setters {
		err = setter(fr.Reader)
		if err != nil {
			return nil, newReaderInitError(err)
		}
	}
	if fr.r == nil {
		return nil, newReaderInitError("must specify io.Reader")
	}
	fr.Buflen, _, err = decodeBufferLength(fr.r)
	if err != nil {
		return nil, newReaderInitError("cannot read buff length", err)
	}
	fr.Datalen = fr.Buflen

	_, err = decodeHeaderMetadata(fr.r)
	if err != nil {
		return nil, newReaderInitError("cannot read header metadata", err)
	}
	fr.Datalen -= 1 // 对应py内meta为空，长度是一个byte, 看ipc.py中bug描述

	name, namelen, err := decodeRpcMessageName(fr.r)
	if err != nil {
		return nil, newReaderInitError("cannot decode ip message name", err)
	}
	fr.MsgName = name
	fr.Datalen -= namelen

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
	return fr, nil
}

func decodeRpcMessageName(r io.Reader) (string, int, error) {
	name, err := stringCodec.Decode(r)
	if err != nil {
		if ed, ok := err.(*ErrDecoder); ok && ed.Err.Error() == "EOF" {
			return "", 0, nil // we're done
		}
		return "", 0, &ErrReadRpcMessageName{err}
	}
	bb := new(bytes.Buffer)
	err = stringCodec.Encode(bb, name)
	if err != nil {
		return "", 0, &ErrReadRpcMessageName{err}
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
		return 0, 0, &ErrReadRpcBufferLen{err}
	}
	return int(bl), 4, nil
}

// rpc的frame嵌入到读文件block的流程中的变动：
// block是每个开头有两个loang标记总block和当前block长度
// ipc是每个frame只开头一个buflen标记当前frame的长度，通过最后一个frame为四个0byte来区分
// 而且开头第一个fram的buflen已经在newrpcreader时候被读出来记录
// !rpc_writer里跟block写法集成的方式于此类似
func frameRead(fr *RpcReader, toDecompress chan<- *readerBlock) {
	// 没有握手不考虑反复的流式传输的py的实现，更简化为一次请求只有一个frame
	lr := io.LimitReader(fr.Reader.r, int64(fr.Datalen)) // 第一个block就把内容都读出来
	bits, err := ioutil.ReadAll(lr)
	if err != nil {
		err = newReaderError("cannot read block", err)
	}

	// 主Scan()方法中是fr.datum<-fr.deblock，因此会等待fr.deblock有值
	toDecompress <- &readerBlock{datumCount: 1, r: bytes.NewReader(bits)} // 第一个block
	// 这一句是三个worker routine触发的开始，执行后会让另一个routine：ocf_reader.go:decode()执行fr.deblocked <- datum。从而主routie的Scan()方法可以继续执行。
	// 在ocf_reader.go:read()中是循环block，因此当主routine,Scan()方法没有被执行的时候应该卡在decode()fr.deblock<-datum这一句

	// routine:frameRead()在循环读取block结束后会继续，因此close掉decompress
	// closedecompress导致另一个routine:decompress()执行结束，执行了close(toDecode)
	close(toDecompress)
}
