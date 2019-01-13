package ankit

import (
	"encoding/csv"
	"io"
)

// Reader is the interface that wraps the basic Read method.
type Reader interface {
	Read() (fields []string, err error)
}

// Copy copies from src to dst until either EOF is reached on src or an error occurs.
func Copy(dst io.Writer, src Reader) error {
	cw := csv.NewWriter(dst)

	for {
		fields, err := src.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if err := cw.Write(fields); err != nil {
			return err
		}
	}
	cw.Flush()

	return cw.Error()
}

// Note consists of fields.
type Note interface {
	Fields() []string
}

type oneNoteReader struct {
	Note
	readed bool
}

// Read reads fields only once.
func (r *oneNoteReader) Read() ([]string, error) {
	if r.readed {
		return nil, io.EOF
	}

	r.readed = true
	return r.Fields(), nil
}

// OneNoteReader returns a Reader with only one Note.
func OneNoteReader(n Note) Reader {
	return &oneNoteReader{Note: n}
}
