package netclient

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"sync"
	"time"

	"net/http"
	"net/http/cookiejar"
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

func (ne *NetworkEngine) Ping(ping_url string, tries int, v ...*map[string]any) error {
	loc := log_item.Locf(`func (ne *NetworkEngine) Ping(ping_url: "%s", tries: %02d, v ...map[string]any) error`, ne.base_url+ping_url, tries)
	var tmp *map[string]any

	if len(v) > 0 {
		tmp = v[0]
	} else {
		tmp = &map[string]any{}
	}

	logging.Logger.Logf(loc, "attempt: %03d", tries)
	err := ne.GetJson(ping_url, tmp)
	if err != nil && tries > 0 {
		<-time.After(time.Second * 5)
		return ne.Ping(ping_url, tries-1, tmp)
	}

	return nil
}

func NewNetworkEngine(client *http.Client, base_url string) *NetworkEngine {
	if client.Jar == nil {
		client.Jar, _ = cookiejar.New(&cookiejar.Options{})
	}

	return &NetworkEngine{
		Client:   client,
		base_url: base_url,
		Map:      ftp_base.NewMutexedMap[*http.Response](),
		send_buf: bytes.NewBuffer(make([]byte, 100_000)),
		recv_buf: bytes.NewBuffer(make([]byte, 100_000)),
	}

}

func (ne *NetworkEngine) PostBytes(route string, data []byte, out_json_item any) (err error) {

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
		return err1
	}
	defer res.Body.Close()
	ne.SetCookie(route, res.Cookies()...)
	ne.Map.Set(route, res)

	err2 = json.NewDecoder(res.Body).Decode(out_json_item)
	if err2 != nil {
		return err2
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
	ne.SetCookie(route, res.Cookies()...)

	err3 = json.NewDecoder(res.Body).Decode(out_json_item)
	if err3 != nil {
		log.Println(res)
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
	ne.SetCookie(route, res.Cookies()...)
	err2 = json.NewDecoder(res.Body).Decode(out_json_item)
	if err2 != nil {
		return logging.Logger.LogErr(loc, err2)
	}

	return
}

func (ne *NetworkEngine) GetCookie(route string, cookie_name string) (cookie *http.Cookie, err error) {
	route = ne.base_url + route

	url_, err1 := url.ParseRequestURI(route)
	if err1 != nil {
		return nil, err1
	}

	rl_cookies := ne.Client.Jar.Cookies(url_)

	for _, ck := range rl_cookies {
		if ck.Name == cookie_name {
			return ck, nil
		}
	}

	return nil, &log_item.LogItem{
		Time:    time.Now(),
		Message: "missing cookie",
	}
}

func (ne *NetworkEngine) SetCookie(route string, cookies ...*http.Cookie) error {
	route = ne.base_url + route
	loc := log_item.Locf(`func (ne *NetworkEngine) SetCookie(route: "%s", cookie *http.Cookie) error`, route)

	uniq := map[string]*http.Cookie{}

	url_, err1 := url.Parse(route)
	if err1 != nil {
		return Logging.LogErr(loc, err1)
	}
	rl_cookies := ne.Client.Jar.Cookies(url_)

	for _, ck := range rl_cookies {
		uniq[ck.Name] = ck
	}

	for _, cookie := range cookies {

		cookie.Path = route
		uniq[cookie.Name] = cookie
	}

	total_ := []*http.Cookie{}
	for _, ck := range uniq {
		total_ = append(total_, ck)
	}

	ne.Client.Jar.SetCookies(url_, total_)

	return nil
}

func (ne *NetworkEngine) GetCookies(route string) (cookies []*http.Cookie, err error) {

	url_, err1 := url.Parse(ne.base_url + route)
	if err1 != nil {
		return nil, err1
	}
	cookies = ne.Client.Jar.Cookies(url_)
	return
}
