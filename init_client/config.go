package initialiseclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"time"

	"os"

	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"
	"github.com/it-shiloheye/ftp_system_lib/base"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
)

var ClientConfig = &ClientConfigStruct{}
var Logger = logging.Logger

func (sc ClientConfigStruct) WriteConfig(file_path string) error {

	tmp, err1 := json.MarshalIndent(&sc, " ", "\t")
	if err1 != nil {
		return err1
	}

	err2 := os.WriteFile(file_path, tmp, fs.FileMode(base.S_IRWXU|base.S_IRWXO))
	if err2 != nil {
		return err2
	}
	return nil
}
func ReadConfig(file_path string) (sc ClientConfigStruct, err error) {

	b, err1 := os.ReadFile(file_path)
	if err1 != nil {
		return sc, err1
	}

	err2 := json.Unmarshal(b, &sc)
	if err2 != nil {
		err = err2
	}

	return
}

func init() {

}

func UpdateClientConfig(lock_file_p string, ctx ftp_context.Context) {
	loc := logging.Loc("UpdateConfig(ctx ftp_context.Context)")
	defer ctx.Finished()
	tc := time.NewTicker(time.Minute)
	for ok := true; ok; {
		select {
		case <-tc.C:
		case _, ok = <-ctx.Done():
		}

		err := WriteClientConfig(lock_file_p + "/config.lock")
		if err != nil {
			Logger.LogErr(loc, err)
		}
		Logger.Logf(loc, "updated config successfully")
	}
}

func InitialiseClientConfig() {
	log.Println("loading config")
	*ClientConfig = BlankClientConfigStruct()

	b, err1 := os.ReadFile("./config.json")
	if err1 != nil {
		if errors.Is(err1, os.ErrNotExist) {

			tmp, err2 := json.MarshalIndent(ClientConfig, " ", "\t")
			if err2 != nil {
				log.Fatalln(err2.Error())
			}
			err3 := os.WriteFile("./config.json", tmp, fs.FileMode(base.S_IRWXU|base.S_IRWXO))
			if err3 != nil {
				log.Fatalln(err3.Error())
			}
			log.Fatalln("fill in config")
			return
		}
		log.Fatalln(err1.Error())
	}

	err3 := json.Unmarshal(b, ClientConfig)
	if err3 != nil {
		log.Fatalln(err3.Error())
	}

	log.Println("successfull loaded config")
}

func WriteClientConfig(lock_file_p string, i ...int) (err ftp_context.LogErr) {
	loc := "WriteConfig() (err ftp_context.LogErr)"

	log.Println(lock_file_p)
	l, err3 := filehandler.Lock(lock_file_p)
	if err3 != nil {
		i_i := 0
		if len(i) > 0 {
			i_i = i[0]
		}
		if i_i < 5 {
			<-time.After(time.Second * 5)
			err = WriteClientConfig(lock_file_p, i_i+2)
			log.Println("try ", i_i, "to write config")
			return
		}
		err = &ftp_context.LogItem{
			Location: loc,
			Time:     time.Now(),
			Err:      true,
			After:    fmt.Sprintf(`l, err3  := filehandler.Lock(%s)`, lock_file_p),
			Message:  "not able to obtain lock",
		}
		return
	}
	defer l.Unlock()

	tmp, err1 := json.MarshalIndent(ClientConfig, " ", "\t")
	if err1 != nil {
		return &ftp_context.LogItem{Location: loc, Time: time.Now(),
			Err:       true,
			After:     `tmp, err1 := json.MarshalIndent(ClientConfig, " ", "\t")`,
			Message:   err1.Error(),
			CallStack: []error{err1},
		}
	}
	err2 := os.WriteFile("./config.json", tmp, fs.FileMode(base.S_IRWXU|base.S_IRWXO))
	if err2 != nil {
		return &ftp_context.LogItem{Location: loc, Time: time.Now(),
			Err:       true,
			After:     `err2 := os.WriteFile("./config.json", tmp, fs.FileMode(base.S_IRWXU|base.S_IRWXO))`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		}
	}

	return
}
