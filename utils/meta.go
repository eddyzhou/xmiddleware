package utils

import (
	"encoding/base64"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

const (
	binSuffix = "-bin"
)

func Get(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return "", false
	}
	key = strings.ToLower(key)
	valSlice, ok := md[key]
	if !ok {
		return "", false
	}
	if len(valSlice) != 1 {
		return "", false
	}
	return valSlice[0], true
}

func Set(ctx context.Context, key string, value string) context.Context {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = metadata.Pairs(key, value)
		return metadata.NewContext(ctx, md)
	}
	k, v := encode(key, value)
	md[k] = []string{v}
	return ctx // we use the same context because we modified the metadata in place.
}

func encode(k, v string) (string, string) {
	k = strings.ToLower(k)
	if strings.HasSuffix(k, binSuffix) {
		val := base64.StdEncoding.EncodeToString([]byte(v))
		v = string(val)
	}
	return k, v
}
