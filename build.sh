DST_DIR=pkg/protocol
SRC_DIR=pkg/protocol
protoc -I=$SRC_DIR --go_out=$DST_DIR $SRC_DIR/request_response.proto