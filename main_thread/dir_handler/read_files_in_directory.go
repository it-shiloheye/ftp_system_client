package dir_handler

import (
	"fmt"
	"io/fs"
	"log"
	"regexp"

	"os"

	"path/filepath"
	"strings"

	"time"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
	"github.com/it-shiloheye/ftp_system_lib/logging"
	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"
	// filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
)

var ClientConfig = initialiseclient.ClientConfig
var Logger = logging.Logger

func ticker(loc log_item.Loc, i ...int) {
	str := ""
	for _i, it := range i {
		if _i == 0 {
			str += fmt.Sprintf("\t%d", it)
			continue
		}
		str += fmt.Sprintf(",\t%d", it)
	}

	Logger.Logf(loc, "loc:%s", str)
}

func GetExcludeList(excluded ...[]string) (tmp []string) {
	tmp_ := append(append(ClientConfig.ExcludeDirs, ".git"), ClientConfig.ExcludeDirs...)
	excluded_dirs_ := FlatMap(append(excluded, tmp_)...)
	dir_uniq := map[string]bool{}
	for _, d := range excluded_dirs_ {
		if len(d) < 1 || dir_uniq[d] {
			continue
		}
		if !strings.Contains(d, "\\") && !strings.Contains(d, "/") {
			tmp = append(tmp, d)
			dir_uniq[d] = true
			continue
		}
		a := strings.Join(strings.Split(d, string(os.PathSeparator)), "/")
		b := strings.Join(strings.Split(d, string(os.PathSeparator)), "\\")
		tmp = append(tmp, a, b)
		dir_uniq[a] = true
		dir_uniq[b] = true
	}

	return
}

type TempFileData struct {
	path string
	fs   fs.DirEntry
}

/*
ReadDir has 3 functions:
1. Read all file names from the dir_data path
2. Hash Files which are missing from the tree-json
3. Mark files for uploading
4. Should also mark files for downloading
*/
func DirHandler(ctx ftp_context.Context, dir_data_path string, exclude_list []string) (err error) {
	loc := log_item.Locf("ReadDir(ctx ftp_context.Context, dir_data: %s) (err log_item.LogErr)", dir_data_path)
	defer ctx.Finished()

	out := []TempFileData{}

	// recursively walk dir, and select the files valid
	err1 := filepath.WalkDir(dir_data_path, func(path string, fs_d fs.DirEntry, err2 error) error {
		loc := log_item.Locf(`filepath.WalkDir("%s", func("%s", _ fs.DirEntry, err2 error) error `, dir_data_path, path)

		if err2 != nil {

			return Logger.LogErr(loc, err2)
		}

		for _, excluded := range exclude_list {
			if not_ok, _ := regexp.MatchString(excluded, path); strings.Contains(path, excluded) || not_ok {
				if fs_d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}

		}

		if fs_d.IsDir() {
			return nil
		}

		out = append(out, TempFileData{
			path: path,
			fs:   fs_d,
		})

		return nil
	})

	if err1 != nil {

		return Logger.LogErr(loc, err1)

	}

	bs := filehandler.NewBytesStore()

	for _, file := range out {

		f_path := file.path
		// if file.fs.IsDir() { // unnecessary function call, already rooted out directories
		// 	continue
		// }
		fh, err1 := filehandler.NewFileHashOpen(f_path)
		if err1 != nil {
			log.Println("something went wrong opening filehash")
			return Logger.LogErr(loc, err1)
		}
		FileTree.FileMap.Set(f_path, fh)
		defer fh.Close()
		fh_old, exists_filemap := FileTree.FileMap.Get(f_path)

		if !exists_filemap || len(fh_old.Hash) < 1 {
			FileTree.AddExtension(string(fh.FileType))
			g_hash, err2 := HashingFunction(bs, f_path, fh)
			if err2 != nil {
				return Logger.LogErr(loc, err)
			}
			FileTree.FileState.Set(g_hash, FileStateToUpload)
			continue
		}

		if fh_old.ModTime != fh.ModTime {
			g_hash, err2 := HashingFunction(bs, f_path, fh)
			if err2 != nil {
				return Logger.LogErr(loc, err)
			}
			FileTree.FileState.Set(g_hash, FileStateToUpload)

		}
	}

	Logger.Logf(loc, "successfully read dir at %s", time.Now().Format(time.RFC822))
	return
}

func FlatMap[T any](lists ...[]T) (res []T) {
	l := 0
	for _, listlet := range lists {
		l += len(listlet)
	}
	res = make([]T, l)
	for _, listlet := range lists {
		res = append(res, listlet...)
	}

	return
}

func NilError(err error) bool {
	if err != nil {
		if len(err.Error()) > 0 {
			return true
		}
	}

	return false
}

// check if file exists, if file doesn't exist, opens a filehash object,
// then hashes the file, updates FileTree with hashed object, and returns generated_hash
func HashingFunction(bs *filehandler.BytesStore, file_p string, fh *filehandler.FileHash) (generated_hash string, err error) {
	loc := log_item.Locf(`func HashingFunction(bs *filehandler.BytesStore, file_p: %s) error`, file_p)
	var err1, err2 error

	bs.Reset()

	Logger.Logf(loc, "to hash: %s\nModTime: %s", file_p, fh.ModTime)

	fh.Size, err1 = bs.ReadFrom(fh.File)
	if err1 != nil {

		return "", Logger.LogErr(loc, err1)
	}

	fh.Hash, err2 = bs.Hash()
	if err2 != nil {
		return fh.Hash, Logger.LogErr(loc, err2)

	}

	Logger.Logf(loc, "done hashing: %s\nat: %s\nhash:\t%s", file_p, fmt.Sprint(time.Now()), fh.Hash)
	return fh.Hash, nil
}
