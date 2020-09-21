package goPryPool

type Driver interface {
	CreateProxies(num int, pool *Pool) ([]*Proxy, error)
}