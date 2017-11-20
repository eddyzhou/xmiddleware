package interceptor

import (
	"bytes"
	"fmt"
	"reflect"
	"time"

	"github.com/eddyzhou/log"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var (
	js = &jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true, OrigName: true}
)

// Logging interceptor for grpc
func Logging(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	start := time.Now()

	log.Printf("calling %s, req=%s", info.FullMethod, marshal(req))
	resp, err = handler(ctx, req)
	log.Printf("finished %s, took=%v, resp=%v, err=%v", info.FullMethod, time.Since(start), resp, err)

	return resp, err
}

// marshal converts a protocol buffer object to JSON string.
func marshal(x interface{}) string {
	if x == nil || reflect.ValueOf(x).IsNil() {
		return fmt.Sprintf("<nil>")
	}

	pb, ok := x.(proto.Message)
	if !ok {
		return fmt.Sprintf("Marshal to json error: not a proto message")
	}

	var buf bytes.Buffer
	if err := js.Marshal(&buf, pb); err != nil {
		return fmt.Sprintf("Marshal to json error: %s", err.Error())
	}
	return buf.String()
}
