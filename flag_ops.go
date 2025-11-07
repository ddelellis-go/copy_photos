package main

import (
	flags "github.com/jessevdk/go-flags"
)

// a A b B c C . D e E . F . G . H i I j J k . l . m M . . o O p P q Q . R s S . . . U . V w W x X y Y z Z
type Opts struct {
	DevIDs        []string `short:"d" long:"dev" description:"multiple instances allowed. full device path (ex: '/dev/sdb1') to scan. Overrides fstypes,fslabels options entirely. Will not override nikonfile or nikon flag"`
	RootPath      string   `short:"r" long:"rootdir" description:"root of directory path, relative to CWD, to copy files into." default:"photos"`
	TargetDirs    []string `short:"t" long:"targetdir" description:"multiple instances allowed. path, relative to 'rootdir', to create date-formatted directory names. call multiple times to create hardlink clones of the initial directory" default:"."`
	FlatPaths     bool     `short:"F" long:"flattenpath" description:"prepend relative to the filesystem root path of the file to its name in the copy target, replacing '/' with '.'"`
	KeepPaths     bool     `short:"K" long:"keeppath" description:"sort files into directories, but keep each file nested in a subdirectory tree matching source filesystem"`
	SkipConflicts bool     `short:"S" long:"skip-conflicts" description:"if a conflict is detected, skip to the next file instead of dying"`

	KeepMounts bool   `long:"keepmounts" description:"don't unmount filesystems on exit or failure"`
	DirFormat  string `short:"f" long:"dirformat" description:"go-time formatted string for target directory names to move files to" default:"20060102"`

	FsTypes  []string `short:"T" long:"fstypes" description:"multiple instances allowed. filesystem type to examine" default:"exfat" default:"fat32"`
	FsLabels []string `short:"L" long:"fslabels" description:"multiple instances allowed. filesystem labels to examine, not case-sensitive." default:"nikon"`

	NikonFile     bool   `short:"N" long:"nikon" description:"look for and validate a 512byte file named NIKON001.DSC in the root of the drive"`
	NikonFilePath string `short:"n" long:"nikonfile" description:"look for and validate a 512byte file with specified name in the root of the drive. Overrides default path assumed by nikon flag"`

	BlkidCache string `long:"blkidtab" description:"path, relative to CWD, for the contents of blkid" default:"/run/blkid/blkid.tab"`
	TempPrefix string `long:"tempname" description:"name of directory to create as a temp mountpoint for discovered block device" default:"camera_sd"`
	TempDir    string `long:"tempdir" description:"root dir, relative to CWD, to create mountpoint for the discovered block device" default:"temp"`

	UserID  int   `short:"u" long:"uid"  description:"numeric user ID to set ownership of files/dirs. setting both user and group IDs will skip checks for rootdir existing" default:"-1" default-mask:"<UID of current user>"`
	GroupID int   `short:"g" long:"gid"  description:"numeric group ID to set ownership of files/dirs. setting both user and group IDs will skip checks for rootdir existing" default:"-1" default-mask:"<GID of current user>"`
	MinSize int64 `long:"minsize" description:"minimum filesize, in bytes, to be included in the file copy list" default:"1000"`

	Verbose bool `short:"v" long:"verbose" description:"print ops while processing"`
}

func parseFlags() (o Opts, err error) {
	_, err = flags.Parse(&o)
	return
}
