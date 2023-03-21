package balancer

type Balancer interface {
	Pick(ids []string) (string, error)
}
