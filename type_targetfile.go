package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Values are defined in increasing amount of work needed
const (
	NoAction    = iota //No action will be taken
	NeedsCopy          //File will be copied
	NeedsVerify        //If target file is an incomplete copy of the original, it will be overwritten. otherwise, a conflict will be thrown
	Conflict           //Requires manual intervention
)

// A TargetFile struct stores the paths for the initial copy (archive) of a copied photo, and the non-hidden sortable file
// the sortable file will be a hardlink to the archive file.
// a more flexible struct would have the Sourcefile and info as named fields, and then a list of generic path/name pairs that would all be hardlinks from the initial copy
type TargetFile struct {
	SourceFile string
	SourceInfo fs.FileInfo

	Data *[]byte
	DataBuff	*bytes.Buffer

	TgtFile    FileWithDirPath
	TargetFile string
	TargetStat fs.FileInfo

	Targets []FileWithDirPath
	Action  int
}

type FileWithDirPath struct {
	Path string
	File string
}

func (t *TargetFile) Generate(rootPath, devDir, filePath string, f fs.DirEntry, linkDirs []string) (e error) {
	debugf("rootpath:%s,sourcefile:%s", rootPath, f.Name())
	if len(linkDirs) < 1 {
		e = fmt.Errorf("impossible code path?")
	}
	t.SourceFile = GeneratePath(devDir, filePath)
	t.SourceInfo, e = f.Info()
	if e != nil {
		return
	}

	t.Targets = make([]FileWithDirPath, len(linkDirs))

	dateDir := t.SourceInfo.ModTime().Local().Format(opts.DirFormat)
	for i, v := range linkDirs {
		var dirPath string
		if opts.KeepPaths {
			dirPath, _ = strings.CutSuffix(filePath, f.Name())
		}
		t.Targets[i].Path = GeneratePath(rootPath, v, dateDir, dirPath)

		endName := f.Name()
		if opts.FlatPaths {
			endName = strings.ReplaceAll(filePath, "/", ".")
		}
		t.Targets[i].File = endName
	}

	t.TargetFile = GeneratePath(t.Targets[0].Path, t.Targets[0].File)

	return
}

func (target *TargetFile) CompareSrcDest() (b bool,e error) {
	return
}

func (target *TargetFile) CopyFromDisk() (err error) {
	debugf("copying file from/to <%s>/<%s>", target.SourceFile, target.TargetFile)
	err = target.MakePaths()
	if err != nil {
		err = fmt.Errorf("Failed creating target path: %v", err)
		return
	}

	target.Data, err = readData(target.SourceFile, target.SourceInfo.Size())
	if err != nil {
		err = fmt.Errorf("Failed reading source file: %v", err)
		return
	}

	if target.Action == NeedsVerify {
		var extantData *[]byte

		debug("need to compare target file contents")
		extantData, err = readData(target.TargetFile, target.TargetStat.Size())
		if err != nil {
			err = fmt.Errorf("failed verifying target file: %v", err)
			return
		}
		debug("read extant data for target,size:", target.SourceInfo.Size())

		if compareByteSlices(extantData, target.Data) {
			debug("tgt is an incomplete copy of src", target.TargetFile, target.SourceFile)
			//target file is bytewise identical to source, but incomplete
			target.Action = NeedsCopy
		} else {
			//target file has some byte value that the source does not
			debug("tgt contains data that src does not")
			target.Action = Conflict
			err = fmt.Errorf("conflict on copying file %s to %s", target.SourceFile, target.TargetFile)
			return
		}
	}

	if target.Action != NeedsCopy {
		err = fmt.Errorf("file %s somehow got to an uncreachable code path while processing copy tasks", target.SourceFile)
		return
	}
	err = writeData(target.Data, target.TargetFile)
	if err != nil {
		return
	}
	debug("wrote file ", target.TargetFile)

	//force atime/mtime/ctime to be `mtime.Unix()` in syscall.Utime
	err = os.Chtimes(target.TargetFile, target.SourceInfo.ModTime(), target.SourceInfo.ModTime())
	if err != nil {
		err = fmt.Errorf("error setting atime/mtime for copied file (%s) to match source: %v", target.TargetFile, err)
		return
	}
	debug("set time")

	if len(target.Targets) > 1 {
		target.Targets = target.Targets[1:]
	} else {
		target.Targets = nil
	}
	return
}

// This should create the dir paths for ArchivePath and SortPath
func (t *TargetFile) MakePaths() error {
	for _, entry := range t.Targets {
		dirErr := os.MkdirAll(entry.Path, 0775)
		if dirErr != nil {
			return fmt.Errorf("Error creating directory: %s: %v", entry.Path, dirErr)
		}
	}
	return nil
}

func GeneratePath(s ...string) string {
	return CleanPath(strings.Join(s, string(os.PathSeparator)))
}

func CleanPath(s string) string {
	s = strings.ReplaceAll(s, "/./", "/")
	s = strings.ReplaceAll(s, "//", "/")
	return s
}
