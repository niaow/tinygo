package blocks

import (
	"strconv"
	"testing"
)

func TestSkipForwards(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in  mask
		out mask
		len int
	}{
		{
			in:  1 << 0,
			out: 0,
			len: 1,
		},
		{
			in:  1 << 2,
			out: 0,
			len: 3,
		},
		{
			in:  1 << (maskBits - 1),
			out: 0,
			len: int(maskBits),
		},
		{
			in:  0b11,
			out: 0b1,
			len: 1,
		},
		{
			in:  0b101,
			out: 0b10,
			len: 1,
		},
		{
			in:  0b10100,
			out: 0b10,
			len: 3,
		},
	}
	for _, c := range cases {
		c := c
		t.Run(strconv.FormatUint(uint64(c.in), 2), func(t *testing.T) {
			t.Parallel()

			out := c.in
			len := out.skipForwards()
			if out != c.out {
				t.Errorf("expected mask 0b%b but got 0b%b", c.out, out)
			}
			if int(len) != c.len {
				t.Errorf("expected len %d but got %d", c.len, len)
			}
		})
	}
}

func TestSkipBackwards(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in  mask
		out mask
		len int
	}{
		{
			in:  1 << (maskBits - 1),
			out: 0,
			len: 1,
		},
		{
			in:  1 << (maskBits - 3),
			out: 0,
			len: 3,
		},
		{
			in:  1 << 0,
			out: 0,
			len: int(maskBits),
		},
		// TODO: test multi-bit
	}
	for _, c := range cases {
		c := c
		t.Run(strconv.FormatUint(uint64(c.in), 2), func(t *testing.T) {
			t.Parallel()

			out := c.in
			len := out.skipBackwards()
			if out != c.out {
				t.Errorf("expected mask 0b%b but got 0b%b", c.out, out)
			}
			if int(len) != c.len {
				t.Errorf("expected len %d but got %d", c.len, len)
			}
		})
	}
}
