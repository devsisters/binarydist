# binarydist

Package binarydist implements binary diff and patch as described on
<http://www.daemonology.net/bsdiff/>. It reads and writes files
compatible with the tools there.

This is forked from kr/binarydist, with some local modifications.
 - This package uses [mendsley's modified algorithm](https://github.com/mendsley/bsdiff) instead of original one. Not compatible with original bsdiff/bspatch.
 - This package uses CGO binding to libbz2 instead of calling "bzip2" binary externally. (increased performance on high workload)
 

Documentation at <http://go.pkgdoc.org/github.com/kr/binarydist>.
