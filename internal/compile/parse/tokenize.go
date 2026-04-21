package parse

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strings"
	"unsafe"
)

const (
	// Statement-terminating operators
	TokenIncrement = iota
	TokenDecrement

	// Brackets
	// NOTE: All semicolon-inserting operators must be <= TokenCloseBrace.
	// NOTE (cont): This is used by the semicolon insertion check.
	TokenCloseParen
	TokenCloseBracket
	TokenCloseBrace
	TokenOpenParen
	TokenOpenBracket
	TokenOpenBrace

	// Statement operators
	TokenAssign
	TokenDefine
	TokenSend
	TokenAddAssign
	TokenSubtractAssign
	TokenMultiplyAssign
	TokenDivideAssign
	TokenModuloAssign
	TokenBitwiseAndAssign
	TokenBitwiseOrAssign
	TokenBitwiseXOrAssign
	TokenBitwiseAndNotAssign
	TokenShiftLeftAssign
	TokenShiftRightAssign

	// Misc
	TokenUnpack
	TokenComma
	TokenDot
	TokenSemicolon
	TokenColon
	TokenNot
	TokenUnderlying

	// Operators precedence 5
	TokenMultiply
	TokenDivide
	TokenModulo
	TokenShiftLeft
	TokenShiftRight
	TokenBitwiseAnd
	TokenBitwiseAndNot
	// Operators precedence 4
	TokenAdd
	TokenSubtract
	TokenBitwiseOr
	TokenBitwiseXOr
	// Operators precedence 3
	TokenEqual
	TokenNotEqual
	TokenLess
	TokenLessOrEqual
	TokenGreater
	TokenGreaterOrEqual
	// Operators precedence 2
	TokenLogicalAnd
	// Operators precedence 1
	TokenLogicalOr

	// Keywords
	TokenCase
	TokenChan
	TokenConst
	TokenDefault
	TokenDefer
	TokenElse
	TokenFor
	TokenFunc
	TokenGo
	TokenGoto
	TokenIf
	TokenImport
	TokenInterface
	TokenMap
	TokenPackage
	TokenRange
	TokenSelect
	TokenStruct
	TokenSwitch
	TokenType
	TokenVar
	TokenBreak
	TokenContinue
	TokenFallthrough
	TokenReturn
)

const firstTokenNoSemicolon = TokenCloseBrace + 1
const semicolonLitOrIdentStart = TokenBreak

const litOrIdentBase = TokenCase
const maxKeyword = TokenReturn
const IdentIdxBase = maxKeyword + 1

type Token struct {
	// ID identifies the contents of the token.
	// It may be an operator/keyword ID (Token*).
	// Otherwise, it is a literal or identifier at litOrIdent[ID - IdentIdxBase].
	ID uint32

	// Offset is the position of the token start relative to the end of the file.
	Offset uint32
}

func (t Token) String() string {
	return t.StringWithLookup(nil)
}

func (t Token) StringWithLookup(idents []string) string {
	if t.ID <= maxKeyword {
		return fmt.Sprintf("%s @-%d", tokenStrings[t.ID], t.Offset)
	} else if identIdx := t.ID - IdentIdxBase; uint(identIdx) < uint(len(idents)) {
		return fmt.Sprintf("%q (idx=%d) @-%d", idents[identIdx], identIdx, t.Offset)
	} else {
		return fmt.Sprintf("unknown ident/lit %d @-%d", identIdx, t.Offset)
	}
}

var tokenStrings = [...]string{
	TokenIncrement: "++",
	TokenDecrement: "--",

	TokenCloseParen:   ")",
	TokenCloseBracket: "]",
	TokenCloseBrace:   "}",
	TokenOpenParen:    "(",
	TokenOpenBracket:  "[",
	TokenOpenBrace:    "{",

	TokenAssign:              "=",
	TokenDefine:              ":=",
	TokenSend:                "<-",
	TokenAddAssign:           "+=",
	TokenSubtractAssign:      "-=",
	TokenMultiplyAssign:      "*=",
	TokenDivideAssign:        "/=",
	TokenModuloAssign:        "%=",
	TokenBitwiseAndAssign:    "&=",
	TokenBitwiseOrAssign:     "|=",
	TokenBitwiseXOrAssign:    "^=",
	TokenBitwiseAndNotAssign: "&^=",
	TokenShiftLeftAssign:     "<<=",
	TokenShiftRightAssign:    ">>=",

	TokenUnpack:     "...",
	TokenComma:      ",",
	TokenDot:        ".",
	TokenSemicolon:  ";",
	TokenColon:      ":",
	TokenNot:        "!",
	TokenUnderlying: "~",

	TokenMultiply:       "*",
	TokenDivide:         "/",
	TokenModulo:         "%",
	TokenShiftLeft:      "<<",
	TokenShiftRight:     ">>",
	TokenBitwiseAnd:     "&",
	TokenBitwiseAndNot:  "&^",
	TokenAdd:            "+",
	TokenSubtract:       "-",
	TokenBitwiseOr:      "|",
	TokenBitwiseXOr:     "^",
	TokenEqual:          "==",
	TokenNotEqual:       "!=",
	TokenLess:           "<",
	TokenLessOrEqual:    "<=",
	TokenGreater:        ">",
	TokenGreaterOrEqual: ">=",
	TokenLogicalAnd:     "&&",
	TokenLogicalOr:      "||",

	TokenCase:        "case",
	TokenChan:        "chan",
	TokenConst:       "const",
	TokenDefault:     "default",
	TokenDefer:       "defer",
	TokenElse:        "else",
	TokenFor:         "for",
	TokenFunc:        "func",
	TokenGo:          "go",
	TokenGoto:        "goto",
	TokenIf:          "if",
	TokenImport:      "import",
	TokenInterface:   "interface",
	TokenMap:         "map",
	TokenPackage:     "package",
	TokenRange:       "range",
	TokenSelect:      "select",
	TokenStruct:      "struct",
	TokenSwitch:      "switch",
	TokenType:        "type",
	TokenVar:         "var",
	TokenBreak:       "break",
	TokenContinue:    "continue",
	TokenFallthrough: "fallthrough",
	TokenReturn:      "return",
}

var itabInit = func() (dst [354]byte) {
	j := uint(0)
	for i := uint(litOrIdentBase); i <= maxKeyword; i++ {
		str := tokenStrings[i]
		copy(dst[j:][:len(str)], str)
		j += uint(len(str))
		binary.LittleEndian.PutUint32(dst[j:], uint32(i))
		j += 9
	}
	if j != uint(len(dst)) {
		panic("length mismatch")
	}
	return
}()

type comment struct {
	// bind is the index in the token list where the comment is located.
	// This can be used to bind comments to tokens (e.g. to match directives to functions).
	bind uint32
	// offset is the offset of the comment start relative to the end of the file.
	offset uint32
	// len is the length of the comment in bytes.
	len uint32
}

func (c comment) String() string {
	return fmt.Sprintf("@%d : -%d len=%d", c.bind, c.offset, c.len)
}

type tokenizeResult struct {
	// litOrIdent is a list of literals/identifiers in order of first appearance.
	// An entry at index i is mapped to token ID identIdxBase + i.
	litOrIdent []string
	// tokens is a list of tokens in the source.
	tokens []Token
	// comments is a list of comments in the source.
	comments []comment
}

type OffsetError struct {
	Err    error
	Offset uint32
}

func (err OffsetError) Error() string {
	return fmt.Sprintf("%v @-%d", err.Err, err.Offset)
}

func (err OffsetError) Unwrap() error {
	return err.Err
}

var (
	ErrSourceTooBig         = errors.New("source is too big (4 GiB+)")
	ErrUnexpectedSourceByte = errors.New("unexpected source byte")
	ErrMissingCloseQuote    = errors.New("missing close quote")
	ErrCommentNotTerminated = errors.New("comment not terminated")
)

// debugDupCheck checks for duplicates in the resulting litOrIdent list.
const debugDupCheck = false

// debugCollisionDist prints metrics about the IdentTable's internal state.
const debugCollisionDist = false

// debugTokenRatio prints the ratio of bytes to tokens.
const debugTokenRatio = false

// tokenize creates a list of tokens in the provided source code.
func tokenize(src []byte) (tokenizeResult, error) {
	if len(src) >= 3 && string(src[:3]) == "\xEE\xBB\xBF" {
		// This is a UTF-8 encoded byte-order mark.
		// Some software will insert this to mark UTF-8 encoded text.
		// Ignore it.
		src = src[3:]
	}

	// We use 32-bit offsets.
	// The length of the source must fit into 32 bits.
	if uint(len(src)) > math.MaxUint32 {
		return tokenizeResult{}, ErrSourceTooBig
	}

	oldLen := len(src)

	// The average bytes-to-tokens ratio is generally a bit above 4.
	// Pre-allocate an estimated buffer to minimize copying.
	// This nearly doubles tokenization speed for some workloads.
	// TODO: the token slice was a bad idea?
	// This only works because we overestimate the size (sometimes by significant amount).
	tokens := make([]Token, 0, len(src)/4)

	// Initialize the litOrIdent table.
	litOrIdent := identTable{
		// The slice has cap == len.
		// The next append will copy it.
		data: itabInit[:],
		last: maxKeyword,
	}

	var comments []comment
	src = src[:0:len(src)]
	var token uint32 = TokenSemicolon
	var err error
end:
	for {
		// Consume whitespace.
		src = scanWhitespace(src, token)
		src = src[len(src):]
		if cap(src) <= 0 {
			err = nil
			break
		}

		// Identify the next token.
		src = src[:1]
		b := src[0]
		start := tokenStarts[b]
	operator:
		switch {
		case start < tokenStartIdent:
			// Parse as an operator.
			token = uint32(start)
			if cap(src) >= 2 {
				// Load the first 2 bytes into a uint16.
				b2 := uint16(b) | uint16(src[:cap(src)][1])<<8
				if cap(src) >= 3 {
					// Load a third byte to make a 24-bit integer.
					b3 := uint32(b2) | uint32(src[:cap(src)][2])<<16

					// Look up the 3-byte sequence in threeByteOpTbl.
					if op3Ent := threeByteOpTbl[(b*17)>>6]; (op3Ent^b3)&0xFF_FF_FF == 0 {
						// Extract the token ID from the entry.
						token = op3Ent >> 24
						src = src[:3]
						break
					}
				}

				// Look up the byte pair in twoByteOpTbl.
				if op2Ent := twoByteOpTbl[(b2*1484)>>11]; uint16(op2Ent) == b2 {
					// Extract the token ID from the entry.
					token = op2Ent >> 16
					src = src[:2]
					break
				}
			}

		default:
			// This is probbably a keyword/literal/identifier.
			// Break from the "operator" label if it is not.
			// Use an if here instead of the main switch to ensure that tokenStartIdent is checked first.
			if start == tokenStartIdent {
				// Scan as a keyword or identifier.
				src = scanIdentifier(src)
			} else {
				switch start {
				case tokenStartDot:
					// A '.' can start a few different tokens:
					//  - the . operator
					//  - the ... operator
					//  - a floating point literal (e.g. ".125")
					// Initially assume it is the . operator.
					token = TokenDot
					if len(src) < 2 {
						// There are no more bytes.
						// Interpret the . alone.
						break operator
					}

					// Match the ... operator.
					if len(src) >= 3 && (uint16(src[:cap(src)][1])|uint16(src[:cap(src)][2])<<8) == '.'|'.'<<8 {
						token = TokenUnpack
						break operator
					}

					// Check if the next byte is a digit.
					next := src[:cap(src)][1]
					if !('0' <= next && next <= '9') {
						// The next byte is not a digit.
						// Interpret the . alone.
						break operator
					}

					// Scan as a number.
					// Treat the leading .X as already parsed.
					src = src[:2]
					b = next
					fallthrough
				case tokenStartNumber:
					// Scan the number.
					src = scanNumber(src, b)

				case tokenStartSlash:
					// A '/' can start a few different elements:
					//  - the / operator
					//  - the /= operator
					//  - a line comment //
					//  - a general comment /*
				notComment:
					switch {
					case cap(src) >= 2:
						switch src[:cap(src)][1] {
						case '/':
							// Scan the line comment.
							src = scanLineComment(src[:2])

						case '*':
							// Scan the general comment.
							src, err = scanGeneralComment(src[:2])
							if err != nil {
								break end
							}

						case '=':
							// Scan as a /= operator.
							token = TokenDivideAssign
							src = src[:2]
							break operator

						default:
							break notComment
						}

						// Append the comment.
						comments = append(comments, comment{
							bind:   uint32(len(tokens)),
							offset: uint32(cap(src)),
							len:    uint32(len(src)),
						})
						continue
					}

					// Scan as a / operator.
					token = TokenDivide
					break operator

				case tokenStartStrOrRune:
					// Scan the string or rune.
					src, err = scanStringOrRune(src, b)
					if err != nil {
						break end
					}

				case tokenStartRawStr:
					// Scan the raw string.
					src, err = scanRawString(src)
					if err != nil {
						break end
					}

				default:
					// The byte is not valid in Go source.
					err = ErrUnexpectedSourceByte
					break end
				}
			}

			// Find or assign an ID for the ident/keyword/literal.
			token = litOrIdent.Get(src)
		}

		// Append the token.
		tokens = append(tokens, Token{
			ID:     token,
			Offset: uint32(cap(src)),
		})
	}
	if err != nil {
		// Attach the offset to the error.
		err = OffsetError{
			Offset: uint32(cap(src)),
			Err:    err,
		}
	}

	if debugCollisionDist {
		dist, load, avgCmp, expAvgCmp := litOrIdent.collisionDist()
		fmt.Printf("hash table state:\n\tcollision dist: %v\n\tload: %.2f%%\n\taverage comparisons per lookup: %.2f\n\ttheoretical average comparisons per lookup: %.2f\n", dist, 100*load, avgCmp, expAvgCmp)
	}

	// Convert the litOrIdent map to a list.
	n := uint(litOrIdent.last) - maxKeyword
	litOrIdentList := make([]string, n)
	catStr := unsafeCastStr(litOrIdent.data[len(itabInit):])
	for len(catStr) >= 9 {
		catStr = catStr[:len(catStr)-9]
		zeroTerm := uint(strings.LastIndexByte(catStr, 0)) + 1
		str := catStr[zeroTerm:]
		catStr = catStr[:zeroTerm]
		litOrIdentList[len(litOrIdentList)-1] = str
		litOrIdentList = litOrIdentList[:len(litOrIdentList)-1]
	}
	if len(litOrIdentList) != 0 {
		panic("incomplete lit or ident list")
	}

	if debugDupCheck {
		dups := make(map[string]int, n)
		for i, str := range litOrIdentList[:n] {
			i += maxKeyword
			if old, ok := dups[str]; ok {
				fmt.Println("duplicated:", str, "id:", old, ":", i)
			}
			dups[str] = i
		}
		if uint(len(dups)) != n {
			panic("found duplicates")
		}
	}

	if debugTokenRatio {
		fmt.Printf("tokens: %d, bytes: %d, bytes/token: %.2f\n", len(tokens), oldLen, float64(oldLen)/float64(len(tokens)))
	}

	return tokenizeResult{
		litOrIdent: litOrIdentList[:n],
		tokens:     tokens,
		comments:   comments,
	}, err
}

// tokenStarts indicates how to parse a token based on its starting byte.
// The table is keyed by the raw starting byte value.
// For bytes which are always the start of an operator, the value is the Token ID of the operator.
// For other bytes, the value is one of the tokenStart* constants.
// For bytes invalid in Go source, the value is 0xFF.
var tokenStarts = func() (tbl [1 << 8]uint8) {
	for i := range tbl {
		tbl[i] = 0xFF
	}

	tbl['+'] = TokenAdd
	tbl['-'] = TokenSubtract
	tbl['*'] = TokenMultiply
	tbl['%'] = TokenModulo
	tbl['&'] = TokenBitwiseAnd
	tbl['|'] = TokenBitwiseOr
	tbl['^'] = TokenBitwiseXOr
	tbl['<'] = TokenLess
	tbl['>'] = TokenGreater
	tbl['='] = TokenAssign
	tbl['!'] = TokenNot
	tbl['~'] = TokenUnderlying
	tbl[':'] = TokenColon
	tbl[','] = TokenComma
	tbl['('] = TokenOpenParen
	tbl[')'] = TokenCloseParen
	tbl['['] = TokenOpenBracket
	tbl[']'] = TokenCloseBracket
	tbl['{'] = TokenOpenBrace
	tbl['}'] = TokenCloseBrace
	tbl[';'] = TokenSemicolon
	tbl['\n'] = TokenSemicolon

	for b := 'a'; b <= 'z'; b++ {
		tbl[b] = tokenStartIdent
	}
	for b := 'A'; b <= 'Z'; b++ {
		tbl[b] = tokenStartIdent
	}
	tbl['_'] = tokenStartIdent
	for b := 0x80; b <= 0xFF; b++ {
		tbl[b] = tokenStartIdent
	}

	for b := '0'; b <= '9'; b++ {
		tbl[b] = tokenStartNumber
	}

	tbl['.'] = tokenStartDot
	tbl['/'] = tokenStartSlash
	tbl['\''] = tokenStartStrOrRune
	tbl['"'] = tokenStartStrOrRune
	tbl['`'] = tokenStartRawStr
	return
}()

const (
	tokenStartIdent = litOrIdentBase + iota
	tokenStartDot
	tokenStartSlash
	tokenStartNumber
	tokenStartStrOrRune
	tokenStartRawStr
)

var twoByteOpTbl = func() (tbl [32]uint32) {
	for _, v := range twoByteOps {
		idx := (uint16(v) * 1484) >> 11
		if tbl[idx] != 0 {
			panic("hash collision")
		}
		tbl[idx] = v
	}
	return
}()

var twoByteOps = [...]uint32{
	'<' | '<'<<8 | TokenShiftLeft<<16,
	'>' | '>'<<8 | TokenShiftRight<<16,
	'&' | '^'<<8 | TokenBitwiseAndNot<<16,
	'+' | '='<<8 | TokenAddAssign<<16,
	'-' | '='<<8 | TokenSubtractAssign<<16,
	'*' | '='<<8 | TokenMultiplyAssign<<16,
	'/' | '='<<8 | TokenDivideAssign<<16,
	'%' | '='<<8 | TokenModuloAssign<<16,
	'&' | '='<<8 | TokenBitwiseAndAssign<<16,
	'|' | '='<<8 | TokenBitwiseOrAssign<<16,
	'^' | '='<<8 | TokenBitwiseXOrAssign<<16,
	'&' | '&'<<8 | TokenLogicalAnd<<16,
	'|' | '|'<<8 | TokenLogicalOr<<16,
	'<' | '-'<<8 | TokenSend<<16,
	'+' | '+'<<8 | TokenIncrement<<16,
	'-' | '-'<<8 | TokenDecrement<<16,
	'=' | '='<<8 | TokenEqual<<16,
	'!' | '='<<8 | TokenNotEqual<<16,
	'<' | '='<<8 | TokenLessOrEqual<<16,
	'>' | '='<<8 | TokenGreaterOrEqual<<16,
	':' | '='<<8 | TokenDefine<<16,
}

var threeByteOpTbl = func() (tbl [4]uint32) {
	for _, v := range threeByteOps {
		idx := (byte(v) * 17) >> 6
		if tbl[idx] != 0 {
			panic("hash collision")
		}
		tbl[idx] = v
	}
	return
}()

var threeByteOps = [...]uint32{
	'<' | '<'<<8 | '='<<16 | TokenShiftLeftAssign<<24,
	'>' | '>'<<8 | '='<<16 | TokenShiftRightAssign<<24,
	'&' | '^'<<8 | '='<<16 | TokenBitwiseAndNotAssign<<24,
}

// scanWhitespace finds the end of the whitespace in src[len(src):cap(src)].
// It increments len(src) until the next byte is not valid whitespace.
// Use the noNewline flag to stop at a newline.
func scanWhitespace(src []byte, prevToken uint32) []byte {
	var spaceMask uint64 = 1<<' ' | 1<<'\t' | 1<<'\r'
	if uint(prevToken)-firstTokenNoSemicolon <= semicolonLitOrIdentStart-firstTokenNoSemicolon {
		// Skip newlines too.
		spaceMask |= 1 << '\n'
	}
	for ; len(src) < cap(src); src = src[:len(src)+1] {
		b := src[:cap(src)][len(src)]
		if b > ' ' || (spaceMask>>b)&1 == 0 {
			break
		}
	}

	return src
}

// scanNumber finds the end of the number in src[len(src):cap(src)].
// It increments len(src) until the next byte is not valid for a number.
func scanNumber(src []byte, b byte) []byte {
	for ; len(src) < cap(src); src = src[:len(src)+1] {
		// Fetch the next byte.
		prev := b
		b = src[:cap(src)][len(src)]

		if (numByteBitmap[b/bits.UintSize]>>(b%bits.UintSize))&1 != 0 {
			// The byte is marked as valid in the bitmap.
			continue
		}

		switch b {
		case '+', '-':
			switch prev {
			case 'e', 'E', 'p', 'P':
				// This is an exponent sign.
				continue
			}
		}

		break
	}
	return src
}

// numByteBitmap is identByteBitmap + '.'.
var numByteBitmap bm8 = func() (b bm8) {
	b = identByteBitmap
	b['.'/bits.UintSize] |= 1 << ('.' % bits.UintSize)
	return
}()

// scanIdentifier finds the end of the identifier in src[:cap(src)].
func scanIdentifier(src []byte) []byte {
	if cap(src) >= 8 {
		// Fast path: match up to 8 bytes as letters.
		raw := binary.LittleEndian.Uint64(src[:8])
		lower := raw | (0x20 * byteMask64)
		cmp := raw
		cmp |= lower - ('a' * byteMask64)
		cmp |= lower + ((0x7F - 'z') * byteMask64)
		cmp &= byteMask64 << 7
		src = src[:uint(bits.TrailingZeros64(cmp))/8]
	}
	for ; len(src) < cap(src); src = src[:len(src)+1] {
		// Fetch the next byte.
		b := src[:cap(src)][len(src)]

		if (identByteBitmap[b/bits.UintSize]>>(b%bits.UintSize))&1 != 0 {
			// The byte is marked as valid in the bitmap.
			continue
		}

		break
	}
	return src
}

const byteMask64 = ^uint64(0) / 0xFF

// identByteBitmap is a bitmap of bytes that are valid in an identifier.
var identByteBitmap bm8 = func() (b bm8) {
	b['0'/bits.UintSize] |= (2<<'9' - 1<<'0') >> ('0' &^ (bits.UintSize - 1))
	b['A'/bits.UintSize] |= ((2<<'Z' - 1<<'A') | 1<<'_') >> ('A' &^ (bits.UintSize - 1))
	b['a'/bits.UintSize] |= (2<<'z' - 1<<'a') >> ('a' &^ (bits.UintSize - 1))
	for i := uint(0x80 / bits.UintSize); i < uint(len(b)); i++ {
		b[i] = ^uint(0)
	}
	return
}()

// bm8 is a bitmap that is 2^8 bits wide.
type bm8 = [(1 << 8) / bits.UintSize]uint

// scanStringOrRune finds the end of a string or rune in src[len(src):cap(src)].
// It increments len(src) until it adds a trailing closeQuote.
// The caller must slice src to include the leading quote so that the string does not end immediately.
// It returns ErrMissingCloseQuote if the source does not include a closing quote.
func scanStringOrRune(src []byte, closeQuote byte) ([]byte, error) {
	for {
		// Find the next quote.
		at := bytes.IndexByte(src[len(src):cap(src)], closeQuote)
		if at < 0 {
			// The close quote is missing.
			return src[:cap(src)], ErrMissingCloseQuote
		}
		at += len(src)

		// Check if the quote is escaped.
		_ = &src[:cap(src)][at]
		src = src[:at+1]
		if countTrailing(src[:at], '\\')&1 == 0 {
			// End the literal with the close quote.
			return src, nil
		}
	}
}

// countTrailing counts the trailing instances of b in the string.
func countTrailing(str []byte, b byte) int {
	oldLen := len(str)

	// Slice off trailing instances of b.
	for len(str) > 0 && str[len(str)-1] == b {
		str = str[:len(str)-1]
	}

	// Count the removed bytes.
	return oldLen - len(str)
}

// scanRawString finds the end of a raw string ('`') in src[len(src):cap(src)].
// It increments len(src) until it adds a trailing '`'.
// The caller must slice src to include the leading '`' so that the string does not end immediately.
// It returns ErrMissingCloseQuote if the source does not include a closing '`'.
func scanRawString(src []byte) ([]byte, error) {
	// Find the close quote.
	at := bytes.IndexByte(src[len(src):cap(src)], '`')
	if at < 0 {
		// The close quote is missing.
		return src[:cap(src)], ErrMissingCloseQuote
	}

	// End the literal with the close quote.
	return src[:at+len(src)+1], nil
}

// scanLineComment finds the end of the line comment ("\n" or EOF) in src[:cap(src)].
// It increments len(src) until len(src) == cap(src) or src[:cap(src)][len(src)] == '\n'.
func scanLineComment(src []byte) []byte {
	return src[:min(uint(cap(src)), uint(bytes.IndexByte(src[:cap(src)], '\n')))]
}

// scanGeneralComment finds the end of a general comment ("*/") in src[len(src):cap(src)].
// It increments len(src) until it adds a trailing */.
// The caller must slice src to include the leading /* so that /*/ is not matched as a complete general comment.
// It returns ErrCommentNotTerminated if the source does not include a closing */.
func scanGeneralComment(src []byte) ([]byte, error) {
	// Find the ending */.
	at := bytes.Index(src[len(src):cap(src)], generalCommentEnd[:])
	if at < 0 {
		return src[:cap(src)], ErrCommentNotTerminated
	}

	// End the literal with the */.
	return src[:at+len(src)+2], nil
}

var generalCommentEnd = [2]byte{'*', '/'}

func unsafeCastStr(str []byte) string {
	return unsafe.String(unsafe.SliceData(str), len(str))
}
