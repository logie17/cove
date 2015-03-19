// Package cove is a thin wrapper around the go toolchain
package cove

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

//GoCmd takes the sub and args and prepares a command like 'go sub arg1 arg2...'
func GoCmd(sub string, args ...string) *exec.Cmd {
	arguments := append([]string{sub}, args...)
	return exec.Command("go", arguments...)
}

// Packages gets all packages that match any of the paths.
// The package list will only contain 1 entry per package in sorted order.
// Invalid paths will generate an error, but will not stop the evaluation of the other paths.
func Packages(paths ...string) ([]string, error) {
	packs, err := output(GoCmd("list", paths...))
	sort.Strings(packs)
	return packs, err
}

// PackageJSON takes a SINGLE fully qualified package import path and decodes the 'go list -json' response.
// See $GOROOT/src/cmd/go/list.go for documentation on the json output.
func PackageJSON(pack string, v interface{}) error {
	return pipeWith(GoCmd("list", "-json", pack), func(stdout io.Reader) error {
		return json.NewDecoder(stdout).Decode(v)
	})
}

// CoverageProfile creates a cover profile file for the provided package
// The file is created in outdir.  The parameter short sets whether to run
// all tests or only the short ones.
// If a profile is able to be created its file name is returned.
func CoverageProfile(short bool, outdir string, pack string) (string, error) {
	profile := getProfileFileName(outdir, pack)

	if err := run(GoCmd("test", pack, fmt.Sprintf("-coverprofile=%s", profile), getShort(short))); err != nil {
		return "", fmt.Errorf("%s:%v", pack, err)
	}

	if _, err := os.Stat(profile); err != nil {
		return "", nil
	}

	return profile, nil
}

// CoverageReport turns the profile into a report using 'go tool cover'
func CoverageReport(profile string, outdir string) (string, error) {
	report := getReportFileName(profile, outdir)
	if err := run(GoCmd("tool", "cover", fmt.Sprintf("-html=%s", profile), "-o", report)); err != nil {
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
