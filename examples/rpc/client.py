#!/usr/bin/env python
# -*- coding: utf-8 -*-


from httplib import HTTPConnection
from StringIO import StringIO
import avro.schema as schema
import avro.io as io
from avro.datafile import DataFileReader


schema = schema.parse(open("simple.avsc").read())

server_addr = ('localhost', 8888)

class UsageError(Exception):
    def __init__(self, value):
        self.value = value
    def __str__(self):
        return repr(self.value)

if __name__ == '__main__':
    # client code - attach to the server and send a message
    conn = HTTPConnection(server_addr[0], server_addr[1])
    conn.request("POST", "/")
    resp = conn.getresponse()

    data = resp.read()
    reader = DataFileReader(StringIO(data), io.DatumReader())

    for a in reader:
        print a
    reader.close()

    # # ipc 方式NOT work
    # buffer_reader = StringIO(data)
    # buffer_decoder = io.BinaryDecoder(buffer_reader)

    # reader = io.DatumReader(schema, schema)
    # for a in reader.read(buffer_decoder):
        # print a
    # reader.close()
