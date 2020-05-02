package net

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"
)

type DomainInfo struct {
	DomainName             string
	RegistrarServer        string
	CreationDate           string
	UpdatedDate            string
	RegistryExpiryDate     string
	Registrar              string
	RegistrantOrganization string
}

// query server const
const (
	DomainWhoisServer = "whois-servers.net"
	Port              = "43"
)

// whois do the whois query and returns whois info
func Whois(domain string) (result DomainInfo, err error) {
	// trim domain
	domainSlice := strings.Split(domain, ".")
	if len(domainSlice) < 2 {
		fmt.Printf("Domain %s is invalid", domain)
		return
	}
	if domainSlice[len(domainSlice)-2] == "" {
		fmt.Printf("Domain %s is invalid", domain)
		return
	}
	domain = fmt.Sprintf("%s.%s", domainSlice[len(domainSlice)-2], domainSlice[len(domainSlice)-1])
	if domain == "" {
		log.Println("Domain is empty")
		return
	}
	// do query
	result, err = Query(domain)
	if err != nil {
		return
	}
	return
}

// query do the query
func Query(domain string) (result DomainInfo, err error) {
	// join domain server
	domainSlice := strings.Split(domain, ".")
	server := domainSlice[len(domainSlice)-1] + "." + DomainWhoisServer
	// do conn
	conn, e := net.DialTimeout("tcp", net.JoinHostPort(server, Port), time.Second*30)
	if e != nil {
		err = e
		return
	}
	_, _ = conn.Write([]byte(domain + "\r\n"))
	_ = conn.SetReadDeadline(time.Now().Add(time.Second * 30))
	// get result
	buffer, e := ioutil.ReadAll(conn)
	conn.Close()
	if e != nil {
		err = e
		return
	}
	// do parser
	result = Parser(string(buffer))
	return
}

func Parser(body string) DomainInfo {
	var domainInfo DomainInfo
	bodyInterface := strings.Split(body, "\r")
	for _, info := range bodyInterface {
		info = strings.TrimSpace(info)
		infoSlice := strings.Split(info, ":")
		switch infoSlice[0] {
		case "Domain Name":
			domainInfo.DomainName = infoSlice[1]
		case "Registrar WHOIS Server":
			domainInfo.RegistrarServer = infoSlice[1]
		case "Creation Date":
			domainInfo.CreationDate = infoSlice[1]
		case "Updated Date":
			domainInfo.UpdatedDate = infoSlice[1]
		case "Registry Expiry Date":
			domainInfo.RegistryExpiryDate = infoSlice[1]
		case "Registrar":
			domainInfo.Registrar = infoSlice[1]
		case "Registrant Organization":
			domainInfo.RegistrantOrganization = infoSlice[1]
		default:
		}
	}
	return domainInfo
}
