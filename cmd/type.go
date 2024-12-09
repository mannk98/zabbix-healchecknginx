package cmd

type containerInfos struct {
	Index    int    `json:"index"`
	Upstream string `json:"upstream"`
	IpPort   string `json:"name"`
}

type config struct {
	NginxContainerIP string
}
