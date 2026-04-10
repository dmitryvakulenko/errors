package rich_error

import "strconv"

type IntStringer int

func (i IntStringer) String() string {
	return strconv.Itoa(int(i))
}
