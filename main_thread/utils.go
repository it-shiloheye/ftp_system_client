package mainthread

import (
	"encoding/base64"
	"strings"

	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	dir_handler "github.com/it-shiloheye/ftp_system_client/main_thread/dir_handler"
	netclient "github.com/it-shiloheye/ftp_system_client/main_thread/network_client"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"

	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
)

func HashingFunction(bs *filehandler.BytesStore, file_p string) error {
	loc := log_item.Locf(`func HashingFunction(bs *filehandler.BytesStore, file_p: %s) error`, file_p)
	var err1, err2, err3 error
	fh, _ := FileTree.FileMap.Get(file_p)
	Logger.Logf(loc, "to hash: %s\nModTime: %s", file_p, fh.ModTime)
	bs.Reset()

	if fh.File == nil {
		fh.File, err1 = ftp_base.OpenFile(file_p, os.O_RDONLY)
		if err1 != nil {

			return Logger.LogErr(loc, &log_item.LogItem{
				Location:  loc,
				After:     fmt.Sprintf(`fh.File, err1 = ftp_base.OpenFile("%s",os.O_RDONLY)`, file_p),
				Message:   err1.Error(),
				CallStack: []error{err1},
			})
		}
	}

	fh.Size, err2 = bs.ReadFrom(fh.File)
	if err2 != nil {

		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			After:     `_, err1 = bs.ReadFrom(fh.File)`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		})
	}

	fh.Hash, err3 = bs.Hash()
	if err3 != nil {
		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			After:     `fh.Hash, err2 = bs.Hash()`,
			Message:   err3.Error(),
			CallStack: []error{err3},
		})

	}

	FileTree.FileState.Set(file_p, dir_handler.FileStateToUpload)
	Logger.Logf(loc, "done hashing: %s\nat: %s\nhash:\t%s", file_p, fmt.Sprint(time.Now()), fh.Hash)
	return nil
}

func UploadFunction(ctx ftp_context.Context, network_e *netclient.NetworkEngine, FileTree *dir_handler.FileTreeJson) error {
	loc := log_item.Locf(`UploadFunction(network_e *netclient.NetworkEngine, FileTree *dir_handler.FileTreeJson) error`)
	defer ctx.Finished()
	defer log.Println("exiting upload function")

	file_state := map[string]dir_handler.FileState{}
	ok := true
	tmp_state := dir_handler.FileStateMissing
	for _, file_hash := range FileTree.FileState.Keys() {
		tmp_state, ok = FileTree.FileState.Get(file_hash)
		if !ok {
			Logger.LogErr(loc, &log_item.LogItem{
				Message: fmt.Sprintf("filehash: %s missing from state", file_hash),
			})
			continue
		}
		switch tmp_state {
		case dir_handler.FileStateToUpload:
			fallthrough
		case dir_handler.FileStateHashed:
			file_state[file_hash] = dir_handler.FileStateToUpload
		}

		log.Println("file_hash: ", file_hash)
	}

	tmp_filetree, _ := FileTree.FileMap.Copy()

	log.Println(loc, "\ntmp_filetree")
	err_c := make(chan error)
	done_c := make(chan any)

	upload_state := map[string]any{}
	uploaded_hashes := []string{}

	uploader := func(ctx ftp_context.Context) {
		defer ctx.Finished()
		log.Println(loc, "\nuploader")
		expires := time.Now().Add(time.Second)
		client_id := &http.Cookie{
			Name:    "client-id",
			Value:   ClientConfig.ClientId,
			Expires: expires,
		}
		dir_id := &http.Cookie{
			Name:    "dir-id",
			Value:   ClientConfig.DirConfig.DirId,
			Expires: expires,
		}
		encode := base64.StdEncoding.EncodeToString
		clear(uploaded_hashes)

		for file_path, fh := range tmp_filetree {

			_, ok := file_state[fh.Hash]
			if !ok {
				continue
			}

			d, err1 := os.ReadFile(file_path)
			if err1 != nil {
				err_c <- Logger.LogErr(loc, err1)
				return
			}

			upload_state[fh.Hash] = encode(d)
			if len(uploaded_hashes) > 9 {
				break
			}
			uploaded_hashes = append(uploaded_hashes, fh.Hash)
		}
		log.Println("posting:\n", file_state)
		response := map[string]any{}
		network_e.SetCookie("/", client_id, dir_id)
		err2 := network_e.PostJson("/upload/bulk", upload_state, &response)
		if err2 != nil {
			err_c <- err2
			return
		}

		for _, hash := range uploaded_hashes {
			delete(file_state, hash)
		}
		if len(uploaded_hashes) > 1 {
			Logger.Logf(loc, "uploaded:\n%s", strings.Join(uploaded_hashes, "\n"))
		}
		done_c <- &struct{}{}
	}

	for len(file_state) > 0 {
		go uploader(ctx.Add())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-err_c:
			return Logger.LogErr(loc, err)
		case <-done_c:

		}
	}
	Logger.Logf(loc, "successfully posted files to server")
	return nil
}

func ConfirmFunction(network_e *netclient.NetworkEngine, fstate_tree *dir_handler.FileTreeJson) (to_upload []string, err error) {
	loc := log_item.Locf(`ConfirmFunction(network_e *netclient.NetworkEngine, fstate_tree *dir_handler.FileState) error`)
	Logger.Logf(loc, "posting filetree_state to server: %s", ClientConfig.ClientId)

	log.Println(loc, "\nfiletree:\n", fstate_tree)

	route := "/upload/confirm"
	client_id := &http.Cookie{
		Name:    "client-id",
		Value:   ClientConfig.ClientId,
		Expires: time.Now().Add(time.Second),
	}
	dir_id := &http.Cookie{
		Name:    "dir-id",
		Value:   ClientConfig.DirConfig.DirId,
		Expires: time.Now().Add(time.Second),
	}
	network_e.SetCookie("/", client_id, dir_id)

	tmp := map[string]string{}
	err1 := network_e.PostJson(route, fstate_tree, &tmp)
	if err1 != nil {

		return nil, Logger.LogErr(loc, err1)
	}

	state, ok := tmp["state"]
	if !ok || state != "success" {
		log.Printf("response from server:\n\n%v\n\n", tmp)
		return nil, fmt.Errorf("invalid response")
	}

	for k, v := range tmp {
		if k == "state" {
			continue
		}
		log.Println(k, ":", v)
		switch v {
		case "missing":
			to_upload = append(to_upload, k)
			fstate_tree.FileState.Set(k, dir_handler.FileStateToUpload)
		}

	}

	Logger.Logf(loc, "successfully posted filetree_state to server")
	return
}
