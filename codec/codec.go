// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import (
	"bufio"
	"errors"
	"io"
	"path/filepath"
	"strings"
)

// ErrFormat indicates that decoding encountered an unknown format.
var ErrFormat = errors.New("codec: unknown format")

type codec struct {
	name, magic string
	extensions  []string
	decode      func(Reader) ([]Song, error)
}

// Codecs is the list of registered codecs.
var codecs []codec

var allExtensions = make(map[string]codec)

// RegisterCodec registers an audio codec for use by Decode.
// Name is the name of the format, like "nsf" or "wav".
// Magic is the magic prefix that identifies the codec's encoding. The magic
// string can contain "?" wildcards that each match any one byte.
// Decode is the function that decodes the encoded codec.
func RegisterCodec(name, magic string, extensions []string, decode func(Reader) ([]Song, error)) {
	c := codec{
		name:       name,
		magic:      magic,
		extensions: extensions,
		decode:     decode,
	}
	for _, e := range extensions {
		allExtensions[e] = c
	}
	codecs = append(codecs, c)
}

// A reader is an io.Reader that can also peek ahead.
type reader interface {
	io.Reader
	Peek(int) ([]byte, error)
}

// asReader converts an io.Reader to a reader.
func asReader(r io.Reader) reader {
	if rr, ok := r.(reader); ok {
		return rr
	}
	return bufio.NewReader(r)
}

// Match returns whether magic matches b. Magic may contain "?" wildcards.
func match(magic string, b []byte) bool {
	if len(magic) != len(b) {
		return false
	}
	for i, c := range b {
		if magic[i] != c && magic[i] != '?' {
			return false
		}
	}
	return true
}

// Sniff determines the format of r's data.
func sniff(r reader) codec {
	for _, f := range codecs {
		b, err := r.Peek(len(f.magic))
		if err == nil && match(f.magic, b) {
			return f
		}
	}
	return codec{}
}

// Reader returns a file reader and the file size in bytes (or 0 if streamed
// or unknown).
type Reader func() (io.ReadCloser, int64, error)

// Decode decodes audio that has been encoded in a registered codec.
// The string returned is the format name used during format registration.
// Format registration is typically done by the init method of the codec-
// specific package.
func Decode(rf Reader) ([]Song, string, error) {
	r, _, err := rf()
	if err != nil {
		return nil, "", err
	}
	defer r.Close()
	rr := asReader(r)
	f := sniff(rr)
	if f.decode == nil {
		return nil, "", ErrFormat
	}
	m, err := f.decode(rf)
	return m, f.name, err
}

func ByExtension(path string, rf Reader) ([]Song, string, error) {
	ext := filepath.Ext(path)
	ext = strings.Trim(ext, ".")
	if ext == "" {
		ext = path
	}
	c, ok := allExtensions[ext]
	if !ok {
		return nil, "", nil
	}
	songs, err := c.decode(rf)
	return songs, c.name, err
}
