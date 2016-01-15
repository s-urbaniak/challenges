package drum

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
)

var (
	// InvalidHeader is the error returned by Decode
	// when a drum machine stream contains an invalid header.
	InvalidHeader = errors.New("invalid header")
)

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
func DecodeFile(path string) (*Pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return NewDecoder(file).Decode()
}

// Decoder decodes a pattern from a reader.
type Decoder struct {
	r io.Reader
}

// NewDecoder returns a new pattern decoder using the given reader.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode decodes a pattern from the underlying reader
// and returns the decoded pattern.
func (d *Decoder) Decode() (*Pattern, error) {
	var (
		header struct {
			Splice  [6]byte
			Size    int64
			Version [32]byte
		}
		tempo float32
	)

	er := errReader{d.r, nil}
	er.read(binary.BigEndian, &header)
	er.read(binary.LittleEndian, &tempo)

	switch {
	case er.err != nil:
		return nil, er.err
	case string(header.Splice[:]) != "SPLICE":
		return nil, InvalidHeader
	}

	version := strings.TrimRight(
		string(header.Version[:]),
		string(0), // trim zero-byte values
	)

	// use limitreader limited by header size minus
	// 32 bytes header.Version + 4 bytes tempo
	er.r = io.LimitReader(d.r, header.Size-36)

	p := &Pattern{
		Version: version,
		Tempo:   tempo,
		Tracks:  []Track{},
	}

loop:
	for {
		var id uint32
		err := er.read(binary.LittleEndian, &id)

		switch {
		case err == io.EOF:
			break loop // done reading
		case err != nil:
			return nil, err
		}

		var len byte
		er.read(binary.LittleEndian, &len)
		instrument := make([]byte, len)
		steps := make([]byte, 16)
		er.readFull(instrument)
		er.readFull(steps)

		if er.err != nil {
			return nil, er.err
		}

		t := Track{
			ID:         id,
			Instrument: string(instrument),
			Steps:      steps,
		}

		p.Tracks = append(p.Tracks, t)
	}

	return p, nil
}

// errReader is an error aware reader being a little helper to avoid err != nil checks.
// It is not goroutine-safe.
type errReader struct {
	r   io.Reader
	err error
}

// read reads structured binary data into data using binary.Read
// returning prematurely if an error already happened.
func (r *errReader) read(order binary.ByteOrder, data interface{}) error {
	if r.err != nil {
		return r.err
	}

	r.err = binary.Read(r.r, order, data)
	return r.err
}

// readFull reads exactly len(buf) bytes into buf using io.ReadFull
// returning prematurely if an error already happened.
func (r *errReader) readFull(buf []byte) (n int, _ error) {
	if r.err != nil {
		return 0, r.err
	}

	n, r.err = io.ReadFull(r.r, buf)
	return n, r.err
}
