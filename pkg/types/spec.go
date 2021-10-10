package types

import (
    "strings"
)

// A SpecTuple is a listing of the host and target arch.
type SpecTuple struct {
	Host   string
	Target string
}

func (st SpecTuple) String() string {
	return st.Host + ":" + st.Target
}

func (st SpecTuple) Native() bool {
	return st.Host == st.Target
}

// SpecTupleFromString returns a spec tuple from its string
// representation.
func SpecTupleFromString(s string) SpecTuple {
	p := strings.SplitN(s, ":", 2)
	return SpecTuple{p[0], p[1]}
}
