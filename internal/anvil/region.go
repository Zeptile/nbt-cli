package anvil

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nbt-cli/internal/coords"

	"github.com/Tnze/go-mc/nbt"
)

const sectorSize = 4096

type Region struct {
	path string
	f    *os.File
	mu   sync.Mutex
}

func OpenRegionFile(path string) (*Region, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}
	return &Region{path: path, f: f}, nil
}

func OpenRegionForWorldXZ(regionDir string, x, z int) (*Region, string, error) {
	rx, rz := coords.WorldToRegionXZ(x, z)
	name := coords.RegionFileName(rx, rz)
	path := filepath.Join(regionDir, name)
	reg, err := OpenRegionFile(path)
	return reg, path, err
}

func (r *Region) Close() error {
	return r.f.Close()
}

func indexFor(cx, cz int) int { return cz*32 + cx }

func (r *Region) readHeaders() ([]byte, []byte, error) {
	buf := make([]byte, sectorSize*2)
	if _, err := r.f.ReadAt(buf, 0); err != nil {
		return nil, nil, err
	}
	return buf[:sectorSize], buf[sectorSize:], nil
}

func (r *Region) getLocation(cx, cz int) (offsetSectors int64, count int, err error) {
	loc, _, err := r.readHeaders()
	if err != nil {
		return 0, 0, err
	}
	idx := indexFor(cx, cz) * 4
	entry := loc[idx : idx+4]
	offset := int64(entry[0])<<16 | int64(entry[1])<<8 | int64(entry[2])
	count = int(entry[3])
	return offset, count, nil
}

func (r *Region) setLocation(cx, cz int, offset int64, count int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	loc := make([]byte, sectorSize)
	if _, err := r.f.ReadAt(loc, 0); err != nil {
		return err
	}
	idx := indexFor(cx, cz) * 4
	loc[idx+0] = byte((offset >> 16) & 0xFF)
	loc[idx+1] = byte((offset >> 8) & 0xFF)
	loc[idx+2] = byte(offset & 0xFF)
	loc[idx+3] = byte(count)
	if _, err := r.f.WriteAt(loc, 0); err != nil {
		return err
	}
	// timestamp
	ts := make([]byte, sectorSize)
	binary.BigEndian.PutUint32(ts[indexFor(cx, cz)*4:indexFor(cx, cz)*4+4], uint32(time.Now().Unix()))
	if _, err := r.f.WriteAt(ts, sectorSize); err != nil {
		return err
	}
	return nil
}

func (r *Region) fileSectorCount() (int64, error) {
	st, err := r.f.Stat()
	if err != nil {
		return 0, err
	}
	return (st.Size() + sectorSize - 1) / sectorSize, nil
}

func (r *Region) buildUsedSectors() ([]bool, error) {
	sectorCount, err := r.fileSectorCount()
	if err != nil {
		return nil, err
	}
	used := make([]bool, sectorCount)
	if sectorCount >= 2 {
		used[0] = true
		used[1] = true
	}
	loc, _, err := r.readHeaders()
	if err != nil {
		return nil, err
	}
	for i := range 1024 {
		entry := loc[i*4 : i*4+4]
		off := int(entry[0])<<16 | int(entry[1])<<8 | int(entry[2])
		cnt := int(entry[3])
		if off == 0 || cnt == 0 {
			continue
		}
		for s := 0; s < cnt && int64(off+s) < int64(len(used)); s++ {
			used[off+s] = true
		}
	}
	return used, nil
}

func (r *Region) findFreeRun(need int) (int64, error) {
	used, err := r.buildUsedSectors()
	if err != nil {
		return 0, err
	}
	// search
	freeLen := 0
	start := 0
	for i := 2; i < len(used); i++ { // skip header
		if !used[i] {
			if freeLen == 0 {
				start = i
			}
			freeLen++
			if freeLen >= need {
				return int64(start), nil
			}
		} else {
			freeLen = 0
		}
	}
	// append
	count, err := r.fileSectorCount()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Region) ReadChunkNBT(cx, cz int) (map[string]any, error) {
	off, cnt, err := r.getLocation(cx, cz)
	if err != nil {
		return nil, err
	}
	if off == 0 || cnt == 0 {
		return nil, errors.New("chunk not present in region")
	}
	pos := off * sectorSize
	header := make([]byte, 5)
	if _, err := r.f.ReadAt(header, pos); err != nil {
		return nil, err
	}
	length := int64(binary.BigEndian.Uint32(header[:4]))
	ctype := header[4]
	if length <= 0 || length > int64(cnt*sectorSize) {
		return nil, fmt.Errorf("invalid chunk length %d", length)
	}
	comp := make([]byte, length-1)
	if _, err := r.f.ReadAt(comp, pos+5); err != nil {
		return nil, err
	}
	var reader io.ReadCloser
	switch ctype {
	case 2:
		zr, err := zlib.NewReader(bytes.NewReader(comp))
		if err != nil {
			return nil, err
		}
		reader = zr
	case 1:
		gr, err := gzip.NewReader(bytes.NewReader(comp))
		if err != nil {
			return nil, err
		}
		reader = gr
	default:
		return nil, fmt.Errorf("unsupported compression type %d", ctype)
	}
	defer reader.Close()
	var chunk map[string]any
	dec := nbt.NewDecoder(reader)
	if _, err := dec.Decode(&chunk); err != nil {
		return nil, err
	}
	return chunk, nil
}

func (r *Region) WriteChunkNBT(cx, cz int, chunk map[string]any) error {
	// encode
	var nbtBuf bytes.Buffer
	enc := nbt.NewEncoder(&nbtBuf)
	if err := enc.Encode(chunk, ""); err != nil {
		return err
	}
	var compBuf bytes.Buffer
	zw := zlib.NewWriter(&compBuf)
	if _, err := zw.Write(nbtBuf.Bytes()); err != nil {
		zw.Close()
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	record := make([]byte, 5+compBuf.Len())
	binary.BigEndian.PutUint32(record[:4], uint32(compBuf.Len()+1))
	record[4] = 2
	copy(record[5:], compBuf.Bytes())
	need := (len(record) + sectorSize - 1) / sectorSize

	oldOff, oldCnt, err := r.getLocation(cx, cz)
	if err != nil {
		return err
	}
	writeOff := oldOff
	writeCnt := oldCnt
	if oldOff == 0 || oldCnt < need {
		off, err := r.findFreeRun(need)
		if err != nil {
			return err
		}
		writeOff = off
		writeCnt = need
	}
	// write
	pos := writeOff * sectorSize
	padSize := writeCnt*sectorSize - len(record)
	buf := make([]byte, 0, writeCnt*sectorSize)
	buf = append(buf, record...)
	if padSize > 0 {
		buf = append(buf, make([]byte, padSize)...)
	}
	if _, err := r.f.WriteAt(buf, pos); err != nil {
		return err
	}
	if err := r.setLocation(cx, cz, writeOff, writeCnt); err != nil {
		return err
	}
	return nil
}


