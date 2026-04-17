package rich_error

import "strconv"

type ByteStringer byte

func (i ByteStringer) String() string {
	return strconv.FormatUint(uint64(i), 16)
}

type StrStringer string

func (s StrStringer) String() string {
	return string(s)
}
