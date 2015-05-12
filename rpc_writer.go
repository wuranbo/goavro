package goavro

import (
	"encoding/binary"
	"fmt"
	"log"
)

type RpcWriter struct {
	*Writer
	msgname string
	Buflen  int
}

func NewRpcResponseWriter(responseSchema string, setters ...WriterSetter) (*RpcWriter, error) {
	var err error
	fw := &RpcWriter{Writer: &Writer{CompressionCodec: CompressionNull, blockSize: DefaultWriterBlockSize}}
	for _, setter := range setters {
		err = setter(fw.Writer)
		if err != nil {
			return nil, &ErrWriterInit{Err: err}
		}
	}
	if fw.w == nil {
		return nil, &ErrWriterInit{Message: "must specify io.Writer"}
	}
	// writer: stuff should already be initialized
	if !IsCompressionCodecSupported(fw.CompressionCodec) {
		return nil, &ErrWriterInit{Message: fmt.Sprintf("unsupported codec: %s", fw.CompressionCodec)}
	}

	if fw.dataCodec, err = NewCodec(responseSchema); err != nil {
		return nil, &ErrWriterInit{Message: "missing schema"}
	}
	// py实现里相当于写下一个frame的bufferlen==0，以此我们让framwriter只写一次frame。
	fw.Sync = make([]byte, 0)
	// setup writing pipeline
	fw.toBlock = make(chan interface{})
	toEncode := make(chan *writerBlock)
	toCompress := make(chan *writerBlock)
	toWrite := make(chan *writerBlock)
	fw.WriterDone = make(chan struct{})
	go blocker(fw.Writer, fw.toBlock, toEncode)
	go encoder(fw.Writer, toEncode, toCompress)
	go compressor(fw.Writer, toCompress, toWrite)
	go framewriter(fw, toWrite)
	return fw, nil
}

func framewriter(fw *RpcWriter, toWrite <-chan *writerBlock) {
	// TODO(wu.ranbo@yottabyte.cn) very import !!, ipc frame简化只有一个block，多个循环是错误。
	for block := range toWrite {
		buflen := len(block.compressed) + 2
		err := binary.Write(fw.w, binary.BigEndian, int32(buflen))
		fw.Buflen = buflen
		if err != nil {
			log.Printf("[WARNING] write buffer len failed!.: %v", err)
		}
		// fake empty meta.
		err = binary.Write(fw.w, binary.BigEndian, int16(0))
		if block.err == nil {
			_, block.err = fw.w.Write(block.compressed)
		}
		if block.err == nil {
			_, block.err = fw.w.Write(fw.Sync)
		}
		if block.err != nil {
			log.Printf("[WARNING] cannot write block: %v", block.err)
			fw.err = block.err // ???
			break
			// } else {
			// 	log.Printf("[DEBUG] block written: %d, %d, %v", len(block.items), len(block.compressed), block.compressed)
		}
	}
	err := binary.Write(fw.w, binary.BigEndian, int32(0))
	if err != nil {
		log.Printf("[WARNING] write stop endding failed!.: %v", err)
	}
	// TODO(wu.ranbo@yottabyte.cn) wuranbo,do it.
	// NOTICE(wu.ranbo@yottabyte.cn) Cause avro python impl not has this zero tail.
	// if fw.err = longCodec.Encode(fw.w, int64(0)); fw.err == nil {
	// fw.err = longCodec.Encode(fw.w, int64(0))
	// }
	fw.WriterDone <- struct{}{}
}
