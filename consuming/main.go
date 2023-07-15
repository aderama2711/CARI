package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
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

type faces struct {
	ngb  string
	rtt  float64
	thg  float64
	tkn  string
	n_oi uint64
	n_in uint64
}

// var facelist map[uint64]faces

func main() {
	fl := make(chan map[uint64]faces)
	pfl := make(chan map[uint64]faces)

	var wg sync.WaitGroup

	wg.Add(1)

	go hello(fl)
	go producer_channel("/facelist", pfl, 1000)

	for {
		select {
		case data := <-fl:
			fmt.Println(data)
			pfl <- data
		}
	}

	wg.Wait()

}

func hello(fl chan map[uint64]faces) {
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
	log.Printf("hello")
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})

	facelist := make(map[uint64]faces)

	interval := 10 * time.Second
	for {
		//update facelist
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
			// parse_facelist(data.Content)
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
			for pointer != uint64(len(data.Content)) {
				// check per face
				if header := hex.EncodeToString([]byte{data.Content[pointer]}); header == "80" {
					pointer++
					// fmt.Println("header:", header)
					octet := check_type([]byte{data.Content[pointer]})
					if octet == 1 {
						length = check_length([]byte{data.Content[pointer]})
					} else {
						length = check_length(data.Content[pointer : pointer+octet])
					}
					pointer += octet
					nextface = pointer + length

					for pointer < nextface {
						if header := hex.EncodeToString([]byte{data.Content[pointer]}); header == "69" {
							// fmt.Println("header:", header)
							pointer++
							octet := check_type([]byte{data.Content[pointer]})
							if octet == 1 {
								length = check_length([]byte{data.Content[pointer]})
							} else {
								length = check_length(data.Content[pointer : pointer+octet])
							}
							pointer += octet
							faceid = get_data(data.Content[pointer : pointer+length])
							// fmt.Println("faceid: ", faceid)
							pointer += length
						} else if header := hex.EncodeToString([]byte{data.Content[pointer]}); header == "92" {
							// fmt.Println("header:", header)
							pointer++
							octet := check_type([]byte{data.Content[pointer]})
							if octet == 1 {
								length = check_length([]byte{data.Content[pointer]})
							} else {
								length = check_length(data.Content[pointer : pointer+octet])
							}
							pointer += octet
							outi = get_data(data.Content[pointer : pointer+length])
							// fmt.Println("outi: ", outi)
							pointer += length
						} else if header := hex.EncodeToString([]byte{data.Content[pointer]}); header == "97" {
							// fmt.Println("data:", data)
							pointer++
							octet := check_type([]byte{data.Content[pointer]})
							if octet == 1 {
								length = check_length([]byte{data.Content[pointer]})
							} else {
								length = check_length(data.Content[pointer : pointer+octet])
							}
							pointer += octet
							innack = get_data(data.Content[pointer : pointer+length])
							// fmt.Println("innack: ", innack)
							pointer += length
						} else if header := hex.EncodeToString([]byte{data.Content[pointer]}); header == "72" {
							// fmt.Println("data:", data)
							pointer++
							octet := check_type([]byte{data.Content[pointer]})
							if octet == 1 {
								length = check_length([]byte{data.Content[pointer]})
							} else {
								length = check_length(data.Content[pointer : pointer+octet])
							}
							pointer += octet
							uri = get_str_data(data.Content[pointer : pointer+length])
							// fmt.Println("innack: ", innack)
							pointer += length
						} else {
							pointer++
							octet := check_type([]byte{data.Content[pointer]})
							if octet == 1 {
								length = check_length([]byte{data.Content[pointer]})
							} else {
								length = check_length(data.Content[pointer : pointer+octet])
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

		//create route
		for k, v := range facelist {
			register_route(v.tkn, 0, int(k))

			fmt.Println(k, v.tkn)
			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("hello"), ndn.ForwardingHint{ndn.ParseName(v.tkn), ndn.ParseName("hello")})
			interest.MustBeFresh = true

			data, rtt, thg, e := consumer_interest(interest)

			if e != nil {
				continue
			}

			fmt.Println(data)

			v.ngb = data
			v.rtt = rtt
			v.thg = thg
			facelist[k] = v

		}
		fmt.Println(facelist)

		fl <- facelist

		time.Sleep(interval)
	}
}

func producer(name string, content string, fresh int) {
	payload := []byte(content)
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

	var signer ndn.Signer

	for {
		ctx := context.Background()
		p, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
			Prefix:      ndn.ParseName(name),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				// fmt.Println(interest)
				return ndn.MakeData(interest, payload, time.Duration(fresh)*time.Millisecond), nil
			},
			DataSigner: signer,
		})

		if e != nil {
			fmt.Println(e)
		}

		<-ctx.Done()
		defer p.Close()
	}
}

func producer_channel(name string, channel chan map[uint64]faces, fresh int) {
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
	log.Printf("prod")
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})

	var signer ndn.Signer

	for {
		ctx := context.Background()
		p, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
			Prefix:      ndn.ParseName(name),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				// fmt.Println(interest)
				content := <-channel
				jsonStr, err := json.Marshal(content)
				if err != nil {
					fmt.Println(err)
				}

				payload := []byte(jsonStr)
				fmt.Println("Producer receive")
				fmt.Println(payload)
				return ndn.MakeData(interest, payload, time.Duration(fresh)*time.Millisecond), nil
			},
			DataSigner: signer,
		})

		if e != nil {
			fmt.Println(e)
		}

		<-ctx.Done()
		defer p.Close()
	}
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
