package v1

type VirtualServiceSpec struct {
	Hosts    []string                 `json:"hosts,omitempty"`
	Gateways []string                 `json:"gateways,omitempty"`
	Http     []VirtualServiceHttpSpec `json:"http,omitempty"`
}

type VirtualServiceHttpSpec struct {
	Name    string                      `json:"name"`
	Match   []VirtualServiceUri         `json:"match"`
	Rewrite VirtualServiceUri           `json:"rewrite"`
	Route   []VirtualServiceDestination `json:"route"`
}

type VirtualServiceUri struct {
	Prefix string `json:"prefix"`
}

type VirtualServiceDestination struct {
	Host   string `json:"host"`
	Subset string `json:"subset"`
}
