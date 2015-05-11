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
var reqMessageSchema string
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
	defer fr.Close()
	for fr.Scan() {
		fmt.Println("in fr.scan()")
		datum, err := fr.Read()
		fmt.Println("in fr.scan(), datum:", datum)
		fmt.Println("in fr.scan(), err:", err)
		record := datum.(*goavro.Record)
		for _, f := range record.Fields {
			fmt.Println("record.Name:", f.Name)
			fmt.Println("record.Datum:", f.Datum)
		}
		if err != nil {
			log.Println("can not read datum:", err)
			continue
		}
		fmt.Println("ipcreauest: ", datum)
	}

	fw, err := goavro.NewRpcResponseWriter(
		// goavro.Compression(goavro.CompressionSnappy),
		responseSchema,
		goavro.Compression(goavro.CompressionNull),
		goavro.WriterSchema(responseSchema),
		goavro.BufferToWriter(w))

	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		fw.Close()
	}()

	messages := make([]interface{}, 0)
	respMessageRecord, err := goavro.NewRecord(goavro.RecordSchema(respMessageSchema))
	if err != nil {
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
	responseRecord.Set("messages", messages)

	fw.Write(responseRecord)
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
	requestSchema = string(req)
	// requestCodec, err = goavro.NewCodec(requestSchema)
	reqMessage, err := ioutil.ReadFile("req_message.avsc")
	reqMessageSchema = string(reqMessage)

	respmsg, err := ioutil.ReadFile("resp_message.avsc")
	respMessageSchema = string(respmsg)

	resp, err := ioutil.ReadFile("response.avsc")
	responseSchema = string(resp)
	responseCodec, err = goavro.NewCodec(responseSchema)

	if err != nil {
		log.Fatal(err)
	}
}
