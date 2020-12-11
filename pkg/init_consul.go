package main

import (
	"github.com/NVIDIA/gpu-monitoring-tools/pkg/consul"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func InitConsul(consulUrl, serviceName, ipPort string) {

	ip := strings.Split(ipPort, ":")
	if len(ip) != 2 {
		logrus.Fatal("ipPort not len 2")

	}
	port, err := strconv.Atoi(ip[1])
	if err != nil {
		logrus.Fatalf("strconv: %s Atoi err: %v", ip[1], err)
	}

	sd, err := consul.NewServiceDiscovery(serviceName, ip[0], uint64(port), 30, 1, "health", consulUrl)
	if err != nil {
		logrus.Fatalf("consul new service discovery err: %v", err)

	}
	if err := sd.ServiceRegistr(); err != nil {
		logrus.Fatalf("service register err: %v", err)
	}
}