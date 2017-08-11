package dnspod

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// DNSpodIP 119.29.29.29 or 182.254.116.116
const DNSpodIP = `119.29.29.29`
const rejectedData = `0.0.0.0,30`

var dnspodURL = fmt.Sprintf(`http://%s/d`, DNSpodIP)

func getAtoRR(qname, ip string, ttl uint32) dns.RR {
	hdr := dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}
	str := hdr.String() + ip
	rr, _ := dns.NewRR(str)
	return rr
}

// DNSPOD is the DNSPOD DNS-over-HTTP provider;
type DNSPOD struct {
	EDNS         net.IP
	FallbackDNS  *net.TCPAddr
	httpclient   *http.Client
	dnsTCPclient *dns.Client
	dnsUDPclient *dns.Client
}

// NewDNSPOD creates a DNSPOD
func NewDNSPOD(EDNS string) (dp *DNSPOD) {
	dp = new(DNSPOD)
	dp.EDNS = net.ParseIP(EDNS)
	dp.FallbackDNS, _ = net.ResolveTCPAddr("tcp4", DNSpodIP+":53")

	dp.httpclient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: false,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	dp.dnsTCPclient = &dns.Client{Net: "tcp", Timeout: 5 * time.Second}
	dp.dnsUDPclient = &dns.Client{Net: "udp", Timeout: 5 * time.Second}
	return
}

// DNSHandleFunc provider miekg/dns.HandleFunc
func (dp *DNSPOD) DNSHandleFunc(w dns.ResponseWriter, req *dns.Msg) {
	var err error
	/* any questions? */
	if len(req.Question) < 1 {
		return
	}

	rmsg := new(dns.Msg)
	rmsg.SetReply(req)
	rawData := "" // ip;...ip,ttl
	q := req.Question[0]
	switch q.Qtype {
	case dns.TypeA:
		log.Println("requesting:", q.Name, dns.TypeToString[q.Qtype])
		httpreq, _ := http.NewRequest(http.MethodGet, dnspodURL, nil)
		qry := httpreq.URL.Query()
		qry.Add("dn", q.Name)
		qry.Add("ttl", "1")
		if nil != dp.EDNS {
			qry.Add("ip", dp.EDNS.String()) //EDNS
		}

		httpreq.URL.RawQuery = qry.Encode()
		httpresp, err := dp.httpclient.Do(httpreq)
		if nil != err {
			log.Println("HTTP GET faild", err)
		} else {
			if body, err := ioutil.ReadAll(httpresp.Body); err != nil {
				log.Println("ReadAll HTTP Body faild", err)
			} else {
				rawData = string(body)
				httpresp.Body.Close()
			}
		}
		// log.Println(httpreq.URL, " | ", rawData)
		if !strings.Contains(rawData, `.`) {
			rmsg = nil
		} else {
			dataStr := strings.Split(rawData, ",")
			ttl := uint64(0)
			if 2 == len(dataStr) {
				ttl, _ = strconv.ParseUint(dataStr[1], 0, 32)
			}
			if ttl == 0 {
				ttl = 30 // minTTL
			}
			ips := strings.Split(dataStr[0], ";")
			for idx := range ips {
				rmsg.Answer = append(rmsg.Answer, getAtoRR(q.Name, ips[idx], uint32(ttl)))
			}
		}
	case dns.TypeANY, dns.TypeAAAA:
		log.Println("request-block", q.Name, dns.TypeToString[q.Qtype])
	default:
		rmsg = nil
	}

	if nil == rmsg {
		rmsg = dp.normalDNS(req)
		if nil == rmsg {
			log.Println("request-error", q.Name, dns.TypeToString[q.Qtype])
			rmsg = new(dns.Msg)
			rmsg.SetReply(req)
		}
	}
	// fmt.Println(rmsg)
	if err = w.WriteMsg(rmsg); nil != err {
		log.Println("Response faild, rmsg:", err)
	}
}

func (dp *DNSPOD) normalDNS(req *dns.Msg) (rmsg *dns.Msg) {
	q := req.Question[0]
	if nil != req {
		log.Println("request-fallback-TCP:", q.Name, dns.TypeToString[q.Qtype])
		rmsg, _, _ = dp.dnsTCPclient.Exchange(req, dp.FallbackDNS.String())
	}
	if nil == rmsg {
		log.Println("request-fallback-UDP:", q.Name, dns.TypeToString[q.Qtype])
		rmsg, _, _ = dp.dnsUDPclient.Exchange(req, dp.FallbackDNS.String())
	}
	return
}
