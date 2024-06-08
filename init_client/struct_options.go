package initialiseclient

import (
	"os"
	"time"

	"github.com/google/uuid"
)

type ClientConfigStruct struct {
	Schema   string `json:"$schema"`
	SchemaId string `json:"$id"`

	// local client ip address
	LocalIp string `json:"local_ip"`
	// web ip address exposed to internet
	WebIp string `json:"web_ip"`
	// where root CA is stored
	CA_Location string `json:"ca_location"`
	// unique id of current client
	ClientId string `json:"client_id"`
	// common name of current client
	CommonName string `json:"common_name"`
	// back up instructions of current client
	DirConfig `json:"all_dirs"`
	// download instructions of current client
	SubscribeDirs []SubscribeDirsStruct `json:"subscribe_dirs"`
	// ips of servers to subscribe and connect to
	ServerIps []string `json:"server_ips"`
	// local directory to store logs
	DataDir string `json:"data_dir"`
	// crash if error, or continue in error state
	StopOnError bool `json:"stop_on_error"`
}

type DirConfig struct {
	Id            string        `json:"id"`
	Path          string        `json:"path"`
	ExcludeDirs   []string      `json:"exclude_dir"`
	ExcluedFile   []string      `json:"exclude_file"`
	ExcludeRegex  []string      `json:"exclude_regex"`
	FollowSymlink bool          `json:"follow_symlink"`
	IncludeDir    []string      `json:"include_dir"`
	IncludeExt    []string      `json:"include_ext"`
	IncludeFile   []string      `json:"include_file"`
	UpdateRate    time.Duration `json:"update_rate_minutes"`
	PathSeparator string        `json:"path_separator"`
}

type SubscribeDirsStruct struct {
	ClientId             string        `json:"client_id"`
	DirsId               []SubcribeDir `json:"dirs_id"`
	PullChanges          bool          `json:"pull_changes"`
	PushChanges          bool          `json:"push_changes"`
	PushChangesFrequency int           `json:"push_changes_frequency"`
}

type SubcribeDir struct {
	DirsId    string `json:"dir_id"`
	LocalPath string `json:"local_path"`
}

func BlankDirConfig() DirConfig {
	return DirConfig{
		Id:            uuid.New().String(),
		Path:          "",
		ExcludeDirs:   []string{},
		ExcluedFile:   []string{},
		ExcludeRegex:  []string{},
		FollowSymlink: false,
		IncludeDir:    []string{},
		IncludeExt:    []string{},
		IncludeFile:   []string{},
	}
}

func BlankSubscribeDirsStruct() SubscribeDirsStruct {
	return SubscribeDirsStruct{
		ClientId:             "",
		DirsId:               []SubcribeDir{},
		PullChanges:          true,
		PushChanges:          false,
		PushChangesFrequency: 0,
	}
}

func BlankClientConfigStruct() ClientConfigStruct {
	return ClientConfigStruct{
		Schema:   "https://json-schema.org/draft/2020-12/schema",
		SchemaId: "",

		// local net client ip address
		LocalIp: "",
		// web ip address exposed to internet
		WebIp: "",
		// where root CA is stored
		CA_Location: "./data/ssl_certs",
		// unique id of current client
		ClientId: uuid.NewString(),
		// common name of current client
		CommonName: "",
		// back up instructions of current client
		DirConfig: DirConfig{
			Id:            uuid.NewString(),
			Path:          "",
			ExcludeDirs:   []string{".git", "node_modules", "vendor", "tmp", ".next"},
			ExcluedFile:   []string{"~"},
			ExcludeRegex:  []string{},
			FollowSymlink: false,
			IncludeDir:    []string{},
			IncludeExt:    []string{},
			IncludeFile:   []string{},
			UpdateRate:    time.Minute,
			PathSeparator: string(os.PathSeparator),
		},
		// download instructions of current client
		SubscribeDirs: []SubscribeDirsStruct{BlankSubscribeDirsStruct()},
		// ips of servers to subscribe and connect to
		ServerIps: []string{},
		// local directory to store logs
		DataDir: "./data",
		// crash if error, or continue in error state
		StopOnError: true,
	}
}
