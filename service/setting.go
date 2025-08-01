package service

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
)

type SettingIP struct {
	Method  string `json:"method"`
	IP      string `json:"ip"`
	Mask    string `json:"mask"`
	Gateway string `json:"gateway"`
}

const filePath = "/lib/systemd/network/36-wired.network"
const SettingIPFile = "/home/root/edge/ip.json"

const dhcp = `[Match]
Name=eth0

[Network]
DHCP=ipv4

[DHCP]
CriticalConnection=true
`

const staticIP = `[Match]
Name=eth0

[Network]
Address={{.Address}}
Gateway={{.Gateway}}
`

func reboot() error {
	cmd := exec.Command("reboot")

	return cmd.Run()
}

func restartNetwork() error {
	cmd := exec.Command("systemctl", "restart", "systemd-networkd")

	return cmd.Run()
}

func settingDHCP() error {

	// old, err := os.ReadFile(filePath)
	// if err != nil {
	// 	return err
	// }

	err := os.WriteFile(filePath, []byte(dhcp), 0666)
	if err != nil {
		return err
	}

	restartNetwork()
	// err = restartNetwork()
	// if err != nil {
	// 	os.WriteFile(filePath, old, 0666)
	// 	return err
	// }

	s := new(SettingIP)
	s.Method = "dhcp"
	data, _ := json.Marshal(s)
	os.WriteFile(SettingIPFile, data, 0666)

	return nil
}

func getAddress(ips, nms string) string {

	ip := net.ParseIP(ips)
	nmip := net.ParseIP(nms)

	if ip == nil || nmip == nil {
		return ""
	}

	if ip.To4() == nil || nmip.To4() == nil {
		return ""
	}

	nm := net.IPMask(nmip)

	ipnet := net.IPNet{
		IP:   ip,
		Mask: nm,
	}

	return ipnet.String()
}

func settingStaticIP(s *SettingIP) error {

	// old, err := os.ReadFile(filePath)
	// if err != nil {
	// 	return err
	// }

	t := template.New("t")
	t, err := t.Parse(staticIP)
	if err != nil {
		return err
	}

	addr := getAddress(s.IP, s.Mask)
	if addr == "" {
		return fmt.Errorf("ip address error")
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, 0666)

	if err != nil {
		return err
	}

	err = t.Execute(f, struct {
		Address string
		Gateway string
	}{addr, s.Gateway})

	if err != nil {
		return err
	}

	f.Close()

	// restart systemd-networkd
	restartNetwork()
	// err = restartNetwork()
	// if err != nil {
	// 	os.WriteFile(filePath, old, 0666)
	// 	return err
	// }

	data, _ := json.Marshal(s)
	ioutil.WriteFile(SettingIPFile, data, 0666)
	return nil
}

func getSystemIP() *SettingIP {

	data, err := ioutil.ReadFile(SettingIPFile)
	if err != nil {
		return nil
	}

	result := new(SettingIP)

	err = json.Unmarshal(data, &result)

	if err != nil {
		return nil
	}

	return result
}

// func getSystemSN() ([]byte, error) {

// 	sn, err := ioutil.ReadFile(SNFile)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sn, nil
// }

func settingSystemIP(item *SettingIP) error {

	m := strings.ToUpper(item.Method)
	if m == "DHCP" {

		return settingDHCP()

	}

	if m == "STATIC" {

		return settingStaticIP(item)

	}

	return fmt.Errorf("setting ip error")

}

func LocalIP() string {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addresses {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.To4().String()
			}
		}
	}
	return ""

}
