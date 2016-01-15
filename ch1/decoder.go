package drum

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
)

var (
	InvalidHeader = errors.New("invalid header")
)

// DecodeFile decodes the drum machine file found at the provided path
// and returns a pointer to a parsed pattern which is the entry point to the
// rest of the data.
// TODO: implement
func DecodeFile(path string) (*Pattern, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return NewDecoder(file).Decode()
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

type errReader struct {
	r   io.Reader
	err error
}

func (r *errReader) Read(order binary.ByteOrder, data interface{}) error {
	if r.err != nil {
		return r.err
	}

	return binary.Read(r.r, order, data)
}

func (r *errReader) ReadFull(buf []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	return io.ReadFull(r.r, buf)
}

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
	er.Read(binary.BigEndian, &header)
	er.Read(binary.LittleEndian, &tempo)

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

	er.r = io.LimitReader(d.r, header.Size-36)

	p := &Pattern{
		Version: version,
		Tempo:   tempo,
	}

loop:
	for {
		var id uint32
		err := er.Read(binary.BigEndian, &id)

		switch {
		case err == io.EOF:
			break loop // done reading

		case err != nil:
			return nil, err
		}

		var len byte
		er.Read(binary.BigEndian, &len)

		instrument := make([]byte, len)
		steps := make([]byte, 16)

		er.ReadFull(instrument)
		er.ReadFull(steps)

		if er.err != nil {
			return nil, er.err
		}
	}

	return p, nil
}

// Pattern is the high level representation of the
// drum pattern contained in a .splice file.
// TODO: implement
type Pattern struct {
	Version string
	Tempo   float32
}
