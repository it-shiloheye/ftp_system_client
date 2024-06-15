package netclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"net/http"
	"net/url"

	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"

	"github.com/it-shiloheye/ftp_system_lib/logging"
	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"
)

type NetworkEngine struct {
	sync.Mutex
	Client   *http.Client
	Map      ftp_base.MutexedMap[*http.Response]
	base_url string
	send_buf *bytes.Buffer
	recv_buf *bytes.Buffer
}

func (ne *NetworkEngine) BaseUrl() string {
	return ne.base_url
}

func (ne *NetworkEngine) Ping(ping_url string, tries int, v ...map[string]any) error {
	loc := log_item.Locf(`func (ne *NetworkEngine) Ping(ping_url: "%s", tries int, v ...map[string]any) error`, ne.base_url+ping_url)
	var tmp map[string]any

	if len(v) > 0 {
		tmp = v[0]
	} else {
		tmp = map[string]any{}
	}

	err := ne.GetJson(ping_url, &tmp)
	if err != nil && tries > 0 {
		<-time.After(time.Second)
		return ne.Ping(ping_url, tries, tmp)
	}

	return Logging.LogErr(loc, err)
}

func NewNetworkEngine(client *http.Client, base_url string) *NetworkEngine {
	return &NetworkEngine{
		Client:   client,
		base_url: base_url,
		Map:      ftp_base.NewMutexedMap[*http.Response](),
		send_buf: bytes.NewBuffer(make([]byte, 100_000)),
		recv_buf: bytes.NewBuffer(make([]byte, 100_000)),
	}

}

func (ne *NetworkEngine) PostBytes(route string, data []byte, out_json_item any) (err error) {
	loc := log_item.Locf(`func (ne *NetworkEngine) PostJson(route: "%s", in_json_item any, out_json_item any) (out []byte, err log_item.LogErr)`, route)
	ne.Lock()
	defer ne.Unlock()
	var err1, err2 error

	send_b := ne.send_buf
	send_b.Reset()

	str_1 := base64.StdEncoding.EncodeToString(data)
	send_b.WriteString(str_1)

	route = ne.BaseUrl() + route

	var res *http.Response
	res, err1 = ne.Client.Post(route, "application/octet-stream", send_b)
	if err1 != nil {
		return logging.Logger.LogErr(loc, err1)

	}
	defer res.Body.Close()

	ne.Map.Set(route, res)

	err2 = json.NewDecoder(res.Body).Decode(out_json_item)
	if err2 != nil {
		return logging.Logger.LogErr(loc, err2)
	}

	return
}

func (ne *NetworkEngine) PostJson(route string, in_json_item any, out_json_item any) (err error) {
	loc := log_item.Locf(`func (ne *NetworkEngine) PostJson(route: "%s", in_json_item any, out_json_item any) (out []byte, err log_item.LogErr)`, route)
	ne.Lock()
	defer ne.Unlock()
	var err1, err2, err3 error

	send_b := ne.send_buf
	send_b.Reset()

	err1 = json.NewEncoder(send_b).Encode(in_json_item)
	if err1 != nil {
		return logging.Logger.LogErr(loc, err1)

	}
	route = ne.BaseUrl() + route

	var res *http.Response
	res, err2 = ne.Client.Post(route, "application/javascript", send_b)
	if err2 != nil {
		return logging.Logger.LogErr(loc, err2)

	}
	defer res.Body.Close()

	err3 = json.NewDecoder(res.Body).Decode(out_json_item)
	if err3 != nil {
		return logging.Logger.LogErr(loc, err3)
	}

	ne.Map.Set(route, res)

	return
}

func (ne *NetworkEngine) GetJson(route string, out_json_item any) (err error) {
	loc := log_item.Locf(`func (ne *NetworkEngine) GetJson(route: "%s", out_json_item any) (out []byte, err log_item.LogErr)`, route)
	ne.Lock()
	defer ne.Unlock()
	var err1, err2 error
	var res *http.Response
	route = ne.BaseUrl() + route

	res, err1 = ne.Client.Get(route)
	if err1 != nil {
		return logging.Logger.LogErr(loc, err1)

	}
	defer res.Body.Close()

	err2 = json.NewDecoder(res.Body).Decode(out_json_item)
	if err2 != nil {
		return logging.Logger.LogErr(loc, err2)
	}

	return
}

func (ne *NetworkEngine) SetCookie(route string, cookie *http.Cookie) error {
	route = ne.base_url + route
	loc := log_item.Locf(`func (ne *NetworkEngine) SetCookie(route: "%s", cookie *http.Cookie) error`, route)
	if ne.Client.Jar == nil {
		ne.Client.Jar = NewCookieJar()
	}

	url_, err1 := url.Parse(route)
	if err1 != nil {
		return Logging.LogErr(loc, err1)
	}
	rl_cookies := ne.Client.Jar.Cookies(url_)

	if len(rl_cookies) < 0 {
		ne.Client.Jar.SetCookies(url_, []*http.Cookie{cookie})
		return nil
	}
	uniq := map[string]*http.Cookie{}

	for _, ck := range rl_cookies {
		uniq[ck.Name] = ck
	}

	uniq[cookie.Name] = cookie

	total_ := []*http.Cookie{}
	for _, ck := range uniq {
		total_ = append(total_, ck)
	}

	ne.Client.Jar.SetCookies(url_, total_)

	return nil
}
