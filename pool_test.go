package goPryPool

import (
	"log"
	"testing"
	"time"
)

func TestGetProxy(t *testing.T) {
	pool := OpenPool(&HdProxy{})
	pool.SetWaitPryTimeoutForGet(1 * time.Second)
	pool.SetMaxOpen(1)
	pry, err := pool.GetPry()
	if err != nil {
		log.Fatal(err)
	}/*
	log.Println("==================================")
	pry2, err := pool.GetPry()
	if err != nil {
		log.Fatal(err)
	}
	pry2.Close()*/
	pry.Close()
	log.Println(pry.GetProxyUrl())
	log.Println("空闲代理:", len(pool.GetFreePry()))
	log.Println("打开数", pool.GetOpenNum())

	log.Println("释放代理")
	log.Println(pry.Release())
	log.Println("空闲代理:", len(pool.GetFreePry()))
	log.Println("打开数", pool.GetOpenNum())
}
