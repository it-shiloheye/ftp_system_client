package dir_handler

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"sync"
	"time"

	"os"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"

	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
	// "golang.org/x/sync/syncmap"
)

var FileTree = NewFileTreeJson()

type FileState string

const (
	FileStateMissing    FileState = "missing"
	FileStateToRead     FileState = "to-read"
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
	// uses file_path as key on client because file path is unique
	FileMap ftp_base.MutexedMap[*filehandler.FileHash] `json:"files"`

	//  uses file_hash as key because hash is unique both client and server-side
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

func WriteFileTree(ctx ftp_context.Context, file_tree_path string) (err error) {
	loc := log_item.Loc("WriteFileTree() (err log_item.LogErr)")
	lock_file_p := file_tree_path + "/file-tree.lock"
	log.Println(lock_file_p)

	l, err1 := filehandler.Lock(lock_file_p)
	for i := 0; ; i += 1 {
		if err1 != nil {
			select {
			case <-ctx.Done():
				return Logger.LogErr(loc, err1)
			case <-time.After(time.Second * 5):
				if i >= 5 {
					return Logger.LogErr(loc, err1)
				}
			}
			l, err1 = filehandler.Lock(file_tree_path + "/file-tree.lock")
			continue
		}
		break
	}
	defer l.Unlock()

	if FileTree == nil {
		log.Fatalln("missing file-tree")
	}
	FileTree.Lock()
	tmp, err1 := json.MarshalIndent(FileTree, " ", "\t")
	FileTree.Unlock()
	if err1 != nil {
		return Logger.LogErr(loc, err1)
	}
	err2 := os.WriteFile(file_tree_path+"/file-tree.json", tmp, fs.FileMode(ftp_base.S_IRWXU|ftp_base.S_IRWXO))
	if err2 != nil {
		return Logger.LogErr(loc, err2)
	}

	Logger.Logf(loc, "updated file-tree successfully")
	return
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
