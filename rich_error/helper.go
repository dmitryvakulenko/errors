package rich_error

import "strconv"

type ByteStringer byte

func (i ByteStringer) String() string {
	return strconv.FormatUint(uint64(i), 16)
}
