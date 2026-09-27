//go:build ignore

// gzip compresses one file: go run scripts/gzip.go <src> <dst>.
package main

import (
	"compress/gzip"
	"io"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: go run scripts/gzip.go <src> <dst>")
	}
	in, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	zw, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := io.Copy(zw, in); err != nil {
		log.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		log.Fatal(err)
	}
	if err := out.Close(); err != nil {
		log.Fatal(err)
	}
}
