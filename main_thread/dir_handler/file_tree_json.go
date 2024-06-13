package dir_handler

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"sync"
	"time"

	"os"

	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"

	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
	// "golang.org/x/sync/syncmap"
)

var FileTree = NewFileTreeJson()

type FileState string

const (
	FileStateHashed     FileState = "hashed"
	FileStateToHash     FileState = "to-hash"
	FileStateUploaded   FileState = "uploaded"
	FileStateToUpload   FileState = "to-upload"
	FileStateDownloaded FileState = "downloaded"
	FileStateToDownload FileState = "to-download"
	FileStateDeleted    FileState = "deleted"
	FileStateToDelete   FileState = "to-delete"
	FileStateToPull     FileState = "to-pull"
	FileStateToPush     FileState = "to-push"
)

type FileTreeJson struct {
	lock       sync.RWMutex
	Extensions map[string]bool
	FileMap    ftp_base.MutexedMap[*filehandler.FileHash] `json:"files"`

	FileState ftp_base.MutexedMap[FileState] `json:"file_state"`
}

func init() {

}

func InitialiseFileTree(file_tree_path string) {
	log.Println("loading filetree")

	FileTree.Lock()
	defer FileTree.Unlock()

	log.Println(file_tree_path)
	b, err1 := os.ReadFile(file_tree_path)
	if err1 != nil {
		if errors.Is(err1, os.ErrNotExist) {

			tmp, err2 := json.MarshalIndent(FileTree, " ", "\t")
			if err2 != nil {
				log.Fatalln(err2)
			}
			err3 := os.WriteFile(file_tree_path, tmp, fs.FileMode(ftp_base.S_IRWXU|ftp_base.S_IRWXO))
			if err2 != nil {
				log.Fatalln(err3)
			}

			log.Println("successfully loaded filetree")
			return
		}
		log.Fatalln(err1)
	}

	err3 := json.Unmarshal(b, FileTree)
	if err3 != nil {
		log.Fatalln(err3)
	}

	log.Println("successfully loaded filetree")
}

func NewFileTreeJson() *FileTreeJson {
	return &FileTreeJson{
		FileMap:    ftp_base.NewMutexedMap[*filehandler.FileHash](),
		Extensions: map[string]bool{},
		FileState:  ftp_base.NewMutexedMap[FileState](),
	}
}

func WriteFileTree(ctx ftp_context.Context, lock_file_p string, file_tree_path string) (err ftp_context.LogErr) {
	loc := logging.Loc("WriteFileTree() (err ftp_context.LogErr)")
	FileTree.RLock()
	defer FileTree.RUnlock()
	l, err1 := filehandler.Lock(lock_file_p)
	if err1 != nil {
		Logger.LogErr(loc, err1)
		<-time.After(time.Second * 5)
		return WriteFileTree(ctx, lock_file_p, file_tree_path)
	}
	defer l.Unlock()

	tmp, err1 := json.MarshalIndent(FileTree, " ", "\t")
	if err1 != nil {
		return &ftp_context.LogItem{Location: string(loc), Time: time.Now(),
			Err:       true,
			After:     `tmp, err1 := json.MarshalIndent(FileTree, " ", "\t")`,
			Message:   err1.Error(),
			CallStack: []error{err1},
		}
	}
	err2 := os.WriteFile(file_tree_path, tmp, fs.FileMode(ftp_base.S_IRWXU|ftp_base.S_IRWXO))
	if err2 != nil {
		return &ftp_context.LogItem{Location: string(loc), Time: time.Now(),
			Err:       true,
			After:     `err2 := os.WriteFile(file_tree_path, tmp, fs.FileMode(ftp_base.S_IRWXU|ftp_base.S_IRWXO))`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		}
	}

	return
}

func UpdateFileTree(ctx ftp_context.Context, lock_file_p string, file_tree_path string) {
	loc := logging.Loc("UpdateFileTree(ctx ftp_context.Context)")

	defer ctx.Finished()
	tc := time.NewTicker(time.Minute)
	for ok := true; ok; {
		select {
		case <-tc.C:
		case _, ok = <-ctx.Done():
		}

		err := WriteFileTree(ctx, lock_file_p, file_tree_path)
		if err != nil {
			Logger.LogErr(loc, err)
		}
		Logger.Logf(loc, "updated filetree successfully")
	}
}

func (ft *FileTreeJson) Lock() {
	ft.lock.Lock()

	ft.FileState.Lock()
	ft.FileMap.Lock()

}
func (ft *FileTreeJson) Unlock() {

	ft.FileState.Unlock()
	ft.FileMap.Unlock()
	ft.lock.Unlock()
}

func (ft *FileTreeJson) RLock() {
	ft.lock.RLock()

	ft.FileState.RLock()
	ft.FileMap.RLock()

}
func (ft *FileTreeJson) RUnlock() {

	ft.FileState.RUnlock()
	ft.FileMap.RUnlock()
	ft.lock.RUnlock()
}

func (ft *FileTreeJson) AddExtension(e string) {
	ft.Lock()
	ft.Extensions[e] = true
	ft.Unlock()
}
