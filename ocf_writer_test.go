// Copyright 2015 LinkedIn Corp. Licensed under the Apache License,
// Version 2.0 (the "License"); you may not use this file except in
// compliance with the License.  You may obtain a copy of the License
// at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.Copyright [201X] LinkedIn Corp. Licensed under the Apache
// License, Version 2.0 (the "License"); you may not use this file
// except in compliance with the License.  You may obtain a copy of
// the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.

package goavro

import (
	"bytes"
	"io"
	"testing"
)

var defaultSync []byte

func init() {
	defaultSync = []byte("\x21\x0f\xc7\xbb\x81\x86\x39\xac\x48\xa4\xc6\xaf\xa2\xf1\x58\x1a")
}

func TestNewWriterBailsUnsupportedCodec(t *testing.T) {
	var err error
	_, err = NewWriter(ToWriter(new(bytes.Buffer)), Compression(""))
	checkError(t, err, "unsupported codec")

	_, err = NewWriter(ToWriter(new(bytes.Buffer)), Compression("ficticious test codec name"))
	checkError(t, err, "unsupported codec")
}

func TestNewWriterBailsMissingWriterSchema(t *testing.T) {
	var err error
	_, err = NewWriter(ToWriter(new(bytes.Buffer)))
	checkError(t, err, "missing schema")

	_, err = NewWriter(ToWriter(new(bytes.Buffer)), Compression(CompressionNull))
	checkError(t, err, "missing schema")

	_, err = NewWriter(ToWriter(new(bytes.Buffer)), Compression(CompressionDeflate))
	checkError(t, err, "missing schema")

	_, err = NewWriter(ToWriter(new(bytes.Buffer)), Compression(CompressionSnappy))
	checkError(t, err, "missing schema")
}

func TestNewWriterBailsInvalidWriterSchema(t *testing.T) {
	_, err := NewWriter(WriterSchema("this should not compile"))
	checkError(t, err, "cannot parse schema")
}

func TestNewWriterBailsBadSync(t *testing.T) {
	_, err := NewWriter(WriterSchema(`"int"`), Sync(make([]byte, 0)))
	checkError(t, err, "sync marker ought to be 16 bytes long")

	_, err = NewWriter(WriterSchema(`"int"`), Sync(make([]byte, syncLength-1)))
	checkError(t, err, "sync marker ought to be 16 bytes long")

	_, err = NewWriter(WriterSchema(`"int"`), Sync(make([]byte, syncLength+1)))
	checkError(t, err, "sync marker ought to be 16 bytes long")
}

func TestNewWriterCreatesRandomSync(t *testing.T) {
	bb := new(bytes.Buffer)
	func(w io.Writer) {
		fw, err := NewWriter(ToWriter(w), WriterSchema(`"int"`))
		if err != nil {
			t.Fatalf("Actual: %#v; Expected: %#v", err, nil)
		}
		defer fw.Close()
	}(bb)

	notExpected := []byte("\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	actual := bb.Bytes()
	actual = actual[len(actual)-syncLength:]
	if bytes.Compare(actual, notExpected) == 0 {
		t.Errorf("Actual: %#v; Expected: some non-zero value bits", actual)
	}
}

func TestWriteHeaderCustomSync(t *testing.T) {
	bb := new(bytes.Buffer)
	func(w io.Writer) {
		fw, err := NewWriter(ToWriter(w), WriterSchema(`"int"`), Sync(defaultSync))
		if err != nil {
			t.Fatalf("Actual: %#v; Expected: %#v", err, nil)
		}
		fw.Close()
	}(bb)

	// NOTE: because key value pair ordering is indeterminate,
	// there are two valid possibilities for the encoded map:
	option1 := []byte("Obj\x01\x14avro.codec\x08null\x16avro.schema\x0a\x22int\x22\x00\x21\x0f\xc7\xbb\x81\x86\x39\xac\x48\xa4\xc6\xaf\xa2\xf1\x58\x1a")
	option2 := []byte("Obj\x01\x16avro.schema\x0a\x22int\x22\x14avro.codec\x08null\x00\x21\x0f\xc7\xbb\x81\x86\x39\xac\x48\xa4\xc6\xaf\xa2\xf1\x58\x1a")

	actual := bb.Bytes()
	if (bytes.Compare(actual, option1) != 0) && (bytes.Compare(actual, option2) != 0) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, option1)
	}
}

func TestWriteWithNullCodec(t *testing.T) {
	bb := new(bytes.Buffer)

	fw, err := NewWriter(ToWriter(bb), WriterSchema(`"int"`), Sync(defaultSync))
	if err != nil {
		t.Fatalf("Actual: %#v; Expected: %#v", err, nil)
	}

	doing := func(fw *Writer) {
		defer func() {
			fw.Close()
		}()
		fw.Write(int32(13))
		fw.Write(int32(42))
		fw.Write(int32(54))
		fw.Write(int32(99))
	}
	doing(fw)

	<-fw.WriterDone // 必须等待才能写完，否则下面测试有并发错误，有时候过有时候不过
	t.Logf("bb: %+v", bb.Bytes())

	// NOTE: because key value pair ordering is indeterminate,
	// there are two valid possibilities for the encoded map:
	option1 := []byte("Obj\x01\x14avro.codec\x08null\x16avro.schema\x0a\x22int\x22\x00" + string(defaultSync) + "\x08\x0a\x1a\x54\x6c\xc6\x01" + string(defaultSync))
	option2 := []byte("Obj\x01\x16avro.schema\x0a\x22int\x22\x14avro.codec\x08null\x00" + string(defaultSync) + "\x08\x0a\x1a\x54\x6c\xc6\x01" + string(defaultSync))

	actual := bb.Bytes()
	if (bytes.Compare(actual, option1) != 0) && (bytes.Compare(actual, option2) != 0) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, option1)
		t.Errorf("Actual: %#v; Expected: %#v", actual, option2)
	}
}

func _TestWriteWithDeflateCodec(t *testing.T) {
	bb := new(bytes.Buffer)
	func(w io.Writer) {
		fw, err := NewWriter(
			BlockSize(2),
			Compression(CompressionDeflate),
			WriterSchema(`"int"`),
			Sync(defaultSync),
			ToWriter(w))
		if err != nil {
			t.Fatalf("Actual: %#v; Expected: %#v", err, nil)
		}
		defer fw.Close()
		fw.Write(int32(13))
		fw.Write(int32(42))
		fw.Write(int32(54))
		fw.Write(int32(99))
	}(bb)

	// NOTE: because key value pair ordering is indeterminate,
	// there are two valid possibilities for the encoded map:
	option1 := []byte("Obj\x01\x04\x14avro.codec\x08null\x16avro.schema\x0a\x22int\x22\x00" + string(defaultSync) + "\x08\x0a\x1a\x54\x6c\xc6\x01" + string(defaultSync) + "\x00\x00")
	option2 := []byte("Obj\x01\x04\x16avro.schema\x0a\x22int\x22\x14avro.codec\x08null\x00" + string(defaultSync) + "\x08\x0a\x1a\x54\x6c\xc6\x01" + string(defaultSync) + "\x00\x00")

	actual := bb.Bytes()
	if (bytes.Compare(actual, option1) != 0) && (bytes.Compare(actual, option2) != 0) {
		t.Errorf("Actual: %#v; Expected: %#v", actual, option1)
	}
}
