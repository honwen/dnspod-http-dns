package dnspod

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/xuyu/ipfilter"
)

const dnspodURL = `http://119.29.29.29/d`
const rejectedData = `0.0.0.0,30`

func getAtoRR(qname, ip string, ttl uint32) dns.RR {
	hdr := dns.RR_Header{Name: qname, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}
	str := hdr.String() + ip
	rr, _ := dns.NewRR(str)
	return rr
}

// DNSPOD is the DNSPOD DNS-over-HTTP provider;
type DNSPOD struct {
	fallbackENDS     string
	fallbackIPFilter *ipfilter.IPFilter

	client *http.Client
}

// NewDNSPOD creates a DNSPOD
func NewDNSPOD(fallbackENDS string) (d *DNSPOD) {
	d = new(DNSPOD)
	d.fallbackENDS = fallbackENDS
	if 0 == len(d.fallbackENDS) {
		d.fallbackENDS = "119.29.29.29"
	}
	d.fallbackIPFilter = new(ipfilter.IPFilter)
	d.fallbackIPFilter.AddIPNetString("169.254.0.0/16")
	d.fallbackIPFilter.AddIPNetString("192.168.0.0/16")
	d.fallbackIPFilter.AddIPNetString("172.16.0.0/12")
	d.fallbackIPFilter.AddIPNetString("127.0.0.0/8")
	d.fallbackIPFilter.AddIPNetString("10.0.0.0/8")

	d.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return
}

// DNSHandleFunc provider miekg/dns.HandleFunc
func (d *DNSPOD) DNSHandleFunc(w dns.ResponseWriter, req *dns.Msg) {
	/* any questions? */
	if len(req.Question) < 1 {
		return
	}
	q := req.Question[0]
	rmsg := new(dns.Msg)
	rmsg.SetReply(req)
	rawData := "" // ip;...ip,ttl
	switch q.Qtype {
	case dns.TypeA:
		log.Println("requesting", q.Name, dns.TypeToString[q.Qtype])
		httpreq, _ := http.NewRequest(http.MethodGet, dnspodURL, nil)
		qry := httpreq.URL.Query()
		qry.Add("dn", q.Name)
		qry.Add("ttl", "1")

		edns := w.RemoteAddr().String()
		edns = edns[0:strings.LastIndex(edns, ":")]

		if d.fallbackIPFilter.FilterIPString(edns) {
			edns = d.fallbackENDS
		}
		qry.Add("ip", edns) //EDNS

		httpreq.URL.RawQuery = qry.Encode()
		httpresp, err := d.client.Do(httpreq)
		if err != nil {
			log.Println("HTTP GET faild", err)
			break
		}
		body, err := ioutil.ReadAll(httpresp.Body)
		if err != nil {
			log.Println("ReadAll HTTP Body faild", err)
			break
		} else {
			httpresp.Body.Close()
		}
		rawData = string(body)
		// log.Println(httpreq.URL, " | ", rawData)
	default:
		log.Println("request-Blocked", q.Name, dns.TypeToString[q.Qtype])
		// rmsg.MsgHdr.Response = false
	}

	if 0 == len(rawData) {
		rawData = rejectedData
	}
	dataStr := strings.Split(rawData, ",")
	ttl := uint64(0)
	if 2 == len(dataStr) {
		ttl, _ = strconv.ParseUint(dataStr[1], 0, 32)
	}
	if ttl == 0 {
		ttl = 30 // minTTL
	}
	ips := strings.Split(dataStr[0], ";")
	for _, ip := range ips {
		rmsg.Answer = append(rmsg.Answer, getAtoRR(q.Name, ip, uint32(ttl)))
	}

	// log.Printf("%+v", rmsg)
	if err := w.WriteMsg(rmsg); err != nil {
		log.Println("Response faild", err)
	}
}
