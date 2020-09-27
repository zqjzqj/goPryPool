package goPryPool

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HdProxy struct {
	mu sync.Mutex
}

//设置芝麻代理的白名单IP才可用
func (hd *HdProxy) CreateProxies(num int, pool *Pool) ([]*Proxy, error) {
	hd.mu.Lock()
	defer hd.mu.Unlock()
	proxyRet := make([]*Proxy, 0, num)
	num2 := num
	log.Println("获取代理数：", num2, len(proxyRet))
	RE:
	reqUrl := fmt.Sprintf("http://ip.ipjldl.com/index.php/api/entry?method=proxyServer.hdtiqu_api_url&packid=0&fa=0&groupid=0&fetch_key=&qty=%d&time=1&port=1&format=json&ss=5&css=&ipport=1&et=1&pi=1&co=1&pro=&city=&dt=1&usertype=4", num2)
	resp, err := http.Get(reqUrl)
	if err != nil {
		return nil, err
	}
	body := resp.Body
	defer body.Close()
	ret, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	jsonRet := gjson.ParseBytes(ret)
	if jsonRet.Get("code").Int() != 0 {
		return nil, errors.New(jsonRet.Get("msg").String())
	}
	now := time.Now()
	now5m := now.Add(time.Minute * 5)
	jsonRet.Get("data").ForEach(func(key, value gjson.Result) bool {
		expire, err := time.ParseInLocation("2006-01-02 15:04:05", value.Get("ExpireTime").String(), time.Local)
		if err != nil {
			return true
		}
		//处理一下过期时间 最多5分钟
		if expire.After(now5m) {
			log.Println("跳过代理：", value.Get("IpAddress"), value.Get("ExpireTime").String())
			return true
		}
		ipAndPort := strings.Split(value.Get("IP").String(), ":")
		if len(ipAndPort) != 2 {
			return true
		}
		ip := ipAndPort[0]
		port, _ := strconv.ParseUint(ipAndPort[1], 10, 64)
		proxyRet = append(proxyRet, &Proxy{
			pool:      pool,
			ipAddr:    ip,
			port:      port,
			createdAt: now,
			expire:    expire,
			isSSL:     false,
			isUse:     false,
			useNum:    0,
			isClosed:  false,
			city:      value.Get("IpAddress").String(),
			isp:       value.Get("ISP").String(),
			delayReleaseNum:       0,
			timeoutCount:0,
			expireAdvance:12 * time.Second,
		})
		return true
	})
	if len(proxyRet) < num {
		num2 -= len(proxyRet)
		time.Sleep(2 * time.Second)
		goto RE
	}
//	os.Exit(0)
	return proxyRet, nil
}