package main

import (
	"testing"

	"google.golang.org/grpc/encoding"
)

func TestGZIPDecompressorRegistered(t *testing.T) {
	if encoding.GetCompressor("gzip") == nil {
		t.Fatal("gzip compressor is not registered")
	}
}
