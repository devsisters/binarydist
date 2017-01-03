package binarydist

/*
#cgo CFLAGS: -I/usr/include
#cgo LDFLAGS: -L/usr/lib -lbz2
#include <bzlib.h>
#include <stdlib.h>

bz_stream* new_bz_stream() {
  return calloc(1, sizeof(bz_stream));
}

void free_bz_stream(bz_stream* stream) {
  free(stream);
}

int bz_compress(bz_stream* stream, int action, char* in, unsigned avail_in, char* out, unsigned avail_out, unsigned* read_bytes, unsigned* written_bytes) {
  stream->next_in = in;
  stream->avail_in = avail_in;
  stream->next_out = out;
  stream->avail_out = avail_out;

  int errcode = BZ2_bzCompress(stream, action);

  stream->next_in = NULL;
  stream->next_out = NULL;

  *read_bytes = avail_in - stream->avail_in;
  *written_bytes = avail_out - stream->avail_out;

  return errcode;
}
*/
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"unsafe"
)

const (
	defaultBlockSize  = 9
	defaultVerbosity  = 0
	defaultWorkFactor = 30
)

type bzip2Writer struct {
	w        io.Writer
	bzstream *C.bz_stream
	outbuf   [65536]byte
}

func (bw *bzip2Writer) Write(b []byte) (int, error) {
	var total = 0

	for len(b) > 0 {
		var read_b, written_b C.uint

		errcode := C.bz_compress(bw.bzstream, C.BZ_RUN, (*C.char)(unsafe.Pointer(&b[0])), C.uint(len(b)), (*C.char)(unsafe.Pointer(&bw.outbuf[0])), C.uint(65530), &read_b, &written_b)
		if errcode != C.BZ_RUN_OK {
			return 0, fmt.Errorf("bzCompress returned non-ok %d.", errcode)
		}

		if int(read_b) == 0 {
			return 0, fmt.Errorf("bzCompress did not read anything")
		}
		total += int(read_b)
		b = b[read_b:]

		if _, err := bytes.NewReader(bw.outbuf[:int(written_b)]).WriteTo(bw.w); err != nil {
			return total, err
		}
	}

	return total, nil
}

func (bw *bzip2Writer) Close() (err error) {
	defer func() {
		errcode := C.BZ2_bzCompressEnd(bw.bzstream)
		if errcode != C.BZ_OK {
			if err != nil {
				err = fmt.Errorf("bzCompressEnd returned non-ok %d.\n%s", errcode, err.Error())
			} else {
				err = fmt.Errorf("bzCompressEnd returned non-ok %d.", errcode)
			}
		}
		C.free_bz_stream(bw.bzstream)
	}()

	var read_b, written_b C.uint

	for {
		errcode := C.bz_compress(bw.bzstream, C.BZ_FINISH, nil, 0, (*C.char)(unsafe.Pointer(&bw.outbuf[0])), C.uint(65530), &read_b, &written_b)

		if errcode != C.BZ_STREAM_END && errcode != C.BZ_FINISH_OK {
			err = fmt.Errorf("bzCompress returned non-ok %d.", errcode)
			return err
		}

		if _, werr := bytes.NewReader(bw.outbuf[:int(written_b)]).WriteTo(bw.w); err != nil {
			err = werr
			break
		}

		if errcode == C.BZ_STREAM_END {
			break
		}
	}

	return err
}

func newBzip2Writer(w io.Writer) (wc io.WriteCloser, err error) {
	var bw bzip2Writer
	bw.w = w
	bw.bzstream = C.new_bz_stream()

	errcode := int(C.BZ2_bzCompressInit(bw.bzstream, defaultBlockSize, defaultVerbosity, defaultWorkFactor))
	if errcode != C.BZ_OK {
		return nil, fmt.Errorf("bzCompressInit returned non-ok %d.", errcode)
	}
	return &bw, nil
}
