package cryptoengine

import "io"

type randReader struct{}

func (randReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	b, err := RandomBytes(len(p))
	if err != nil {
		return 0, err
	}
	copy(p, b)
	return len(p), nil
}

// RandReader provides OpenSSL-backed randomness via io.Reader.
var RandReader io.Reader = randReader{}
