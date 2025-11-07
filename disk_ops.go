package main

import (
	"fmt"
	"golang.org/x/net/html"
	"os"
	"strings"
)

const fsTypeKey = "type"
const fsLabelKey = "label"
const mountMode = "ro"
const nikonFile = "NIKON001.DSC"

func findAndMountDisks(mountDir string) (mountedDirs []string, err error) {
	targetDisks, err := getDiskPaths()
	if err != nil {
		err = fmt.Errorf("Unable to locate applicable disk: %v", err)
		return
	}
	if len(targetDisks) == 0 {
		debug("no valid disks found")
		return
	}
	debug("target disk list", targetDisks)

	for _, v := range targetDisks {
		mountLocation := mountDir + v.DevID
		debugf("mounting %s to %s", v.DevID, mountLocation)
		err = v.Mount(mountLocation, mountMode)
		if err != nil {
			return
		}
		mountedDirs = append(mountedDirs, mountLocation)
	}

	return
}

func getDiskPaths() (devIDs []Dev, err error) {
	if opts.DevIDs == nil {
		//if no devid is specified, it will scan all devids
		opts.DevIDs = []string{""}
	} else {
		//if devids are specified, assume that whatever label/fstype is discovered is expected

		//label is validated by checking if discovered label has an expected label as a prefix
		//fstype is validated by checking if expected fs is empty string, or if discovered fs is an exact match of one of the expected filesystems
		opts.FsTypes = []string{""}
		opts.FsLabels = []string{""}
	}

	raw, err := os.Open(opts.BlkidCache)
	defer raw.Close()

	if err != nil {
		err = fmt.Errorf("Failed opening %s:%s", opts.BlkidCache, err)
		return
	}

	data, err := html.Parse(raw)
	if err != nil {
		err = fmt.Errorf("Failed parsing dev data: %s", err)
		return
	}

	for n := range data.Descendants() {
		ok, fs := validateAttrs(n)
		if ok {
			devIDs = append(devIDs, Dev{DevID: n.Data, Filesystem: fs})
		}
	}
	return
}

func validateAttrs(n *html.Node) (ok bool, fs string) {
	if n.Type == html.TextNode && len(n.Parent.Attr) > 0 {
		var hasCorrectLabel, hasCorrectFS bool

		debug(fmt.Sprintf("examining attributes for %s", n.Data))

		nameMatch, _ := findEmptyOrMatch(n.Data, opts.DevIDs)
		if !nameMatch {
			debugf("%s is not a specified devname", n.Data)
			return
		}

		for _, v := range n.Parent.Attr {
			debug(fmt.Sprintf("comparing attr %s to %s", v.Key, fsLabelKey))
			if v.Key == fsLabelKey {
				hasCorrectLabel = findLabelWithPrefix(opts.FsLabels, v.Val)
			}

			if v.Key == fsTypeKey {
				hasCorrectFS, fs = findFsType(v.Val, opts.FsTypes)
			}
			debug("found fs with expected label:", hasCorrectLabel, "; found fs with expected type:", hasCorrectFS)
			if hasCorrectLabel && hasCorrectFS && fs != "" {
				ok = true
			}
		}
		debug(fmt.Sprintf("done with  %s", n.Data))
	}

	return
}

func findFsType(fs string, fstypes []string) (bool, string) {
	return findEmptyOrMatch(fs, fstypes)
}
func findEmptyOrMatch(discovered string, expected []string) (bool, string) {
	for _, exp := range expected {
		if exp == "" || exp == discovered {
			return true, discovered
		}
	}
	return false, ""
}

func findLabelWithPrefix(prefixes []string, label string) bool {
	for _, prefix := range prefixes {
		debug(fmt.Sprintf("is %s a prefix to '%s'", prefix, label))
		if strings.HasPrefix(strings.ToLower(label), prefix) {
			return true
		}
	}
	return false
}
