package parse

import (
	"flag"
	"go/scanner"
	"go/token"
	"os"
	"reflect"
	"slices"
	"testing"
)

const exampleSrc = `package main

import "fmt"

func main() {
	fmt.Println("Hello world!")
}
`

func TestTokenize(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		name   string
		src    string
		result tokenizeResult
		err    error
	}{
		{
			name: "Empty",
		},
		{
			name: "SpaceOnly",
			src:  " \t\r\n",
		},
		{
			name: "Identifier",
			src:  "abc123_æ",
			result: tokenizeResult{
				litOrIdent: []string{"abc123_æ"},
				tokens: []Token{
					{ID: IdentIdxBase, Offset: 9},
				},
			},
		},
		{
			name: "Number",
			src:  "0x1.0p-16",
			result: tokenizeResult{
				litOrIdent: []string{"0x1.0p-16"},
				tokens: []Token{
					{ID: IdentIdxBase, Offset: 9},
				},
			},
		},
		{
			name: "LeadingDecimal",
			src:  "0.125",
			result: tokenizeResult{
				litOrIdent: []string{"0.125"},
				tokens: []Token{
					{ID: IdentIdxBase, Offset: 5},
				},
			},
		},
		// TODO: test operators
		{
			name: "ExampleSrc",
			src:  exampleSrc,
			result: tokenizeResult{
				litOrIdent: []string{
					`main`,
					`"fmt"`,
					`fmt`,
					`Println`,
					`"Hello world!"`,
				},
				tokens: []Token{
					{ID: TokenPackage, Offset: 73},
					{ID: IdentIdxBase, Offset: 65},
					{ID: TokenSemicolon, Offset: 61},
					{ID: TokenImport, Offset: 59},
					{ID: IdentIdxBase + 1, Offset: 52},
					{ID: TokenSemicolon, Offset: 47},
					{ID: TokenFunc, Offset: 45},
					{ID: IdentIdxBase, Offset: 40},
					{ID: TokenOpenParen, Offset: 36},
					{ID: TokenCloseParen, Offset: 35},
					{ID: TokenOpenBrace, Offset: 33},
					{ID: IdentIdxBase + 2, Offset: 30},
					{ID: TokenDot, Offset: 27},
					{ID: IdentIdxBase + 3, Offset: 26},
					{ID: TokenOpenParen, Offset: 19},
					{ID: IdentIdxBase + 4, Offset: 18},
					{ID: TokenCloseParen, Offset: 4},
					{ID: TokenSemicolon, Offset: 3},
					{ID: TokenCloseBrace, Offset: 2},
					{ID: TokenSemicolon, Offset: 1},
				},
			},
		},
	} {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			res, err := tokenize([]byte(c.src))

			// Compare the error.
			if err != c.err {
				t.Errorf("expected error %q but found %q", c.err, err)
			}

			// Compare the litOrIdent lists.
			if expected, got := c.result.litOrIdent, res.litOrIdent; !slices.Equal(expected, got) {
				t.Error("expected lit/idents:")
				for _, v := range expected {
					t.Logf("\t%q", v)
				}
				t.Error("got lit/idents:")
				for _, v := range got {
					t.Logf("\t%q", v)
				}
			}

			// Compare the token lists.
			if expected, got := c.result.tokens, res.tokens; !slices.Equal(expected, got) {
				t.Error("expected tokens:")
				for _, v := range expected {
					t.Log("\t", v.StringWithLookup(c.result.litOrIdent))
				}
				t.Error("got tokens:")
				for _, v := range got {
					t.Log("\t", v.StringWithLookup(res.litOrIdent))
				}
			}

			// Compare the comment lists.
			if expected, got := c.result.comments, res.comments; !reflect.DeepEqual(expected, got) {
				t.Error("expected comments:")
				for _, v := range expected {
					t.Log("\t", v)
				}
				t.Error("got comments:")
				for _, v := range got {
					t.Log("\t", v)
				}
			}
		})
	}
}

func TestTokenizeSelf(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile("tokenize.go")
	if err != nil {
		t.Fatal("failed to read source:", err)
	}

	_, err = tokenize(src)
	if err != nil {
		t.Error("tokenize failed:", err)
	}
}

func BenchmarkTokenize(b *testing.B) {
	path := *tokenizeBenchSrc
	if path == "" {
		b.Skip()
	}

	src, err := os.ReadFile(path)
	if err != nil {
		b.Fatal("failed to read source:", err)
	}

	b.SetBytes(int64(len(src)))

	b.ResetTimer()

	for i := b.N; i > 0; i-- {
		_, err := tokenize(src)
		if err != nil {
			b.Fatal("failed to tokenize:", err)
		}
	}
}

func BenchmarkLines(b *testing.B) {
	path := *tokenizeBenchSrc
	if path == "" {
		b.Skip()
	}

	src, err := os.ReadFile(path)
	if err != nil {
		b.Fatal("failed to read source:", err)
	}

	b.SetBytes(int64(len(src)))

	b.ResetTimer()

	for i := b.N; i > 0; i-- {
		_ = Lines(src)
	}
}

func BenchmarkGoScanner(b *testing.B) {
	path := *tokenizeBenchSrc
	if path == "" {
		b.Skip()
	}

	src, err := os.ReadFile(path)
	if err != nil {
		b.Fatal("failed to read source:", err)
	}

	b.SetBytes(int64(len(src)))

	b.ResetTimer()

	for i := b.N; i > 0; i-- {
		fset := token.NewFileSet()
		var s scanner.Scanner
		s.Init(fset.AddFile(path, fset.Base(), len(src)), src, nil, scanner.ScanComments)
		for {
			_, tok, _ := s.Scan()
			if tok == token.EOF {
				break
			}
		}
	}
}

var tokenizeBenchSrc = flag.String("tokenizeSrc", "", "tokenize benchmark source path")

// hash for 2-byte operators: (b * 1484) >> 11
// hash for 3-byte operators: (byte(b) * 17) >> 6

// The following functions were used to find the hash coefficients.
/*
func TestFindOperHash2(t *testing.T) {
	var k uint16
	for {
		k++
		var mask uint32
		for _, b := range twoByteOps {
			mask |= 1 << ((k * uint16(b)) >> 11)
		}
		if bits.OnesCount32(mask) == len(twoByteOps) {
			t.Error(uint(k))
			break
		}
		if k == math.MaxUint16 {
			break
		}
	}
}

func TestFindOperHash3(t *testing.T) {
	var k byte
	for {
		k++
		var mask uint8
		for _, b := range threeByteOps {
			mask |= 1 << ((k * byte(b)) >> 6)
		}
		if bits.OnesCount8(mask) == len(threeByteOps) {
			t.Error(uint(k))
			break
		}
		if k == math.MaxUint8 {
			break
		}
	}
}
*/
