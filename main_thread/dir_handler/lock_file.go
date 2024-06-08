package dir_handler

import (
	"errors"
	"fmt"

	"log"
	"os"
	"time"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

type LockFile struct {
	name string
}

type err_type struct {
	err string
}

func (err *err_type) Error() string {
	return err.err
}

var (
	ErrExist = &err_type{err: "file already exists"}
)

func Lock(file_path string) (lf *LockFile, err error) {
	err2 := os.Mkdir(file_path, os.FileMode(os.ModeExclusive))
	if err2 != nil {
		if errors.Is(err2, os.ErrExist) {
			return nil, ErrExist
		}
		err = &ftp_context.LogItem{
			Location: fmt.Sprintf(`Lock("%s" string) (lf *LockFile, err error)`, file_path),
			Time:     time.Now(),
			After:    "err2 := os.Mkdir(file_path, os.FileMode(os.ModeExclusive))",
			Message:  err2.Error(),
		}
		return nil, err
	}

	return
}

func (lf *LockFile) Unlock() error {
	err1 := os.Remove(lf.name)
	log.Println(&ftp_context.LogItem{
		Location: `func (lf *LockFile) Unlock() error`,
		Time:     time.Now(),
		After:    fmt.Sprintf(`err1 := os.Remove("%s")`, lf.name),
		Message:  err1.Error(),
	})
	return err1
}
