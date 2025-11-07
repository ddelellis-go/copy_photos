package main

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// operates on a single dir entry, this function is specific to source files on the mounted SD card
func findFiles(queue map[string]TargetFile, rootPath, devDir, thisPath string, d fs.DirEntry, err error) error {
	debugf("rootpath:<%s>, operating file:<%s>", rootPath, thisPath)
	debugf("%+#v", d)
	var task TargetFile

	if err != nil {
		return fmt.Errorf("error on path:%v", err)
	}

	if d.IsDir() || !d.Type().IsRegular() {
		return nil
	}

	task, err = prepareCopy(rootPath, devDir, thisPath, d)
	if err != nil {
		return fmt.Errorf("error operating on %s: %s\n", thisPath, err)
	}

	extant, ohNo := queue[task.TargetFile]
	if ohNo {
		return fmt.Errorf("Conflict detected: '%s' and '%s' have the same target file: '%s' ", extant.SourceFile, thisPath, task.TargetFile)
	}
	switch task.Action {
	case NoAction:
		debug(thisPath + " needs no action")
		return nil
	case NeedsCopy:
		debug(thisPath + " will be copied")
		//continue
	case NeedsVerify:
		debug(thisPath + " will be copied, if target file is an incomplete copy")
		//continue
	case Conflict:
		debug("target does not appear to be an incomplete copy of " + thisPath)
		if opts.SkipConflicts {
			debugf("conflict detected %s <> %s, but skipping", task.TargetFile, thisPath)
			return nil
		}
		return fmt.Errorf("conflict detected: %s has different contents than %s", task.TargetFile, thisPath)
	default:
		debug("i have no idea what do with:", thisPath, task.TargetFile)
		return fmt.Errorf("unhandled status for copying %s to %s: %d", thisPath, task.TargetFile, task.Action)
	}

	queue[task.TargetFile] = task
	return nil
}

func prepareCopy(rootPath, devDir, thisPath string, d fs.DirEntry) (target TargetFile, err error) {
	var tgt TargetFile
	srcErr := tgt.Generate(rootPath, devDir, thisPath, d, opts.TargetDirs)
	if srcErr != nil {
		err = fmt.Errorf("Error generating target paths from source file (%s) info: %v", d.Name(), srcErr)
		return
	}
	debug("got src file meta:", d.Name())
	if tgt.SourceInfo.Size() < opts.MinSize {
		debug("this file is below minsize", tgt.SourceInfo.Size(), opts.MinSize)
		return
	}

	//does the file already exist?
	fileInfo, eNoEnt := os.Stat(tgt.TargetFile)
	if eNoEnt == nil {
		//no error means a file is already there
		tgt.TargetStat = fileInfo
		tgt.Action = compareSrcTgt(tgt.SourceInfo, fileInfo)
	} else {
		tgt.Action = NeedsCopy
	}

	debug(fmt.Sprintf("Op level for file %s is %d", tgt.TargetFile, tgt.Action))

	target = tgt
	return
}

func readData(src string, expectedSize int64) (data *[]byte, err error) {
	debugf("reading file <%s>", src)
	raw, err := os.ReadFile(src)
	if err != nil {
		err = fmt.Errorf("Error trying to read file contents of %s: %v", src, err)
		return
	}
	readSize := int64(len(raw))
	if readSize != expectedSize {
		err = fmt.Errorf("Wrong number of bytes read (%d) from %s (%d expected)", readSize, src, expectedSize)
		return
	}
	debug("raw data is of correct expected size:", readSize)

	data = &raw
	return
}

func writeData(data *[]byte, target string) (err error) {
	err = os.WriteFile(target, *data, 0664)
	if err != nil {
		err = fmt.Errorf("Error trying to write %s: %v", target, err)

		delErr := os.Remove(target)
		if delErr != nil {
			err = fmt.Errorf("%v. Also failed to delete partial file: %v", err, delErr)
		}
		return
	}

	return
}

func compareSrcTgt(src, tgt fs.FileInfo) int {
	if tgt.Size() < src.Size() {
		debug("target is smaller than source, possible incomplete copy")
		//This may have been an incomplete filecopy
		return NeedsVerify
	}
	if tgt.Size() == src.Size() && tgt.ModTime() == src.ModTime() {
		debug("target and source have same size/mtime")
		//These are likely the same file
		return NoAction
	}

	//some kind of conflict
	debug("target is larger than source and/or has different mtime")
	return Conflict
}

func getFileStat(path string) (uid, gid int, mode fs.FileMode, err error) {
	stat, err := os.Stat(path)
	if err != nil {
		return
	}

	sys := stat.Sys().(*syscall.Stat_t)
	uid = int(sys.Uid)
	gid = int(sys.Gid)
	mode = fs.FileMode(sys.Mode)
	return
}

func getOwnership(path string) (uid, gid int, err error) {
	uid, gid, _, err = getFileStat(path)

	return
}

func setPermissions(path string, uid, gid int, bitMode fs.FileMode) error {
	u, g, m, err := getFileStat(path)
	if err != nil {
		return fmt.Errorf("Failed checking ownership of %s: [%w]", path, err)
	}
	if ( uid != u || gid != g) {
		chownErr := os.Chown(path, uid, gid)
		if chownErr != nil {
			return fmt.Errorf("error trying to change owner of copied file (%s) to match dir owner:group (%d:%d) [%w]", path, uid, gid, chownErr)
		}

		debug("changed owner of ", path)
	}

	if m != bitMode {
		chmodErr := os.Chmod(path, bitMode)
		if chmodErr != nil {
			return fmt.Errorf("Error setting permissions on %s to %v: %w", path, bitMode, chmodErr)
		}
		debug("set permissions on ", path)
	}

	return nil
}

func compareByteSlices(short, full *[]byte) (status bool) {
	m := len(*full)
	if len(*short) < m {
		debugf("file on disk (%d) is shorter than souce file (%d)", len(*short), len(*full))
	}
	for i := 0; i < m; i++{
		if (*short)[i] != (*full)[i] {
			debug("differing value found at byte ", i)
			return
		}
	}
	status = true
	return
}
