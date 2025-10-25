package anvil

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"nbt-cli/internal/coords"

	"github.com/Tnze/go-mc/nbt"
)

func writeTestRegion(t *testing.T, path string, chunk map[string]any) {
	t.Helper()

	locTable := make([]byte, sectorSize)
	tsTable := make([]byte, sectorSize)

	locIdx := indexFor(0, 0) * 4
	locTable[locIdx] = 0
	locTable[locIdx+1] = 0
	locTable[locIdx+2] = 2
	locTable[locIdx+3] = 1
	// fixed timestamp
	binary.BigEndian.PutUint32(tsTable[locIdx:locIdx+4], 0x01020304)

	var nbtBuf bytes.Buffer
	enc := nbt.NewEncoder(&nbtBuf)
	if err := enc.Encode(chunk, ""); err != nil {
		t.Fatalf("encode chunk: %v", err)
	}
	var compBuf bytes.Buffer
	zw := zlib.NewWriter(&compBuf)
	if _, err := zw.Write(nbtBuf.Bytes()); err != nil {
		t.Fatalf("compress chunk: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close compressor: %v", err)
	}
	record := make([]byte, 5+compBuf.Len())
	binary.BigEndian.PutUint32(record[:4], uint32(compBuf.Len()+1))
	record[4] = 2
	copy(record[5:], compBuf.Bytes())
	if len(record) > sectorSize {
		t.Fatalf("test record exceeds sector size")
	}
	chunkSector := make([]byte, sectorSize)
	copy(chunkSector, record)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create region file: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(locTable); err != nil {
		t.Fatalf("write location table: %v", err)
	}
	if _, err := f.Write(tsTable); err != nil {
		t.Fatalf("write timestamp table: %v", err)
	}
	if _, err := f.Write(chunkSector); err != nil {
		t.Fatalf("write chunk sector: %v", err)
	}
}

func TestOpenRegionFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "r.0.0.mca")
	if err := os.WriteFile(path, make([]byte, sectorSize*2), 0o666); err != nil {
		t.Fatalf("write empty region: %v", err)
	}
	reg, err := OpenRegionFile(path)
	if err != nil {
		t.Fatalf("OpenRegionFile: %v", err)
	}
	if err := reg.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestOpenRegionForWorldXZ(t *testing.T) {
	tmp := t.TempDir()
	rx, rz := coords.WorldToRegionXZ(256, -768)
	path := filepath.Join(tmp, coords.RegionFileName(rx, rz))
	if err := os.WriteFile(path, make([]byte, sectorSize*2), 0o666); err != nil {
		t.Fatalf("write empty region: %v", err)
	}
	reg, gotPath, err := OpenRegionForWorldXZ(tmp, 256, -768)
	if err != nil {
		t.Fatalf("OpenRegionForWorldXZ: %v", err)
	}
	if gotPath != path {
		t.Fatalf("path mismatch: got %s want %s", gotPath, path)
	}
	reg.Close()
}

func TestReadWriteChunkNBT(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "r.0.0.mca")
	chunk := map[string]any{"Level": map[string]any{"Status": "full"}}
	writeTestRegion(t, path, chunk)

	reg, err := OpenRegionFile(path)
	if err != nil {
		t.Fatalf("open region: %v", err)
	}
	defer reg.Close()

	data, err := reg.ReadChunkNBT(0, 0)
	if err != nil {
		t.Fatalf("ReadChunkNBT: %v", err)
	}
	level, ok := data["Level"].(map[string]any)
	if !ok || level["Status"] != "full" {
		t.Fatalf("unexpected chunk contents: %#v", data)
	}

	level["Status"] = "post"
	if err := reg.WriteChunkNBT(0, 0, data); err != nil {
		t.Fatalf("WriteChunkNBT: %v", err)
	}

	data2, err := reg.ReadChunkNBT(0, 0)
	if err != nil {
		t.Fatalf("ReadChunkNBT second read: %v", err)
	}
	level2, ok := data2["Level"].(map[string]any)
	if !ok || level2["Status"] != "post" {
		t.Fatalf("expected status 'post', got %#v", data2)
	}
}

