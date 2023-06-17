package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

var facelist map[uint64]faces

func main() {
	facelist = make(map[uint64]faces)

	var wg sync.WaitGroup

	wg.Add(1)

	// consumer("/ndn/coba")

	// //Serve /hello interest
	time.Sleep(1 * time.Second)

	// //hello protocol every 5 second
	var (
		client mgmt.Client
		face   mgmt.Face
		fwFace l3.FwFace
	)

	client, e := nfdmgmt.New()

	face, e = client.OpenFace()
	if e != nil {
		fmt.Println(e)
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		fmt.Println(e)
	}
	fwFace.AddRoute(ndn.Name{})
	fw.AddReadvertiseDestination(face)

	log.Printf("uplink opened, state is %s", l3face.State())
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})

	interval := 10 * time.Second
	interval_interest := 100 * time.Millisecond
	for {
		//update facelist
		update_facelist()
		fmt.Println(facelist)

		//create route
		for k, v := range facelist {
			register_route(v.tkn, 0, int(k))

			fmt.Println(k, v.tkn)
			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("hello"), ndn.ForwardingHint{ndn.ParseName(v.tkn), ndn.ParseName("hello")})

			data, rtt, thg, e := consumer_interest(interest)

			if e != nil {
				continue
			}

			fmt.Println(data)

			v.ngb = data
			v.rtt = rtt
			v.thg = thg
			facelist[k] = v

			time.Sleep(interval_interest)
		}
		fmt.Println(facelist)

		time.Sleep(interval)
	}

	// go producer("hello", "Hello World!", 10)

	// time.Sleep(1 * time.Second)

	// var (
	// 	client mgmt.Client
	// 	face   mgmt.Face
	// 	fwFace l3.FwFace
	// )

	// client, e := nfdmgmt.New()

	// face, e = client.OpenFace()
	// if e != nil {
	// 	fmt.Println(e)
	// }
	// l3face := face.Face()

	// fw := l3.GetDefaultForwarder()
	// if fwFace, e = fw.AddFace(l3face); e != nil {
	// 	fmt.Println(e)
	// }
	// fwFace.AddRoute(ndn.Name{})
	// fw.AddReadvertiseDestination(face)

	// log.Printf("uplink opened, state is %s", l3face.State())
	// l3face.OnStateChange(func(st l3.TransportState) {
	// 	log.Printf("uplink state changes to %s", l3face.State())
	// })

	// data, _, _, e := consumer("hello")
	// fmt.Println(data)
	// fmt.Println(e)

	wg.Wait()

}

func consumer(name string) (content string, rtt float64, thg float64, e error) {
	// seqNum := rand.Uint64()
	// var nData, nErrors atomic.Int64

	interest := ndn.ParseName(name)

	t0 := time.Now()

	data, e := endpoint.Consume(context.Background(), ndn.MakeInterest(interest),
		endpoint.ConsumerOptions{})

	rtt = float64(time.Since(t0).Milliseconds())

	if e == nil {
		// nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		// fmt.Println(data.Content)
		content = string(data.Content[:])
		// fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
		thg = float64(len(content)) / float64(rtt/1000)
	} else {
		// nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		// fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		return content, 0, 0, e
	}

	return content, rtt, thg, nil
}

func consumer_interest(Interest ndn.Interest) (content string, rtt float64, thg float64, e error) {
	// seqNum := rand.Uint64()
	// var nData, nErrors atomic.Int64

	t0 := time.Now()

	data, e := endpoint.Consume(context.Background(), Interest,
		endpoint.ConsumerOptions{})

	raw_rtt := time.Since(t0)

	rtt = float64(raw_rtt / time.Millisecond)

	fmt.Println(rtt)

	if e == nil {
		// nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		// fmt.Println(data.Content)
		content = string(data.Content[:])
		// fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
		thg = float64(len(content)) / float64(rtt/1000)
	} else {
		// nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		// fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		return content, 0, 0, e
	}

	return content, rtt, thg, nil
}

func update_facelist() {
	c, _ := nfdmgmt.New()

	var sigNonce [8]byte
	rand.Read(sigNonce[:])

	interest := ndn.Interest{
		Name:        ndn.ParseName("/localhost/nfd/faces/list"),
		MustBeFresh: true,
		CanBePrefix: true,
		SigInfo: &ndn.SigInfo{
			Nonce: sigNonce[:],
			Time:  uint64(time.Now().UnixMilli()),
		},
	}

	c.Signer.Sign(&interest)

	data, e := endpoint.Consume(context.Background(), interest,
		endpoint.ConsumerOptions{})

	if e != nil {
		fmt.Println(e)
	} else {
		parse_facelist(data.Content)
	}
}

func register_route(name string, cost int, faceid int) {

	c, _ := nfdmgmt.New()

	cr, e := c.Invoke(context.TODO(), nfdmgmt.RibRegisterCommand{
		Name:   ndn.ParseName(name),
		Origin: 0,
		Cost:   cost,
		FaceID: faceid,
	})

	if e != nil {
		fmt.Println(e)
	}
	if cr.StatusCode != 200 {
		fmt.Println("unexpected response status %d", cr.StatusCode)
	} else {
		fmt.Println("Route registered")
	}
}

type faces struct {
	ngb  string
	rtt  float64
	thg  float64
	tkn  string
	n_oi uint64
	n_in uint64
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytes(n int) string {
	var src = rand.NewSource(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func parse_facelist(raw []byte) {
	var (
		nextface uint64
		length   uint64
		pointer  uint64
		faceid   uint64
		innack   uint64
		outi     uint64
		uri      string
	)

	pointer = 0
	for pointer != uint64(len(raw)) {
		// check per face
		if data := hex.EncodeToString([]byte{raw[pointer]}); data == "80" {
			pointer++
			// fmt.Println("data:", data)
			octet := check_type([]byte{raw[pointer]})
			if octet == 1 {
				length = check_length([]byte{raw[pointer]})
			} else {
				length = check_length(raw[pointer : pointer+octet])
			}
			pointer += octet
			nextface = pointer + length

			for pointer < nextface {
				if data := hex.EncodeToString([]byte{raw[pointer]}); data == "69" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					faceid = get_data(raw[pointer : pointer+length])
					// fmt.Println("faceid: ", faceid)
					pointer += length
				} else if data := hex.EncodeToString([]byte{raw[pointer]}); data == "92" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					outi = get_data(raw[pointer : pointer+length])
					// fmt.Println("outi: ", outi)
					pointer += length
				} else if data := hex.EncodeToString([]byte{raw[pointer]}); data == "97" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					innack = get_data(raw[pointer : pointer+length])
					// fmt.Println("innack: ", innack)
					pointer += length
				} else if data := hex.EncodeToString([]byte{raw[pointer]}); data == "72" {
					// fmt.Println("data:", data)
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet
					uri = get_str_data(raw[pointer : pointer+length])
					// fmt.Println("innack: ", innack)
					pointer += length
				} else {
					pointer++
					octet := check_type([]byte{raw[pointer]})
					if octet == 1 {
						length = check_length([]byte{raw[pointer]})
					} else {
						length = check_length(raw[pointer : pointer+octet])
					}
					pointer += octet + length
				}
			}

			// token := make([]byte, 16)
			// rand.Read(token)
			// stoken := hex.EncodeToString(token)
			stoken := "/" + RandStringBytes(16)
			fmt.Println(uri)
			if _, ok := facelist[faceid]; ok {
				fmt.Println("Use existing")
				facelist[faceid] = faces{n_oi: outi, n_in: innack, tkn: facelist[faceid].tkn, ngb: facelist[faceid].ngb, rtt: facelist[faceid].rtt, thg: facelist[faceid].thg}
			} else {
				fmt.Println("Create new")
				facelist[faceid] = faces{n_oi: outi, n_in: innack, tkn: stoken}
			}

		}
	}
}

func check_type(wire []byte) (res uint64) {
	tlv_type := int(wire[0])
	if tlv_type < 253 {
		res = 1
	} else if tlv_type == 253 {
		res = 3
	} else if tlv_type == 254 {
		res = 5
	} else {
		res = 9
	}

	return res
}

func check_length(wire []byte) (res uint64) {
	byte_length := len(wire)
	if byte_length == 1 {
		res = uint64(wire[0])
	} else if byte_length == 2 {
		res = uint64(binary.BigEndian.Uint16(wire))
	} else if byte_length == 4 {
		res = uint64(binary.BigEndian.Uint32(wire))
	} else {
		res = binary.BigEndian.Uint64(wire)
	}

	return res
}

func get_data(wire []byte) (res uint64) {
	byte_length := len(wire)
	if byte_length == 1 {
		res = uint64(wire[0])
	} else if byte_length == 2 {
		res = uint64(binary.BigEndian.Uint16(wire))
	} else if byte_length == 4 {
		res = uint64(binary.BigEndian.Uint32(wire))
	} else {
		res = binary.BigEndian.Uint64(wire)
	}

	return res
}

func get_str_data(wire []byte) (res string) {
	res = string(wire)

	return res
}
