package proxy

type Driver interface {
	CreateProxies(num int, pool *Pool) ([]*Proxy, error)
}