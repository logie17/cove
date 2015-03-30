// Package cove is a thin wrapper around the go toolchain
package cove

// Copyright 2015 MediaMath <http://www.mediamath.com>.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MediaMath/cove/cmd"
)

//GoCmd takes the sub and args and prepares a command like 'go sub arg1 arg2...'
func GoCmd(sub string, args ...string) *exec.Cmd {
	arguments := append([]string{sub}, args...)
	return exec.Command("go", arguments...)
}

// Get runs 'go get pack'
func Get(pack string) error {
	return cmd.Run(GoCmd("get", pack))
}

// PackageExists checks to see if a given package name exists
func PackageExists(pack string) bool {
	err := cmd.Run(GoCmd("list", pack))
	return err == nil
}

// MissingDependencies returns any go packages that are missing for a given package.
func MissingDependencies(pack string) ([]string, error) {
	var deps []string
	var parsed missing
	if err := PackageJSON(pack, &parsed); err != nil {
		return deps, err
	}

	return missingFromParsed(&parsed)
}

func missingFromParsed(parsed *missing) ([]string, error) {
	if !parsed.Incomplete {
		return []string{}, nil
	}

	seen := make(map[string]bool)
	var missing []string
	for _, errs := range parsed.DepsErrors {
		for _, imports := range errs.ImportStack {
			if _, contains := seen[imports]; !contains {
				seen[imports] = true
				missing = append(missing, imports)
			}
		}
	}
	return missing, nil
}

type missing struct {
	Incomplete bool
	DepsErrors []struct {
		ImportStack []string
	}
}

// Packages gets all packages that match any of the paths.
// The package list will only contain 1 entry per package in sorted order.
// Invalid paths will generate an error, but will not stop the evaluation of the other paths.
func Packages(paths ...string) ([]string, error) {
	packs, err := cmd.Output(GoCmd("list", paths...))
	sort.Strings(packs)
	return packs, err
}

// PackageJSON takes a SINGLE fully qualified package import path and decodes the 'go list -json' response.
// See $GOROOT/src/cmd/go/list.go for documentation on the json output.
func PackageJSON(pack string, v interface{}) error {
	return cmd.PipeWith(GoCmd("list", "-json", pack), func(stdout io.Reader) error {
		return json.NewDecoder(stdout).Decode(v)
	})
}

// CoverageProfile creates a cover profile file for the provided package
// The file is created in outdir.  The parameter short sets whether to run
// all tests or only the short ones. The mode specifies which style of cover mode is used.
// If a profile is able to be created its file name is returned.
func CoverageProfile(short bool, mode string, outdir string, pack string) (string, error) {
	if direrr := os.MkdirAll(outdir, 0744); direrr != nil {
		return "", direrr
	}

	profile := getProfileFileName(outdir, pack)

	if err := cmd.Run(GoCmd("test", pack, fmt.Sprintf("-covermode=%s", mode), fmt.Sprintf("-coverprofile=%s", profile), getShort(short))); err != nil {
		return "", fmt.Errorf("%s:%v:%v", pack, err, profile)
	}

	if _, err := os.Stat(profile); os.IsNotExist(err) {
		return "", nil
	}

	return profile, nil
}

// CoverageReport turns the profile into a report using 'go tool cover'
func CoverageReport(profile string, outdir string) (string, error) {
	report := getReportFileName(profile, outdir)
	if err := cmd.Run(GoCmd("tool", "cover", fmt.Sprintf("-html=%s", profile), "-o", report)); err != nil {
		return "", err
	}

	return report, nil
}

func getReportFileName(profile string, outdir string) string {
	report := filepath.Base(profile)
	extension := filepath.Ext(report)
	name := report[0 : len(report)-len(extension)]
	fullPath := filepath.Join(outdir, name)
	return fmt.Sprintf("%s.html", fullPath)
}

func getProfileFileName(outdir string, pack string) string {
	profile := strings.Replace(pack, "/", ".", -1)
	fullPath := filepath.Join(outdir, profile)
	return fmt.Sprintf("%s.out", fullPath)
}

func getShort(short bool) string {
	if short {
		return "-short"
	}

	return ""
}
