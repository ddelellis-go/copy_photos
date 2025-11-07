package main

import (
	"fmt"
	"github.com/moby/sys/mount"
	"os"
)

func (dev *Dev) Mount(mountPoint, mode string) (err error) {
	err = os.MkdirAll(mountPoint, 0755)
	if err != nil {
		err = fmt.Errorf("Unable to create temp dir for mountpoint: %v", err)
		return
	}
	debug("made mountpoint", mountPoint)

	err = mount.Mount(dev.DevID, mountPoint, dev.Filesystem, mode)
	if err != nil {
		err = fmt.Errorf("Unable to mount disk to mountpoint: %v", err)
		return
	}
	debug("mounted disk")
	if opts.NikonFile {
		if opts.NikonFilePath == "" {
			opts.NikonFilePath = nikonFile
		}
	}
	if opts.NikonFilePath != "" {
		r, e := readData(GeneratePath(mountPoint, nikonFile), 512)
		if e != nil {
			err = fmt.Errorf("Called with --nikon or --nikonfile but %s not found in root of %s", nikonFile, dev.DevID)
			return
		}

		empty := make([]byte,512)
		if !compareByteSlices(r, &empty) {
			err = fmt.Errorf("called with --nikon or --nikonfile but %s at root of %s is not a 512-byte file of 0s", nikonFile, dev.DevID)
			return
		}
	}
	return
}

type Dev struct {
	DevID, Filesystem string
}
