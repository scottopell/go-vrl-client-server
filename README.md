

## Target Configuration
- Start vector with a 'source' and 'sink' that both point to local unix domain sockets
- Data will be fed in via `go-vector-input.sock` as JSON logs
- Data will be emitted to an output socket `go-vector-result.sock`
- TODO - configure desired VRL program

### Simple socket-in, socket-out
Vector config:
```
[sources.from-the-agent]
type = "socket"
mode = "unix_stream"
decoding.codec = "json"
path = "/tmp/go-vector-input.sock"

[transforms.remap_agentlogs]
inputs = [ "from-the-agent"]
type = "remap"
source = '''
    . = replace(string!(.message), r'\b\w{4}\b', "rust")
'''

[sinks.back-to-the-agent]
type = "socket"
inputs = ["remap_agentlogs"]
mode = "unix"
encoding.codec = "json"
path = "/tmp/go-vector-result.sock"
```

Connect to input socket and send fake logs:
```
$ ~/dev/flog/flog -l -b 1024 -r 5 | awk '{ printf "{ \"message\": \" %s \" }\n",$0 }' | socat UNIX-CONNECT:/tmp/go-vector-input.sock -
```

Connect to output socket and recieve the logs to stdout:
```
socat UNIX-LISTEN:/tmp/go-vector-result.sock -
```

## Observations
- Vector's socket source takes `unix_datagram` or `unix_stream`, but vector's socket sink takes "unix"
- Some vector socket messages do not contain the socket path, which makes debugging hard. Will submit PR for this.

Before:
```
2022-11-21T16:24:22.337399Z  INFO vector::topology::builder: Healthcheck: Passed.
thread 'vector-worker' panicked at 'Failed to bind to listener socket: Os { code: 48, kind: AddrInUse, message: "Address already in use" }', src/sources/util/unix_stream.rs:46:57
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
2022-11-21T16:24:22.337648Z  INFO vector_common::shutdown: All sources have finished.
2022-11-21T16:24:22.337659Z  INFO vector: Vector has stopped.
2022-11-21T16:24:22.337664Z ERROR source{component_kind="source" component_id=from-the-agent component_type=socket component_name=from-the-agent}: vector::topology: An error occurred that Vector couldn't handle: the task panicked and was aborted.
```

After:
```
2022-11-21T16:36:31.081277Z ERROR vector::topology::builder: msg="Healthcheck: Failed Reason." error=Failed connecting to socket at path /tmp/go-vector-result.sock: No such file or directory (os error 2) component_kind="sink" component_type="socket" component_id=back-to-the-agent component_name=back-to-the-agent
2022-11-21T16:36:31.081314Z  INFO vector: Vector has started. debug="false" version="0.26.0" arch="aarch64" revision=""
2022-11-21T16:36:31.081424Z  INFO vector::app: API is disabled, enable by setting `api.enabled` to `true` and use commands like `vector top`.
thread 'vector-worker' panicked at 'Failed to bind to listener socket at path: /tmp/go-vector-input.sock', src/sources/util/unix_stream.rs:46:76
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
2022-11-21T16:36:31.082430Z  INFO vector_common::shutdown: All sources have finished.
2022-11-21T16:36:31.082437Z  INFO vector: Vector has stopped.
2022-11-21T16:36:31.082485Z ERROR source{component_kind="source" component_id=from-the-agent component_type=socket component_name=from-the-agent}: vector::topology: An error occurred that Vector couldn't handle: the task panicked and was aborted.
```