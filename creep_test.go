package main

import (
	"fmt"
	"github.com/Qitmeer/qitmeer/params"
	"net"
	"path/filepath"
	"testing"
	"time"
)

func Test_creep1(t *testing.T) {
	var err error
	activeNetParams = &params.TestNetParams
	manager, err = NewManager(filepath.Join(defaultHomeDir,
		activeNetParams.Name))
	if err != nil {
		t.Fatal(err.Error())
	}
	custom := []net.IP{
		net.ParseIP("121.196.55.29"),
		net.ParseIP("121.196.28.213"),
		net.ParseIP("121.196.54.163"),
		net.ParseIP("47.114.183.16"),
		net.ParseIP("47.114.184.240"),
	}
	manager.AddAddresses(custom)
	globalWg.Add(1)
	go creep()
	globalWg.Add(1)
	go func() {
		ticker := time.NewTicker(time.Second * 1)
		for i := 0; i <= 100; i++ {
			select {
			case _ = <-ticker.C:
				fmt.Printf("Test print ---- address length = %d\n", len(manager.Addresses()))
				fmt.Printf("Test print ---- address = %v\n", manager.Addresses())
			}
		}
		globalWg.Done()
	}()
	time.Sleep(180 * time.Second)
	t.Logf("Test print ---- address length = %d", len(manager.Addresses()))
	t.Logf("Test print ---- address = %v", manager.Addresses())
}

func Test_creep2(t *testing.T) {
	var err error
	activeNetParams = &params.TestNetParams
	manager, err = NewManager(filepath.Join(defaultHomeDir,
		activeNetParams.Name))
	if err != nil {
		t.Fatal(err.Error())
	}
	custom := []net.IP{
		net.ParseIP("121.196.55.29"),
		net.ParseIP("121.196.28.213"),
		net.ParseIP("121.196.54.163"),
		net.ParseIP("47.114.183.16"),
		net.ParseIP("47.114.184.240"),
	}
	customFindIps := map[string]bool{}
	for _, ip := range custom {
		newIps := creepOne(ip)
		for _, ip := range newIps {
			customFindIps[ip.String()] = true
		}
	}
	newIpFindIps := map[string][]net.IP{}
	for ip, _ := range customFindIps {
		newIps := creepOne(net.ParseIP(ip))
		newIpFindIps[ip] = newIps
	}
	find := false
	for newIp, findIps := range newIpFindIps {
		for _, customIp := range custom {
			for _, findIp := range findIps {
				if customIp.String() == findIp.String() {
					find = true
					t.Logf("The custom address %s can be found by the self-discovered node %s", customIp.String(), newIp)
				}
			}
		}
	}
	if !find {
		t.Fatal("Custom address cannot be found by a node of its own discovery")
	}
}
