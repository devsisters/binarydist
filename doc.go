// Package binarydist implements binary diff and patch as described on
// http://www.daemonology.net/bsdiff/. It reads and writes files
// compatible with the tools there.
package binarydist

var magic = [16]byte{'E', 'N', 'D', 'S', 'L', 'E', 'Y', '/', 'B', 'S', 'D', 'I', 'F', 'F', '4', '3'}

type header struct {
	Magic   [16]byte
	NewSize int64
}
