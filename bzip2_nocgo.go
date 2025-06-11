//go:build !cgo
// +build !cgo

package binarydist

import (
	"io"

	"github.com/dsnet/compress/bzip2"
)

func newBzip2Writer(w io.Writer) (wc io.WriteCloser, err error) {
	return bzip2.NewWriter(w, nil)
}
