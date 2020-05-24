//go:generate statik -dest=.. -f -p=scripts -src=../scripts -include=*.sh

package handler

import (
	"log"

	"github.com/rakyll/statik/fs"

	//register data
	_ "github.com/jpillora/installer/scripts"
)

var installScript = []byte{}

func init() {
	//load static file
	hfs, err := fs.New()
	if err != nil {
		log.Fatalf("bad static file system: %s, fix statik", err)
	}
	installScript, err = fs.ReadFile(hfs, "/install.sh")
	if err != nil {
		log.Fatalf("read script file: %s, fix statik", err)
	}
}
