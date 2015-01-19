package main

import (
	"flag"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/jnschaeffer/septafs/septa"
)

var mountpoint string

func init() {
	flag.StringVar(&mountpoint, "mountpoint", "", "mount point for septafs")
}

func main() {
	flag.Parse()

	if mountpoint == "" {
		flag.Usage()
		os.Exit(2)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("septafs"),
		fuse.Subtype("septa"),
		fuse.LocalVolume(),
		fuse.VolumeName("SeptaFS"),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	if err = fs.Serve(c, septa.FS{}); err != nil {
		log.Fatal(err)
	}

	<-c.Ready
	if err = c.MountError; err != nil {
		log.Fatal(err)
	}
}
