// Package main provides ...
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/wuranbo/goavro"
)

var requestCodec goavro.Codec
var responseCodec goavro.Codec
var requestSchema string
var responseSchema string
var respMessageSchema string

type Hello struct{}

func (h Hello) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request) {

	fr, err := goavro.NewRpcRequestReader(requestSchema, goavro.FromReader(r.Body))
	if err != nil {
		log.Fatal(err)
	}

	doing := func(datum interface{}, err error) error {
		fmt.Println("in scan datum:%v.", datum)
		if datum == nil {
			return nil
		}
		record := datum.(*goavro.Record)
		for _, f := range record.Fields {
			fmt.Println("record.Name:", f.Name)
			fmt.Println("record.Datum:", f.Datum)
		}
		if err != nil {
			log.Println("can not read datum:", err)
		}
		return nil
	}

	fr.ScanData(doing)
	fr.Close()

	fw, err := goavro.NewRpcResponseWriter(
		responseSchema,
		goavro.Compression(goavro.CompressionNull),
		goavro.WriterSchema(responseSchema),
		goavro.ToWriter(w))
	if err != nil {
		log.Printf("make writer failed!", err)
	}

	responseRecord, err := goavro.NewRecord(goavro.RecordSchema(responseSchema))
	if err != nil {
		log.Printf("heartbeat response schema to record failed!", err)
	}
	responseRecord.Set("data_hash", []byte{'s', 'f', 'a', 'd', 'f'})
	responseRecord.Set("data", nil)
	responseRecord.Set("data", nil)
	responseRecord.Set("last_request_hash", []byte{'s', 'f', 'a', 'd', 'f'})
	responseRecord.Set("ts_recv", int64(2))
	responseRecord.Set("ts_send", int64(2))

	fw.Write(responseRecord)

	defer func() {
		fr.Close()
		fw.Close()
		<-fw.WriterDone
	}()
}

func main() {
	var h Hello
	err := http.ListenAndServe("localhost:8888", h)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	// var err error
	req, err := ioutil.ReadFile("RpcRequest.avsc")
	if err != nil {
		fmt.Println("err:", err)
	}
	requestSchema = string(req)

	resp, err := ioutil.ReadFile("HeartbeatResponse.avsc")
	responseSchema = string(resp)

	if err != nil {
		log.Fatal(err)
	}
}
