package flg

import "bytes"

func NewUnloader() *Unloader {
	return &Unloader{
		buf: &bytes.Buffer{},
	}
}

type Unloader struct {
	buf *bytes.Buffer
}
