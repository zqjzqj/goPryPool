package goPryPool

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"
)

var DefaultExpireAdvance time.Duration = 0
var DefaultExpireAdvanceAsNotUse time.Duration = 0

type Proxy struct {
	pool *Pool
	ipAddr string
	port uint64
	createdAt time.Time
	expire time.Time
	isSSL bool
	IsSocks5 bool
	isUse bool
	useNum uint
	useNumTotal uint
	mu sync.Mutex
	isClosed bool
	city string
	isp string

	//延迟释放
	delayReleaseNum int //大于0时 不会立即释放 必须在释放数达到这个值时才会真正释放 小于0不释放
	delayReleaseRNum int

	timeoutCount uint8

	expireAdvance time.Duration
	expireAdvanceAsNotUse time.Duration
	isListenExpired bool
	err error
	expiredCh chan struct{}
	cancelListenExpired chan struct{}
	MaxUseNum uint
}

func NewErrProxy(err error) *Proxy {
	return &Proxy{
		pool:             nil,
		ipAddr:           "",
		port:             0,
		createdAt:        time.Time{},
		expire:           time.Time{},
		isSSL:            false,
		isUse:            false,
		useNum:           0,
		useNumTotal:      0,
		mu:               sync.Mutex{},
		isClosed:         false,
		city:             "",
		isp:              "",
		delayReleaseNum:  0,
		delayReleaseRNum: 0,
		timeoutCount:     0,
		expireAdvance:    0,
		err:              err,
		isListenExpired: false,
	}
}

func NewProxy(pool *Pool, ip string, port uint64, expire time.Time, isSSl bool, city string, isp string) *Proxy {
	return &Proxy{
		pool:             pool,
		ipAddr:           ip,
		port:             port,
		createdAt:        time.Time{},
		expire:           expire,
		isSSL:            isSSl,
		isUse:            false,
		useNum:           0,
		useNumTotal:      0,
		mu:               sync.Mutex{},
		isClosed:         false,
		city:             city,
		isp:              isp,
		delayReleaseNum:  0,
		delayReleaseRNum: 0,
		timeoutCount:0,
		expireAdvance: DefaultExpireAdvance,
		expireAdvanceAsNotUse:DefaultExpireAdvanceAsNotUse,
		isListenExpired: false,
		MaxUseNum: 0,
	}
}

func (pry Proxy) GetPoolDriverName() string {
	if pry.pool == nil {
		return ""
	}
	ty := reflect.TypeOf(pry.pool.driver)
	return ty.Elem().Name()
}

func (pry Proxy) IsListenExpired() bool {
	return pry.isListenExpired
}

func (pry *Proxy) SetExpiredAdvance(t time.Duration) {
	pry.expireAdvance = t
}

func (pry *Proxy) SetExpiredAdvanceAsNotUse(t time.Duration) {
	pry.expireAdvanceAsNotUse = t
}

func (pry *Proxy) AddTimeoutCount() {
	pry.mu.Lock()
	pry.timeoutCount++
	pry.mu.Unlock()
}

func (pry *Proxy) GetTimeoutCount() uint8 {
	return pry.timeoutCount
}

func (pry *Proxy) SetDelayReleaseNum(num int) {
	pry.mu.Lock()
	defer pry.mu.Unlock()
	pry.delayReleaseNum = num
}

func (pry *Proxy) SetUse() {
	pry.mu.Lock()
	defer pry.mu.Unlock()
	pry.useNum += 1
	pry.useNumTotal += 1
	pry.isUse = true
	pry.createListenAutoExpireLocked()
}

func (pry *Proxy) CreateListenAutoExpire() {
	pry.mu.Lock()
	defer pry.mu.Unlock()
	pry.createListenAutoExpireLocked()
}

func (pry *Proxy) GetExpiredCh() <-chan struct{} {
	return pry.expiredCh
}

func (pry *Proxy) cancelListenAutoExpireLocked() {
	if pry.isListenExpired {
		pry.cancelListenExpired <- struct{}{}
	}
	return
}

func (pry *Proxy) CancelListenAutoExpire() {
	pry.mu.Lock()
	defer pry.mu.Unlock()
	pry.cancelListenAutoExpireLocked()
	return
}

func (pry *Proxy) createListenAutoExpireLocked() {
	if pry.pool.IsAutoCloseExpiredPry && !pry.isListenExpired {
		pry.expiredCh = make(chan struct{}, 1)
		pry.cancelListenExpired = make(chan struct{}, 1)
		pry.isListenExpired = true
		go func() {
			t := time.NewTicker(2 * time.Second)
			defer func() {
				pry.mu.Lock()
				pry.isListenExpired = false
				t := time.NewTimer(time.Second * 5)
				select {
				case <-t.C:
				case pry.expiredCh <- struct{}{}:
				}
				close(pry.expiredCh)
				close(pry.cancelListenExpired)
				pry.mu.Unlock()
				t.Stop()
			}()
			for {
				select {
				case <-pry.pool.ctx.Done():
					pry.Close()
					return
				case <-t.C:
					if pry.IsExpiredAndClose() {
						return
					}
				case <-pry.cancelListenExpired:
					return
				}
			}
		}()
	}
}

func (pry *Proxy) GetProxyIpAddr() string {
	return fmt.Sprintf("%s:%d", pry.ipAddr, pry.port)
}

func (pry Proxy) GetIpAddr() string {
	return pry.ipAddr
}

func (pry Proxy) GetPort() uint64 {
	return pry.port
}

func (pry *Proxy) GetProxyUrl() string {
	if pry.IsSocks5 {
		return fmt.Sprintf("socks5://%s:%d", pry.ipAddr, pry.port)
	}
	if !pry.isSSL {
		return fmt.Sprintf("http://%s:%d", pry.ipAddr, pry.port)
	}
	return fmt.Sprintf("https://%s:%d", pry.ipAddr, pry.port)
}

func (pry Proxy) GetCity() string {
	return pry.city
}

func (pry Proxy) GetIsp() string {
	return pry.isp
}

//当前时间+30s 超过这个时间就算过期
//因为如果只差几秒过期的代理拿出来可能会运行不完一套程序
func (pry *Proxy) IsExpired() bool {
	return pry.expire.Before(time.Now().Add(pry.expireAdvance))
}

func (pry *Proxy) IsExpiredAndClose() bool {
	ret := pry.IsExpired()
	if ret {
		pry.Close()
	}
	return ret
}

func (pry *Proxy) GetExpire() time.Time {
	return pry.expire
}

func (pry *Proxy) IsClosed() bool {
	return pry.isClosed
}

func (pry *Proxy) IsUse() bool {
	return pry.isUse
}

func (pry *Proxy) GetUseNum() uint {
	return pry.useNum
}

func (pry *Proxy) GetTotalUseNum() uint {
	return pry.useNumTotal
}

func (pry *Proxy) Close() {
	pry.mu.Lock()
	defer pry.mu.Unlock()
	if pry.isClosed {
		return
	}

	pry.pool.mu.Lock()
	pry.pool.openNum--
	if pry.pool.openNum == 0 && len(pry.pool.pryRequests) > 0 {
		pry.pool.mu.Unlock()
		pry.pool.CreateNewProxies()
	} else {
		pry.pool.mu.Unlock()
	}
	pry.isUse = false
	pry.useNum = 0
	pry.isClosed = true
	pry.cancelListenAutoExpireLocked()
}

//释放一个代理  在代理用完之后调用此方法
//否则将会直接丢弃 会导致代理的重用率下降
func (pry *Proxy) Release() bool {
	if pry.isClosed {
		log.Println("Release： already closed....")
		return true
	}
	if pry.IsExpiredAndClose() {
		return true
	}

	if pry.MaxUseNum > 0 && pry.useNumTotal <= pry.MaxUseNum {
		pry.Close()
		return true
	}
	pry.mu.Lock()
	if pry.isClosed {
		log.Println("Release： already closed....")
		pry.mu.Unlock()
		return true
	}

	if !pry.isUse {
		log.Println("Release： already Release....")
		pry.mu.Unlock()
		return true
	}

	//小于零表示不释放
	if pry.delayReleaseNum < 0 {
		log.Println("Release： delayReleaseNum < 0 not Release")
		pry.mu.Unlock()
		return true
	}

	//处理延迟释放
	if pry.delayReleaseNum > 0 {
		pry.delayReleaseRNum++
		if pry.delayReleaseNum > pry.delayReleaseRNum {
			log.Println("Release：delayRelease....", pry.delayReleaseRNum, " max : ", pry.delayReleaseNum)
			pry.mu.Unlock()
			return true
		}
	}
	pry.isUse = false
	pry.useNum = 0
	pry.delayReleaseRNum = 0
	pry.delayReleaseNum = 0
	pry.mu.Unlock()

	pry.pool.mu.Lock()
	ret := pry.pool.putPryLocked(pry)
	pry.pool.mu.Unlock()
	if !ret {
		return false
	}

	return true
}

type Proxies []*Proxy

//Len()
func (ps Proxies) Len() int {
	return len(ps)
}

//Less(): 成绩将有低到高排序
func (ps Proxies) Less(i, j int) bool {
	return ps[i].expire.Before(ps[j].expire)
}

//Swap()
func (ps Proxies) Swap(i, j int) {
	ps[i], ps[j] = ps[j], ps[i]
}
