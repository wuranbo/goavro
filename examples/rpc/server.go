// Package main provides ...
package main

import (
	"log"
	"net/http"

	"github.com/wuranbo/goavro"
)

var codec goavro.Codec
var recordSchema string = `
{
	"type": "record",
	"name": "comments",
	"doc:": "A basic schema for storing blog comments",
	"namespace": "com.example",
	"fields": [
	{
		"doc": "Name of user",
		"type": "string",
		"name": "username"
	},
	{
		"doc": "The content of the user's message",
		"type": "string",
		"name": "comment"
	},
	{
		"doc": "Unix epoch time in milliseconds",
		"type": "long",
		"name": "timestamp"
	}
	]
}
`

type Hello struct{}

func (h Hello) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request) {

	fw, err := codec.NewWriter(
		// goavro.Compression(goavro.CompressionSnappy),
		goavro.Compression(goavro.CompressionNull),
		goavro.WriterSchema(recordSchema),
		goavro.ToWriter(w))

	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		fw.Close()
		<-fw.WriterDone
		// w.Close()
	}()

	someRecord, err := goavro.NewRecord(goavro.RecordSchema(recordSchema))
	if err != nil {
		log.Fatal(err)
	}

	someRecord.Set("username", "goodman")
	someRecord.Set("comment", "goodman die loney.")
	someRecord.Set("timestamp", int64(1))
	fw.Write(someRecord)

	if someRecord, err = goavro.NewRecord(goavro.RecordSchema(recordSchema)); err != nil {
		log.Fatal(err)
	}
	someRecord.Set("username", "badman")
	someRecord.Set("comment", "badman is bad.")
	someRecord.Set("timestamp", int64(2))
	fw.Write(someRecord)

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
	codec, err = goavro.NewCodec(recordSchema)
	if err != nil {
		log.Fatal(err)
	}

}
