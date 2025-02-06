package hostiptopodmap

import "sync"

type ActivePods struct {
	mu          sync.RWMutex
	connections map[string]string
}

func NewActivePods() *ActivePods {
	return &ActivePods{
		connections: make(map[string]string),
	}
}

func (pods *ActivePods) Add(link string, podDNS string) {
	pods.mu.Lock()
	defer pods.mu.Unlock()
	pods.connections[link] = podDNS
}

func (pods *ActivePods) Get(link string) string {
	pods.mu.RLock()
	defer pods.mu.RUnlock()
	if dns, exists := pods.connections[link]; exists {
		return dns
	}
	return ""
}
func (pods *ActivePods) Delete(link string) {
	pods.mu.Lock()
	defer pods.mu.Unlock()
	delete(pods.connections, link)
}
