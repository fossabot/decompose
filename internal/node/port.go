package node

import (
	"sort"
	"strconv"

	"github.com/s0rg/set"
)

type Port struct {
	Kind  string `json:"kind"`
	Value int    `json:"value"`
}

type Ports []Port

func (p *Port) Label() string {
	return strconv.Itoa(p.Value) + "/" + p.Kind
}

func (p *Port) ID() string {
	return p.Kind + strconv.Itoa(p.Value)
}

func (ps Ports) Dedup() (rv Ports) {
	state := make(map[string]set.Set[int])

	for i := 0; i < len(ps); i++ {
		p := &ps[i]

		s, ok := state[p.Kind]
		if !ok {
			s = make(set.Unordered[int])

			state[p.Kind] = s
		}

		s.Add(p.Value)
	}

	total := 0

	for _, s := range state {
		total += s.Len()
	}

	rv = make([]Port, 0, total)

	for k, s := range state {
		ports := set.ToSlice(s)

		if len(ports) > 1 {
			sort.Ints(ports)
		}

		for _, p := range ports {
			rv = append(rv, Port{
				Kind:  k,
				Value: p,
			})
		}
	}

	return rv
}
