package drum

import (
	"encoding/binary"
	"errors"
	"fmt"
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

	r.err = binary.Read(r.r, order, data)
	return r.err
}

func (r *errReader) ReadFull(buf []byte) (n int, _ error) {
	if r.err != nil {
		return 0, r.err
	}

	n, r.err = io.ReadFull(r.r, buf)
	return n, r.err
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
		err := er.Read(binary.LittleEndian, &id)

		switch {
		case err == io.EOF:
			break loop // done reading
		case err != nil:
			return nil, err
		}

		var len byte
		er.Read(binary.LittleEndian, &len)
		instrument := make([]byte, len)
		steps := make([]byte, 16)
		er.ReadFull(instrument)
		er.ReadFull(steps)

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

// Pattern is the high level representation of the
// drum pattern contained in a .splice file.
type Pattern struct {
	Version string
	Tempo   float32
	Tracks  []Track
}

func (p Pattern) String() string {
	s := fmt.Sprintf(
		"Saved with HW Version: %s\nTempo: %g\n",
		p.Version, p.Tempo,
	)

	for _, t := range p.Tracks {
		s += t.String() + "\n"
	}

	return s

}

type Track struct {
	ID         uint32
	Instrument string
	Steps      Steps
}

func (t Track) String() string {
	return fmt.Sprintf(
		"(%d) %v\t%v",
		t.ID, t.Instrument, t.Steps,
	)
}

type Steps []byte

func (steps Steps) String() string {
	var s string
	for i := 0; i < len(steps); i++ {
		if i%4 == 0 {
			s += "|"
		}

		if steps[i] > 0 {
			s += "x"
		} else {
			s += "-"
		}
	}
	s += "|"
	return s
}
