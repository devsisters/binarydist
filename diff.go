package binarydist

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
)

func matchlen(a, b []byte) (i int) {
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	return i
}

func search(I []int32, obuf, nbuf []byte, st, en int) (pos, n int) {
	if en-st < 2 {
		x := matchlen(obuf[I[st]:], nbuf)
		y := matchlen(obuf[I[en]:], nbuf)

		if x > y {
			return int(I[st]), x
		} else {
			return int(I[en]), y
		}
	}

	x := st + (en-st)/2
	if bytes.Compare(obuf[I[x]:], nbuf) < 0 {
		return search(I, obuf, nbuf, x, en)
	} else {
		return search(I, obuf, nbuf, st, x)
	}
	panic("unreached")
}

// Diff computes the difference between old and new, according to the bsdiff
// algorithm, and writes the result to patch.
func Diff(old, new io.Reader, patch io.Writer) error {
	obuf, err := ioutil.ReadAll(old)
	if err != nil {
		return err
	}

	nbuf, err := ioutil.ReadAll(new)
	if err != nil {
		return err
	}

	pbuf, err := diffBytes(obuf, nbuf)
	if err != nil {
		return err
	}

	hdr := header{Magic: magic, NewSize: int64(len(nbuf))}

	err = binary.Write(patch, signMagLittleEndian{}, hdr)
	if err != nil {
		return err
	}

	_, err = patch.Write(pbuf)
	return err
}

// Diff With bytes
func DiffBytes(obuf, nbuf []byte, patch io.Writer) error {
	pbuf, err := diffBytes(obuf, nbuf)
	if err != nil {
		return err
	}

	hdr := header{Magic: magic, NewSize: int64(len(nbuf))}

	err = binary.Write(patch, signMagLittleEndian{}, hdr)
	if err != nil {
		return err
	}

	_, err = patch.Write(pbuf)
	return err
}

func diffBytes(obuf, nbuf []byte) ([]byte, error) {
	var patch seekBuffer
	err := diff(obuf, nbuf, &patch)
	if err != nil {
		return nil, err
	}
	return patch.buf, nil
}

func diff(obuf, nbuf []byte, patch io.WriteSeeker) error {
	var lenf int
	I := make([]int32, len(obuf)+1)
	// be sure first elme is len(obuf)
	I[0] = int32(len(obuf))
	text_32(obuf, I[1:])

	// Compute the differences, writing ctrl as we go
	pfbz2, err := newBzip2Writer(patch)
	if err != nil {
		return err
	}
	defer pfbz2.Close()

	var scan, pos, length int
	var lastscan, lastpos, lastoffset int
	for scan < len(nbuf) {
		var oldscore int
		scan += length
		for scsc := scan; scan < len(nbuf); scan++ {
			pos, length = search(I, obuf, nbuf[scan:], 0, len(obuf))

			for ; scsc < scan+length; scsc++ {
				if scsc+lastoffset < len(obuf) &&
					obuf[scsc+lastoffset] == nbuf[scsc] {
					oldscore++
				}
			}

			if (length == oldscore && length != 0) || length > oldscore+8 {
				break
			}

			if scan+lastoffset < len(obuf) && obuf[scan+lastoffset] == nbuf[scan] {
				oldscore--
			}
		}

		if length != oldscore || scan == len(nbuf) {
			var s, Sf int
			lenf = 0
			for i := 0; lastscan+i < scan && lastpos+i < len(obuf); {
				if obuf[lastpos+i] == nbuf[lastscan+i] {
					s++
				}
				i++
				if s*2-i > Sf*2-lenf {
					Sf = s
					lenf = i
				}
			}

			lenb := 0
			if scan < len(nbuf) {
				var s, Sb int
				for i := 1; (scan >= lastscan+i) && (pos >= i); i++ {
					if obuf[pos-i] == nbuf[scan-i] {
						s++
					}
					if s*2-i > Sb*2-lenb {
						Sb = s
						lenb = i
					}
				}
			}

			if lastscan+lenf > scan-lenb {
				overlap := (lastscan + lenf) - (scan - lenb)
				s := 0
				Ss := 0
				lens := 0
				for i := 0; i < overlap; i++ {
					if nbuf[lastscan+lenf-overlap+i] == obuf[lastpos+lenf-overlap+i] {
						s++
					}
					if nbuf[scan-lenb+i] == obuf[pos-lenb+i] {
						s--
					}
					if s > Ss {
						Ss = s
						lens = i + 1
					}
				}

				lenf += lens - overlap
				lenb -= lens
			}

			/* Write control data */
			if err := binary.Write(pfbz2, signMagLittleEndian{}, int64(lenf)); err != nil {
				return err
			}

			val := (scan - lenb) - (lastscan + lenf)
			if err := binary.Write(pfbz2, signMagLittleEndian{}, int64(val)); err != nil {
				return err
			}

			val = (pos - lenb) - (lastpos + lenf)
			if err := binary.Write(pfbz2, signMagLittleEndian{}, int64(val)); err != nil {
				return err
			}

			/* Write diff data */
			buffer := bytes.NewBuffer(nil)
			for i := 0; i < lenf; i++ {
				buffer.WriteByte(nbuf[lastscan+i] - obuf[lastpos+i])
			}
			if err := binary.Write(pfbz2, signMagLittleEndian{}, buffer.Bytes()); err != nil {
				return err
			}

			/* Write extra data */
			buffer = bytes.NewBuffer(nil)
			extraN := (scan - lenb) - (lastscan + lenf)
			for i := 0; i < extraN; i++ {
				buffer.WriteByte(nbuf[lastscan+lenf+i])
			}
			if err := binary.Write(pfbz2, signMagLittleEndian{}, buffer.Bytes()); err != nil {
				return err
			}

			lastscan = scan - lenb
			lastpos = pos - lenb
			lastoffset = pos - scan
		}
	}

	return nil
}
