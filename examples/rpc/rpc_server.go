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

// var reqMessageSchema string
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

	fw, err := goavro.NewRpcResponseWriter(
		// goavro.Compression(goavro.CompressionSnappy),
		responseSchema,
		goavro.Compression(goavro.CompressionNull),
		goavro.WriterSchema(responseSchema),
		goavro.ToWriter(w))

	if err != nil {
		log.Fatal(err)
	}

	messages := make([]interface{}, 0)

	respMessageRecord, err := goavro.NewRecord(goavro.RecordSchema(respMessageSchema))
	if err != nil {
		fmt.Println("sss")
		log.Fatal(err)
	}
	respMessageRecord.Set("to", "t")
	respMessageRecord.Set("from", "f")
	respMessageRecord.Set("respbody", "r")
	messages = append(messages, respMessageRecord)

	responseRecord, err := goavro.NewRecord(goavro.RecordSchema(responseSchema))
	if err != nil {
		log.Fatal(err)
	}

	// responseRecord.Set("messages", nil)
	// responseRecord.Set("messages", "null")
	responseRecord.Set("messages", messages)

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
	var err error
	req, err := ioutil.ReadFile("request.avsc")
	if err != nil {
		fmt.Println("err:", err)
	}
	requestSchema = string(req)

	respm, err := ioutil.ReadFile("resp_message.avsc")
	respMessageSchema = string(respm)

	resp, err := ioutil.ReadFile("response.avsc")
	responseSchema = string(resp)
	responseCodec, err = goavro.NewCodec(responseSchema)

	if err != nil {
		log.Fatal(err)
	}
}
