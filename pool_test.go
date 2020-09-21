package goPryPool

import (
	"log"
	"testing"
)

func TestGetProxy(t *testing.T) {
	pool := OpenPool(&ZmProxy{})
	pool.SetMaxOpen(2)
	pry, err := pool.GetPry()
	if err != nil {
		log.Fatal(err)
	}

	log.Println(pry.GetProxyUrl())
	log.Println("空闲代理:", len(pool.GetFreePry()))
	log.Println("打开数", pool.GetOpenNum())

	log.Println("释放代理")
	log.Println(pry.Release())
	log.Println("空闲代理:", len(pool.GetFreePry()))
	log.Println("打开数", pool.GetOpenNum())
}
