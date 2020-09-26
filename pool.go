package goPryPool

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var openPryMinNum = 10
var ErrPryExpired = errors.New("该代理已过期")
var ErrPoolClosed = errors.New("代理池已关闭")
var ErrPryClosed = errors.New("对应代理已关闭")
var ErrWaitCreatedPry = errors.New("等待创建新的代理")
var ErrMaxOpenCreatedPry = errors.New("已达到最大创建代理，无法创建新的代理")
var ErrPutCreatedPry = errors.New("写入代理池失败")

type Pool struct {
	driver Driver

	//最大代理数
	maxOpen int

	//最大空闲代理数
	maxIdle int

	//已获取的代理数
	openNum int

	freePry []*Proxy

	mu sync.Mutex

	pryRequests map[uint]chan *Proxy
	nextPryRequestKey uint

	openerCh          chan uint

	//等待请求数
	waitRequestCount uint

	nextPryRequest uint

	cancel context.CancelFunc

	ctx context.Context

	closed  bool

	waitDurationByPry int64
}

func OpenPool(apiDriver Driver) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		driver:               apiDriver,
		maxOpen:              0,
		maxIdle:              0,
		openNum:              0,
		pryRequests:          make(map[uint]chan *Proxy),
		nextPryRequestKey:    0,
		openerCh:             make(chan uint, 1),
		waitRequestCount:     0,
		nextPryRequest:       0,
		cancel:               cancel,
		closed:               false,
		waitDurationByPry: 0,
		ctx:ctx,
	}

	go p.pryOpener()
	return p
}

func (p *Pool) SetMaxOpen(num int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxOpen = num
	if p.maxOpen < 0 {
		p.maxOpen = 0
	}
}

func (p *Pool) GetMaxOpen() int {
	return p.maxOpen
}

func (p *Pool) SetMaxIdle(num int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxIdle = num
	if p.maxIdle < 0 {
		p.maxIdle = 0
	}
}

func (p Pool) GetOpenNum() int {
	return p.openNum
}

func (p Pool) GetFreePry() []*Proxy {
	return p.freePry
}
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	cancel := p.cancel
	cancel()
	log.Println("proxy pool closed....")
	return
}

func (p *Pool) pryOpener() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case num := <-p.openerCh:
			err := p.createPry(int(num))
			if err != nil {
				log.Println(err)
			}
		}
	}
}

//清理多余的空闲代理
func (p *Pool) PryIdleCleaner() {
	p.mu.Lock()
	defer p.mu.Unlock()

	//检查上下文是否关闭
	select {
	case <-p.ctx.Done():
		return
	default:
	}
	freeNum := len(p.freePry)
	sort.Sort(Proxies(p.freePry))
	if p.maxIdle > 0 && freeNum > p.maxIdle {
		i := freeNum - p.maxIdle
		copy(p.freePry, p.freePry[i:])
		p.freePry = p.freePry[:freeNum - i]
		p.openNum -= i
	}
}

//创建代理
//需要在创建之前 增加对应的openNum 这里创建失败后会对应减去
func (p *Pool) createPry(num int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.maxOpen > 0 {
		if p.openNum > p.maxOpen {
			p.openNum -= num
			return ErrMaxOpenCreatedPry
		}
	}
	cpc := 0
	RE:
	pry, err := p.driver.CreateProxies(num, p)
	if err != nil {
		if cpc < 5 {
			cpc++
			time.Sleep(1 * time.Second)
			goto RE
		}
		p.openNum -= num
		return err
	}
	cPryNum := len(pry)
	if cPryNum > num {
		pry = pry[:num]
	}
	if !p.putPryLocked(pry...) {
		p.openNum -= num
		return ErrPutCreatedPry
	} else {
		//需要补齐openNum
		if cPryNum < num {
			p.openNum -= (num - cPryNum)
		}
	}
	if p.maxIdle > 0 {
		go p.PryIdleCleaner()
	}
	return nil
}

func (p *Pool) PutPry(pry *Proxy) bool {
	
	if pry.IsClosed() {
		return false
	}

	if pry.IsExpiredAndClose() {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.maxOpen > 0 && !pry.IsUse() {
		if p.openNum + 1 > p.maxOpen {
			return false
		}
	}

	return p.putPryLocked(pry)
}

//该方法只处理进来的代理 不会做任何计数操作
func (p *Pool) putPryLocked(pry ...*Proxy) bool {
	pryNum := len(pry)
	reqC := len(p.pryRequests)
	//有等待队列 则优先处理
	if reqC > 0 {
		i := 0
		for key, req := range p.pryRequests {
			if i == pryNum - 1 {
				break
			}
			delete(p.pryRequests, key)
			pry[i].SetUse()
			req <- pry[i]
			p.waitRequestCount--
			i++
		}

		if i == pryNum - 1 {
			return true
		}

		//处理一下剩余的代理
		copy(pry, pry[i:])
		pry = pry[:pryNum - i]
	}

	//加入空闲池
	p.freePry = append(p.freePry, pry...)
	return true
}

func (p *Pool) nextRequestKeyLocked() uint {
	next := p.nextPryRequestKey
	p.nextPryRequestKey++
	return next
}

func (p *Pool) CreateNewProxies() {
	p.mu.Lock()
	defer p.mu.Unlock()
	num := len(p.pryRequests)
	if num < openPryMinNum {
		num = openPryMinNum
	}
	if p.maxOpen > 0 {
		if p.openNum + num > p.maxOpen {
			num = p.maxOpen - p.openNum
			if num <= 0 {
				return
			}
		}
	}
	p.openNum += num
	p.openerCh <- uint(num)
}

func (p *Pool) GetPry() (*Proxy, error) {
	var pry *Proxy
	var err error
	for i := 0; (i < p.maxOpen + 10); i++ {
		pry, err = p.get()
		if err == nil {
			if pry.useNumTotal <= 1 {
				if pry.expireAdvanceAsNotUse > 0 && pry.expire.Before(time.Now().Add(pry.expireAdvanceAsNotUse)) {
					continue
				}
			}
			return pry, nil
		}

		if err == ErrWaitCreatedPry {
			log.Println("wait created proxy...")
			runtime.Gosched()
		}
		if err == ErrPryExpired {
			log.Println("proxy expired...")
			runtime.Gosched()
		}
	}
	return nil, err
}

func (p *Pool) get() (*Proxy, error) {
	p.mu.Lock()

	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}

	//检查上下文是否关闭
	select {
	case <-p.ctx.Done():
		p.mu.Unlock()
		return nil, p.ctx.Err()
	default:
	}

	//获取闲置代理数
	numFree := len(p.freePry)
	//如果存在闲置连接
	if numFree > 0 {
		pry := p.freePry[0]
		copy(p.freePry, p.freePry[1:])
		p.freePry = p.freePry[:numFree - 1]
		p.mu.Unlock()
		pry.SetUse()
		if pry.IsClosed() {
			return nil, ErrPryClosed
		}
		if pry.IsExpiredAndClose() {
			return nil, ErrPryExpired
		}
		return pry, nil
	}

	//没有闲置代理 并且已经创建打开的代理数大于等于最大限制数
	if p.maxOpen > 0 && p.openNum >= p.maxOpen {
		req := make(chan *Proxy)
		reqKey := p.nextRequestKeyLocked()
		p.pryRequests[reqKey] = req
		p.waitRequestCount++
		p.mu.Unlock()

		waitStart := time.Now()
		select {
		case <-p.ctx.Done():
			delete(p.pryRequests, reqKey)
			atomic.AddInt64(&p.waitDurationByPry, int64(time.Since(waitStart)))
			return nil, p.ctx.Err()
		case pry, ok := <-req:
			atomic.AddInt64(&p.waitDurationByPry, int64(time.Since(waitStart)))
			if !ok {
				return nil, ErrPoolClosed
			}

			if pry.IsClosed() {
				return nil, ErrPryClosed
			}

			if pry.IsExpiredAndClose() {
				return nil, ErrPryExpired
			}
			return pry, nil
		}
	}
	p.mu.Unlock()
	p.CreateNewProxies()
	return nil, ErrWaitCreatedPry
}
