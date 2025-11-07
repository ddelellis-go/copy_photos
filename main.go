package main

//TODO: support multiple root dirs?
//TODO: report to redis for status readout
//TODO: image similarity comparison
import (
	"os"
	"fmt"
	"maps"
	"time"
	"io/fs"
	"slices"
	"encoding/json"

	"github.com/moby/sys/mount"
	"self_utilities/progress_bar"
)

var opts Opts

func mountPointName() string {
	return fmt.Sprintf("%s/%s.%X", opts.TempDir, opts.TempPrefix, time.Now().Unix())
}

func main() {
	var rc int
	var err error

	defer func() {
		if err != nil {
			fmt.Println(err)
			rc = 1
		}
		os.Exit(rc)
	}()
	opts, err = parseFlags()
	if err != nil {
		return
	}
	var photosUid, photosGid int

	photosUid, photosGid, err = getIds()
	if err != nil {
		err = fmt.Errorf("Error getting uid/gid of target root dir: %w", err)
		return
	}
	debugf("user/group ids of target dir: %d, %d", photosUid, photosGid)

	mountPoint := mountPointName()

	//TODO: turn this into a struct that can have functions, then you can just do `defer mountedDirs.unmount()` with all this logic hidden away
	mountedDirs, err := findAndMountDisks(mountPoint)
	defer func() {
		if !opts.KeepMounts {
			for _, v := range mountedDirs {
				debugf("unmounting %s", v)
				err = mount.Unmount(v)
				if err != nil {
					err = fmt.Errorf("Error unmounting filesystem: %w", err)
					return
				}
				debugf("umonted disk from %v", v)
				vFS := os.DirFS(v)
				foundFiles := false
				fErr := fs.WalkDir(vFS, ".", func(path string, entry fs.DirEntry, fErr error) error {
					if entry.Type().IsRegular() {
						fErr = fmt.Errorf("found regular file: %s/%s", v, path)
						foundFiles = true
					}
					return fErr
				})
				if fErr != nil || foundFiles {
					debugf("can't clean up mountpoint %s: %v", v, err)
				} else {
					debugf("no regular files found in %s, cleaning up", v)
					os.RemoveAll(v)
				}
			}
		} else {
			debug("not unmounting disks")
		}
	}()

	if err != nil {
		err = fmt.Errorf("Error finding or mounting an applicable block device: %w", err)
		return
	}
	fileQueue := make(map[string]TargetFile)

	for _, mDir := range mountedDirs {
		fsRoot := os.DirFS(mDir)
		debug("found fsroot:", fsRoot)
		err = fs.WalkDir(fsRoot, ".", func(path string, entry fs.DirEntry, err error) error {
			return findFiles(fileQueue, opts.RootPath, mDir, path, entry, err)
		})

		if err != nil {
			err = fmt.Errorf("Error traversing filesystem: %w", err)
			return
		}
	}

	bar, err := progress_bar.NewProgress(len(fileQueue), 20)
	if err != nil {
		debugf("Failed to initialize progress bar: %v", err)
		err = nil
	}

	debug(fmt.Sprintf("found %d files", len(fileQueue)))
	for i, k := range slices.Sorted(maps.Keys(fileQueue)) {
		v := fileQueue[k]
		debugf("%d of %d", i+1, len(fileQueue))
		err = v.CopyFromDisk()
		if err != nil {
			return
		}
		bar.Draw("Copying files ", v.SourceInfo.Name(), i+1)
	}

	debug("forcing owner/perms")
	photoDirRoot := os.DirFS(opts.RootPath)

	err = fs.WalkDir(photoDirRoot, ".", func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return fmt.Errorf("encountered error setting permissions: %w", e)
		}
		if !(d.Type().IsRegular() || d.IsDir()) {
			return nil
		}
		var bitMode fs.FileMode
		if d.IsDir() {
			bitMode = 0775
		} else {
			bitMode = 0664
		}

		fmt.Printf("Forcing permissions on %s \r", path)
		return setPermissions(GeneratePath(opts.RootPath, path), photosUid, photosGid, bitMode)
	})

	if err != nil {
		err = fmt.Errorf("Error forcing owner/perms of content: %w", err)
		return
	}
	debug("done")
}

func getIds() (user, group int, err error) {
	if opts.UserID > -1 && opts.GroupID > -1 {
		user = opts.UserID
		group = opts.GroupID
		return
	}
	path := opts.RootPath

	u, g, err := getOwnership(path)
	if err != nil {
		return
	}

	user = u
	group = g
	return
}

func debug(a ...any) {
	if opts.Verbose {
		fmt.Printf("%s ", time.Now().Format("2006-01-02 15:04:05.000Z07:00"))
		fmt.Println(a...)
	}
}

func debugf(s string, a ...any) {
	debug(fmt.Sprintf(s, a...))
}

type StatusReport struct {
	TTL    time.Duration
	Detail []string
	Status int
}

func jsonDump(a any) {
	b, e := json.MarshalIndent(a, "#", "    ")
	if e != nil {
		debugf("issue trying to marshal data: %v", e)
	}
	debug(string(b))

}
