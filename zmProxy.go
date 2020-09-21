package proxy

import (
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type ZmProxy struct {

}

//设置芝麻代理的白名单IP才可用
func (zm *ZmProxy) CreateProxies(num int, pool *Pool) ([]*Proxy, error) {
	reqUrl := fmt.Sprintf("http://webapi.http.zhimacangku.com/getip?num=%d&type=2&pro=0&city=0&yys=0&port=1&time=1&ts=1&ys=1&cs=1&lb=1&sb=0&pb=45&mr=1", num)
	//reqUrl := fmt.Sprintf("http://http.tiqu.alicdns.com/getip3?num=%d&type=2&pro=0&city=0&yys=0&port=1&time=1&ts=1&ys=1&cs=1&lb=1&sb=0&pb=45&mr=1&regions=110000,130000,140000,310000,320000,330000,350000,360000,370000,410000,420000,440000,500000,510000,530000&gm=4", num)
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
	proxyRet := make([]*Proxy, 0, num)
	now := time.Now()
	jsonRet.Get("data").ForEach(func(key, value gjson.Result) bool {
		log.Println(value.Get("expire_time").String())
		expire, err := time.ParseInLocation("2006-01-02 15:04:05", value.Get("expire_time").String(), time.Local)
		if err != nil {
			return true
		}
		proxyRet = append(proxyRet, &Proxy{
			pool:      pool,
			ipAddr:    value.Get("ip").String(),
			port:      value.Get("port").Uint(),
			createdAt: now,
			expire:    expire,
			isSSL:     false,
			isUse:     false,
			useNum:    0,
			isClosed:  false,
			city:      value.Get("city").String(),
			isp:       value.Get("isp").String(),
			delayReleaseNum:       0,
			timeoutCount:0,
		})
		return true
	})
	return proxyRet, nil
}
