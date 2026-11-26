// Copyright © 2026-present the longxi-go/ioc authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compiler

import (
	"errors"
	"strings"
)

const (
	flagDoubleQuote   = "\""
	flagSingleQuote   = "'"
	flagWordSeparator = " "

	goflagsWordContainsBothQuoteCharactersMessage = "GOFLAGS word contains both quote characters"
	unterminatedGOFLAGSQuoteMessage               = "unterminated GOFLAGS quote"
)

const flagWhitespace = " \t\n\r"

// splitFlagWords follows the Go command's GOFLAGS grammar: quotes surround a
// whole word and their contents are literal, without shell escape processing.
func splitFlagWords(input string) ([]string, error) {
	var words []string
	for input = strings.TrimLeft(input, flagWhitespace); input != ""; input = strings.TrimLeft(input, flagWhitespace) {
		if input[0] == flagSingleQuote[0] || input[0] == flagDoubleQuote[0] {
			quote := input[:1]
			word, rest, closed := strings.Cut(input[1:], quote)
			if !closed {
				return nil, errors.New(unterminatedGOFLAGSQuoteMessage)
			}
			words = append(words, word)
			input = rest
			continue
		}
		end := strings.IndexAny(input, flagWhitespace)
		if end < 0 {
			words = append(words, input)
			break
		}
		words = append(words, input[:end])
		input = input[end:]
	}
	return words, nil
}

func joinFlagWords(words []string) (string, error) {
	quoted := make([]string, 0, len(words))
	for _, word := range words {
		if word == "" ||
			strings.ContainsAny(word, flagWhitespace) ||
			word[0] == flagSingleQuote[0] ||
			word[0] == flagDoubleQuote[0] {
			switch {
			case !strings.Contains(word, flagSingleQuote):
				word = flagSingleQuote + word + flagSingleQuote
			case !strings.Contains(word, flagDoubleQuote):
				word = flagDoubleQuote + word + flagDoubleQuote
			default:
				return "", errors.New(goflagsWordContainsBothQuoteCharactersMessage)
			}
		}
		quoted = append(quoted, word)
	}
	return strings.Join(quoted, flagWordSeparator), nil
}
