package dir_handler

import (
	"fmt"
	"io/fs"

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

type ReadDirResult struct {
	FilesList []*filehandler.FileBasic
	ToRehash  []string
	ToUpload  []string
}

func ticker(loc log_item.Loc, i int) {

	// Logger.Logf(loc, "%d", i)
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

func ReadDir(ctx ftp_context.Context, dir_data initialiseclient.DirConfig) (err error) {
	loc := log_item.Loc("ReadDir(ctx ftp_context.Context, dir_data initialiseclient.DirConfig) (err log_item.LogErr)")
	defer ctx.Finished()
	ticker(loc, 1)

	ticker(loc, 2)
	dirs_excluded_dirs_list := GetExcludeList(dir_data.ExcludeDirs, dir_data.ExcludeRegex, dir_data.ExcluedFile)

	ticker(loc, 3)

	var out []TempFileData

	err1 := filepath.WalkDir(dir_data.Path, func(path string, fs_d fs.DirEntry, err2 error) error {
		loc := log_item.Locf(`filepath.WalkDir("%s", func("%s", _ fs.DirEntry, err2 error) error `, dir_data.Path, path)

		if err2 != nil {

			return Logger.LogErr(loc, err2)
		}

		for _, excluded := range dirs_excluded_dirs_list {
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

	FS := os.DirFS(dir_data.Path)

	for _, file := range out {
		f_path := file.path
		if file.fs.IsDir() {
			continue
		}
		fs_, err3 := fs.Stat(FS, file.fs.Name())

		fh_old, exists_filemap := FileTree.FileMap.Get(f_path)
		if !exists_filemap {
			if err3 != nil {
				Logger.LogErr(loc, err3)
				continue
			}

			fh := &filehandler.FileHash{
				FileBasic: &filehandler.FileBasic{
					Path: f_path,
				},
				ModTime: fmt.Sprint(fs_.ModTime()),
			}
			fh.FileType = filehandler.Ext(fh.FileBasic)
			FileTree.FileMap.Set(f_path, fh)
			FileTree.FileState.Set(f_path, FileStateToHash)
			FileTree.AddExtension(string(fh.FileType))
			continue
		}

		if fh_old.ModTime != fmt.Sprint(fs_.ModTime()) {
			FileTree.FileState.Set(f_path, FileStateToHash)

		}
	}

	ticker(loc, 6)
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

func list_file_tree(dir_path string, exclude_paths []string) (out []*filehandler.FileBasic, err error) {

	return
}
