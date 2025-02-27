//  Copyright (c) 2016 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package querystring

import (
	"bufio"
	"io"
	"strings"
	"unicode"

	"github.com/sirupsen/logrus"
)

const reservedChars = "+-=&|><!(){}[]^\"~*?:\\/ "

func unescape(escaped string) string {
	// see if this character can be escaped
	if strings.ContainsAny(escaped, reservedChars) {
		return escaped
	}
	// otherwise return it with the \ intact
	return "\\" + escaped
}

type queryStringLex struct {
	in            *bufio.Reader
	buf           string
	currState     lexState
	currConsumed  bool
	inEscape      bool
	nextToken     *yySymType
	nextTokenType int
	seenDot       bool
	nextRune      rune
	nextRuneSize  int
	atEOF         bool
}

func (l *queryStringLex) reset() {
	l.buf = ""
	l.inEscape = false
	l.seenDot = false
}

func (l *queryStringLex) Error(msg string) {
	panic(msg)
}

func (l *queryStringLex) Lex(lval *yySymType) (rv int) {
	var err error

	for l.nextToken == nil {
		if l.currConsumed {
			l.nextRune, l.nextRuneSize, err = l.in.ReadRune()
			if err != nil && err == io.EOF {
				l.nextRune = 0
				l.atEOF = true
			} else if err != nil {
				return 0
			}
		}

		l.currState, l.currConsumed = l.currState(l, l.nextRune, l.atEOF)
		if l.currState == nil {
			return 0
		}
	}

	*lval = *l.nextToken
	rv = l.nextTokenType
	l.nextToken = nil
	l.nextTokenType = 0
	return rv
}

func newConditionStringLex(in io.Reader) *queryStringLex {
	return &queryStringLex{
		in:           bufio.NewReader(in),
		currState:    startState,
		currConsumed: true,
	}
}

type lexState func(l *queryStringLex, next rune, eof bool) (lexState, bool)

func startState(l *queryStringLex, next rune, eof bool) (lexState, bool) {
	if eof {
		return nil, false
	}

	// handle inside escape case up front
	if l.inEscape {
		l.inEscape = false
		l.buf += unescape(string(next))
		return inStrState, true
	}

	switch next {
	case '"', '/':
		return inPhraseState, true
	case '+', '-', ':', '>', '<', '=', '(', ')', '[', ']', '{', '}', '*':
		l.buf += string(next)
		return singleCharOpState, true
	}

	switch {
	case !l.inEscape && next == '\\':
		l.inEscape = true
		return startState, true
	case unicode.IsDigit(next):
		l.buf += string(next)
		return inNumOrStrState, true
	case !unicode.IsSpace(next):
		l.buf += string(next)
		return inStrState, true
	}

	// doesn't look like anything, just eat it and stay here
	l.reset()
	return startState, true
}

func inPhraseState(l *queryStringLex, next rune, eof bool) (lexState, bool) {
	// unterminated phrase eats the phrase
	if eof {
		l.Error("unterminated quote")
		return nil, false
	}

	// only a non-escaped " ends the phrase
	if !l.inEscape && (next == '"' || next == '/') {
		// end phrase
		switch next {
		case '"':
			l.nextTokenType = tPHRASE
		case '/':
			l.nextTokenType = tSLASH
		}
		l.nextToken = &yySymType{
			s: l.buf,
		}
		logDebugTokens("PHRASE - '%s'", l.nextToken.s)
		l.reset()
		return startState, true
	} else if !l.inEscape && next == '\\' {
		l.inEscape = true
	} else if l.inEscape {
		// if in escape, end it
		l.inEscape = false
		l.buf += unescape(string(next))
	} else {
		l.buf += string(next)
	}

	return inPhraseState, true
}

func singleCharOpState(l *queryStringLex, next rune, eof bool) (lexState, bool) {
	l.nextToken = &yySymType{}

	switch l.buf {
	case "+":
		l.nextTokenType = tPLUS
		logDebugTokens("PLUS")
	case "-":
		l.nextTokenType = tMINUS
		logDebugTokens("MINUS")
	case ":":
		l.nextTokenType = tCOLON
		logDebugTokens("COLON")
	case ">":
		l.nextTokenType = tGREATER
		logDebugTokens("GREATER")
	case "<":
		l.nextTokenType = tLESS
		logDebugTokens("LESS")
	case "=":
		l.nextTokenType = tEQUAL
		logDebugTokens("EQUAL")
	case "(":
		l.nextTokenType = tLEFTBRACKET
		logDebugTokens("LEFTBRACKET")
	case ")":
		l.nextTokenType = tRIGHTBRACKET
		logDebugTokens("RIGHTBRACKET")
	case "[":
		l.nextTokenType = tLEFTRANGE
		logDebugTokens("LEFTBRACKET")
	case "]":
		l.nextTokenType = tRIGHTRANGE
		logDebugTokens("tRIGHTRANGE")
	case "{":
		l.nextTokenType = tLEFTBRACES
		logDebugTokens("tLEFTBRACES")
	case "}":
		l.nextTokenType = tRIGHTBRACES
		logDebugTokens("tRIGHTBRACES")
	case "*":
		l.nextTokenType = tSTAR
		logDebugTokens("tSTAR")
	}

	l.reset()
	return startState, false
}

func inNumOrStrState(l *queryStringLex, next rune, eof bool) (lexState, bool) {
	// only a non-escaped space ends the tilde (or eof)
	if eof || (!l.inEscape && next == ' ' || next == ':' || next == ')' || next == ']' || next == '}') {
		// end number
		consumed := true
		if !eof && (next == ':' || next == ')' || next == ']' || next == '}') {
			consumed = false
		}

		l.nextTokenType = tNUMBER
		l.nextToken = &yySymType{
			s: l.buf,
		}
		logDebugTokens("NUMBER - '%s'", l.nextToken.s)
		l.reset()
		return startState, consumed
	} else if !l.inEscape && next == '\\' {
		l.inEscape = true
		return inNumOrStrState, true
	} else if l.inEscape {
		// if in escape, end it
		l.inEscape = false
		l.buf += unescape(string(next))
		// go directly to string, no successfully or unsuccessfully
		// escaped string results in a valid number
		return inStrState, true
	}

	// see where to go
	if !l.seenDot && next == '.' {
		// stay in this state
		l.seenDot = true
		l.buf += string(next)
		return inNumOrStrState, true
	} else if unicode.IsDigit(next) {
		l.buf += string(next)
		return inNumOrStrState, true
	}

	// doesn't look like an number, transition
	l.buf += string(next)
	return inStrState, true
}

func inStrState(l *queryStringLex, next rune, eof bool) (lexState, bool) {
	// end on non-escped space, colon, tilde, boost (or eof)
	if eof || (!l.inEscape && (next == ' ' || next == ':' || next == ')' || next == ']')) {
		// end string
		consumed := true
		if !eof && (next == ':' || next == ')' || next == ']') {
			consumed = false
		}

		switch l.buf {
		case "and", "AND":
			l.nextTokenType = tAND
			l.nextToken = &yySymType{}
			l.reset()
			return startState, consumed
		case "or", "OR":
			l.nextTokenType = tOR
			l.nextToken = &yySymType{}
			l.reset()
			return startState, consumed
		case "not", "NOT":
			l.nextTokenType = tNOT
			l.nextToken = &yySymType{}
			l.reset()
			return startState, consumed
		case "to", "TO":
			l.nextTokenType = tTO
			l.nextToken = &yySymType{}
			l.reset()
			return startState, consumed
		}

		l.nextTokenType = tSTRING
		l.nextToken = &yySymType{
			s: l.buf,
		}
		logDebugTokens("STRING - '%s'", l.nextToken.s)
		l.reset()

		return startState, consumed
	} else if !l.inEscape && next == '\\' {
		l.inEscape = true
	} else if l.inEscape {
		// if in escape, end it
		l.inEscape = false
		l.buf += unescape(string(next))
	} else {
		l.buf += string(next)
	}

	return inStrState, true
}

func logDebugTokens(format string, v ...interface{}) {
	if debugLexer {
		logrus.Printf(format, v...)
	}
}
