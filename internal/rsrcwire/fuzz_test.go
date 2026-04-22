package rsrcwire

import (
	"encoding/binary"
	"os"
	"testing"

	"github.com/CWBudde/lvrsrc/internal/binaryx"
	"github.com/CWBudde/lvrsrc/internal/corpus"
)

var corpusSeeds = []string{
	"config-data.ctl",
	"get-vi-description.vi",
	"action.ctl",
	"request.ctl",
	"module-data--cluster.ctl",
	"is-int.vi",
	"module-timeout--constant.vi",
	"ndjson-parser.vi",
	"write-ini.vi",
}

func FuzzParseFile(f *testing.F) {
	f.Add([]byte("RSRC"))
	for _, name := range corpusSeeds {
		addFileSeed(f, name)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}

func FuzzParseHeader(f *testing.F) {
	f.Add([]byte("RSRC\r\n\x00\x03LVINLBVW\x00\x00\x00 \x00\x00\x00 \x00\x00\x00 \x00\x00\x00 "))
	for _, name := range corpusSeeds {
		addHeaderSeed(f, name)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseHeader(data)
	})
}

func FuzzNameTable(f *testing.F) {
	f.Add([]byte{5, 'a', 'l', 'p', 'h', 'a', 5, 'b', 'e', 't', 'a', '!'}, uint16(0), uint16(6))
	f.Add([]byte{15, 'C', 'o', 'n', 'f', 'i', 'g', ' ', 'D', 'a', 't', 'a', '.', 'c', 't', 'l'}, uint16(0), uint16(0))

	f.Fuzz(func(t *testing.T, data []byte, off1, off2 uint16) {
		r := binaryx.NewReader(data, binary.BigEndian)
		offsets := make(map[uint32]struct{})
		if len(data) > 0 {
			offsets[uint32(int(off1)%len(data))] = struct{}{}
			offsets[uint32(int(off2)%len(data))] = struct{}{}
		}
		_, _, _, _ = parseNames(r, 0, int64(len(data)), offsets, &parseState{})
	})
}

func addFileSeed(f *testing.F, name string) {
	f.Helper()

	data, err := os.ReadFile(corpus.Path(name))
	if err == nil {
		f.Add(data)
	}
}

func addHeaderSeed(f *testing.F, name string) {
	f.Helper()

	data, err := os.ReadFile(corpus.Path(name))
	if err == nil && len(data) >= headerSize {
		f.Add(append([]byte(nil), data[:headerSize]...))
	}
}
