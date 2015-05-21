#!/usr/bin/env python

# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements.  See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership.  The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import sys
import httplib

import avro.ipc as ipc
import avro.protocol as protocol

PROTOCOL = protocol.parse(open("Agent.avsc").read())

server_addr = ('localhost', 8888)

if __name__ == '__main__':
    # client code - attach to the server and send a message
    client = ipc.HTTPTransceiver(server_addr[0], server_addr[1], '/heartbeat')
    requestor = ipc.Requestor(PROTOCOL, client)

    # fill in the Message record and send it
    req = dict()
    req['version'] = 1
    req['host_id'] = "192.168.1.169"
    req['status_hash'] = "falkjdfaksjdf"
    req['status'] = None
    req['last_response_hash'] = "sfalf"
    req['host_stats'] = None
    req['process_stats'] = []
    req['total_cpu'] = 3

    params = dict()
    params['request'] = req
    print("Result: " + str(requestor.request('heartbeat', params)))

    # cleanup
    client.close()
