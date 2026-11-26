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

package command

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/longxi-go/ioc/pkg/codegen"
)

const (
	exitSuccess          = 0
	exitFailure          = 1
	exitUsage            = 2
	checkCommand         = "check"
	commandLabel         = "ioctl "
	currentDirectory     = "."
	directoryFlag        = "dir"
	entryFlag            = "entry"
	explainCommand       = "explain"
	generateCommand      = "generate"
	graphCommand         = "graph"
	helpCommand          = "help"
	helpLongFlag         = "--help"
	helpShortFlag        = "-h"
	libraryFlag          = "library"
	listSeparator        = ","
	reportSummaryFormat  = "%s: %d definitions, %d generated files\n"
	rootsFlag            = "roots"
	tagsFlag             = "tags"
	unknownCommandFormat = "unknown command %q\n%s"

	applicationEntryPackageMessage           = "application entry package"
	commaSeparatedBuildTagsMessage           = "comma-separated build tags"
	commaSeparatedOwnedSourcePatternsMessage = "comma-separated owned source patterns"
	moduleDirectoryMessage                   = "module directory"
	publishReusableLibraryMetadataMessage    = "publish reusable library metadata"
	unexpectedPositionalArgumentsMessage     = "unexpected positional arguments"
)

const usage = "usage: ioctl generate|check|explain|graph [--dir .] [--entry ./cmd/app] [--library] [--tags a,b] [--roots ./services/...]\n"

// Run executes one command and returns a process-compatible status. Explain
// emits candidate metadata; graph emits all declared potential edges as JSON.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if _, err := fmt.Fprint(stderr, usage); err != nil {
			return exitFailure
		}
		return exitUsage
	}
	verb := args[0]
	if verb == helpCommand || verb == helpLongFlag || verb == helpShortFlag {
		if _, err := fmt.Fprint(stdout, usage); err != nil {
			return exitFailure
		}
		return exitSuccess
	}
	if verb != generateCommand &&
		verb != checkCommand &&
		verb != explainCommand &&
		verb != graphCommand {
		if _, err := fmt.Fprintf(stderr, unknownCommandFormat, verb, usage); err != nil {
			return exitFailure
		}
		return exitUsage
	}
	flags := flag.NewFlagSet(commandLabel+verb, flag.ContinueOnError)
	flags.SetOutput(stderr)
	var options codegen.Options
	var tags, roots string
	flags.StringVar(&options.Dir, directoryFlag, currentDirectory, moduleDirectoryMessage)
	flags.StringVar(&options.Entry, entryFlag, "", applicationEntryPackageMessage)
	flags.BoolVar(&options.Library, libraryFlag, false, publishReusableLibraryMetadataMessage)
	flags.StringVar(&tags, tagsFlag, "", commaSeparatedBuildTagsMessage)
	flags.StringVar(&roots, rootsFlag, "", commaSeparatedOwnedSourcePatternsMessage)
	if err := flags.Parse(args[1:]); err != nil {
		return exitUsage
	}
	if flags.NArg() != 0 {
		if _, err := fmt.Fprintln(stderr, unexpectedPositionalArgumentsMessage); err != nil {
			return exitFailure
		}
		return exitUsage
	}
	if tags != "" {
		options.Tags = strings.Split(tags, listSeparator)
	}
	if roots != "" {
		options.Roots = strings.Split(roots, listSeparator)
	}
	options.Check = verb == checkCommand
	var report *codegen.Report
	var err error
	if verb == explainCommand || verb == graphCommand {
		report, err = codegen.Inspect(ctx, options)
	} else {
		report, err = codegen.Run(ctx, options)
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return exitFailure
		}
		return exitFailure
	}
	switch verb {
	case explainCommand:
		err = json.NewEncoder(stdout).Encode(report.Definitions)
	case graphCommand:
		err = json.NewEncoder(stdout).Encode(report.Edges)
	default:
		_, err = fmt.Fprintf(stdout, reportSummaryFormat, verb, len(report.Definitions), len(report.Files))
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, err); writeErr != nil {
			return exitFailure
		}
		return exitFailure
	}
	return exitSuccess
}
