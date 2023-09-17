package core

import (
	"encoding/binary"
	"net"
	"strconv"
	"strings"
)

type ipCIDR struct {
	ip   string
	mask net.IPMask
}

var lanIPCIDRs []ipCIDR = []ipCIDR{
	*parseIPCIDR("127.0.0.0/8"),    // 127.0.0.0/8 环回地址
	*parseIPCIDR("10.0.0.0/8 "),    // 10.0.0.0/8 私有地址空间A类
	*parseIPCIDR("172.16.0.0/12"),  // 172.16.0.0/12 私有地址空间B类
	*parseIPCIDR("192.168.0.0/16"), // 192.168.0.0/16 私有地址空间C类
}

func parseIPCIDR(ipCIDRStr string) *ipCIDR {
	ss := strings.Split(ipCIDRStr, "/")
	n, _ := strconv.Atoi(ss[1])
	return &ipCIDR{
		ip:   ss[0],
		mask: getIPv4Mask(n),
	}
}

func uint32ToBytes(i uint32) []byte {
	var buf = make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func getIPv4Mask(n int) net.IPMask {
	f := uint32(0xffffffff)
	m := f << (32 - n)
	bs := uint32ToBytes(m)
	return net.IPv4Mask(bs[0], bs[1], bs[2], bs[3])
}

func IsLanIP(ip string) bool {
	for _, ipCIDR := range lanIPCIDRs {
		_ip := net.ParseIP(ip)
		if _ip == nil {
			return false
		}
		mip := _ip.Mask(ipCIDR.mask).String()
		if mip == ipCIDR.ip {
			return true
		}
	}
	return false
}
