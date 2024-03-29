package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

type faces struct {
	Ngb  int     `json:"Ngb"`
	Rtt  float64 `json:"Rtt"`
	Thg  float64 `json:"Thg"`
	Tkn  string  `json:"Tkn"`
	N_oi uint64  `json:"N_oi"`
	N_in uint64  `json:"N_in"`
}

var facelist map[uint64]faces

var mutex sync.Mutex

func main() {
	facelist = make(map[uint64]faces)

	var wg sync.WaitGroup

	wg.Add(3)

	// consumer for hello procedure (to neighbor)
	go consumer_hello(&wg)
	time.Sleep(500 * time.Millisecond)

	// producer for neighbor info (for controller)
	go producer_info("/info", 100, &wg)
	time.Sleep(500 * time.Millisecond)

	// producer for route update (for controller)
	go producer_update("/update", 100, &wg)
	time.Sleep(500 * time.Millisecond)
	wg.Wait()

}

func consumer_hello(wg *sync.WaitGroup) {
	// //hello protocol every 60 second
	defer wg.Done()
	var (
		client mgmt.Client
		face   mgmt.Face
		fwFace l3.FwFace
	)

	client, e := nfdmgmt.New()

	face, e = client.OpenFace()
	if e != nil {
		log.Println("Error occured : ", e)
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		log.Println("Error occured : ", e)
	}
	fwFace.AddRoute(ndn.Name{})
	fw.AddReadvertiseDestination(face)

	log.Printf("uplink opened, state is %s", l3face.State())
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})

	time.Sleep(1 * time.Second)

	interval := 60 * time.Second
	for {
		//update facelist
		update_facelist()
		log.Println("Facelist : ", facelist)

		log.Println("===== Hello Procedure =====")

		var recheck_facelist map[uint64]faces
		recheck_facelist = make(map[uint64]faces)

		//create route
		for k, v := range facelist {
			register_route(v.Tkn, 0, int(k))

			log.Print(k, v.Tkn)
			// send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("hello"), ndn.ForwardingHint{ndn.ParseName(v.Tkn), ndn.ParseName("hello")})
			interest.MustBeFresh = true

			log.Println("Sending Interest")
			log.Println(interest)
			data, Rtt, Thg, e := consumer_interest(interest)
			log.Println("The result are here")

			if e != nil {
				log.Println("Error occured : ", e)
				recheck_facelist[k] = v
				continue
			}
			data = strings.ReplaceAll(data, "A", "")

			// Define a regular expression to match digits
			reg := regexp.MustCompile("[0-9]+")

			// Find all matches in the input string
			matches := reg.FindAllString(data, -1)

			// Combine matches to get the numeric string
			numericString := ""
			for _, match := range matches {
				numericString += match
			}

			idata, err := strconv.Atoi(numericString)
			if err != nil {
				log.Printf("IMPOSIBLE!")
			}

			log.Println(" neighbor : ", idata)

			v.Ngb = idata
			v.Rtt = Rtt
			v.Thg = Thg
			facelist[k] = v

			time.Sleep(500 * time.Millisecond)

		}

		time.Sleep(1 * time.Second)

		log.Println("Hello Retries")

		// retries
		for i := 0; i < 2; i++ {
			if len(recheck_facelist) != 0 {
				for k, v := range recheck_facelist {
					register_route(v.Tkn, 0, int(k))

					log.Print(k, v.Tkn)
					// send hello interest to every face
					interest := ndn.MakeInterest(ndn.ParseName("hello"), ndn.ForwardingHint{ndn.ParseName(v.Tkn), ndn.ParseName("hello")})
					interest.MustBeFresh = true

					log.Println("Sending Interest")
					log.Println(interest)
					data, Rtt, Thg, e := consumer_interest(interest)
					log.Println("The result are here")

					if e != nil {
						log.Println("Error occured : ", e)
						recheck_facelist[k] = v
						v.Ngb = 0
						facelist[k] = v
						continue
					}
					data = strings.ReplaceAll(data, "A", "")

					// Define a regular expression to match digits
					reg := regexp.MustCompile("[0-9]+")

					// Find all matches in the input string
					matches := reg.FindAllString(data, -1)

					// Combine matches to get the numeric string
					numericString := ""
					for _, match := range matches {
						numericString += match
					}

					idata, err := strconv.Atoi(numericString)
					if err != nil {
						log.Printf("IMPOSIBLE!")
					}

					log.Println(" neighbor : ", idata)

					v.Ngb = idata
					v.Rtt = Rtt
					v.Thg = Thg
					facelist[k] = v

					delete(recheck_facelist, k)

					time.Sleep(500 * time.Millisecond)

				}
			} else {
				break
			}
			time.Sleep(1 * time.Second)
		}

		log.Println("Updated Facelist : ", facelist)

		time.Sleep(interval)
	}
}

// Commented for "future use"
// func producer(name string, content string, fresh int) {
// 	payload := []byte(content)
// 	var (
// 		client mgmt.Client
// 		face   mgmt.Face
// 		fwFace l3.FwFace
// 	)

// 	client, e := nfdmgmt.New()

// 	face, e = client.OpenFace()
// 	if e != nil {
// 		log.Println("Error occured : ", e)
// 	}
// 	l3face := face.Face()

// 	fw := l3.GetDefaultForwarder()
// 	if fwFace, e = fw.AddFace(l3face); e != nil {
// 		log.Println("Error occured : ", e)
// 	}
// 	fwFace.AddRoute(ndn.Name{})
// 	fw.AddReadvertiseDestination(face)

// 	log.Printf("uplink opened, state is %s", l3face.State())
// 	l3face.OnStateChange(func(st l3.TransportState) {
// 		log.Printf("uplink state changes to %s", l3face.State())
// 	})

// 	var signer ndn.Signer

// 	for {
// 		ctx := context.Background()
// 		p, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
// 			Prefix:      ndn.ParseName(name),
// 			NoAdvertise: false,
// 			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
// 				// fmt.Println(interest)
// 				return ndn.MakeData(interest, payload, time.Duration(fresh)*time.Millisecond), nil
// 			},
// 			DataSigner: signer,
// 		})

// 		if e != nil {
// 			log.Println("Error occured : ", e)
// 		}

// 		<-ctx.Done()
// 		defer p.Close()
// 	}
// }

func producer_info(name string, fresh int, wg *sync.WaitGroup) {
	defer wg.Done()
	var (
		client mgmt.Client
		face   mgmt.Face
		fwFace l3.FwFace
	)

	client, e := nfdmgmt.New()

	face, e = client.OpenFace()
	if e != nil {
		log.Println("Error occured : ", e)
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		log.Println("Error occured : ", e)
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
				content, err := json.Marshal(facelist)
				if err != nil {
					log.Printf(err.Error())
				}
				payload := []byte(string(content))
				return ndn.MakeData(interest, payload, time.Duration(fresh)*time.Millisecond), nil
			},
			DataSigner: signer,
		})

		if e != nil {
			log.Println("Error occured : ", e)
		}

		<-ctx.Done()
		defer p.Close()
	}
}

func producer_update(name string, fresh int, wg *sync.WaitGroup) {
	defer wg.Done()
	var (
		client mgmt.Client
		face   mgmt.Face
		fwFace l3.FwFace
	)

	client, e := nfdmgmt.New()

	face, e = client.OpenFace()
	if e != nil {
		log.Println("Error occured : ", e)
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		log.Println("Error occured : ", e)
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
				// Get App Param
				log.Println("Payload = " + string(interest.AppParameters))
				splits := strings.Split(string(interest.AppParameters), ",")
				cost, _ := strconv.Atoi(splits[1])
				face, _ := strconv.Atoi(splits[2])
				expired, _ := strconv.Atoi(splits[3])
				register_route_controller(splits[0], cost, face, expired)
				payload := []byte(string(interest.AppParameters))
				return ndn.MakeData(interest, payload, time.Duration(fresh)*time.Millisecond), nil
			},
			DataSigner: signer,
		})

		if e != nil {
			log.Println("Error occured : ", e)
		}

		<-ctx.Done()
		defer p.Close()
	}
}

func consumer_interest(Interest ndn.Interest) (content string, Rtt float64, Thg float64, e error) {
	// seqNum := rand.Uint64()
	// var nData, nErrors atomic.Int64

	t0 := time.Now()

	data, e := endpoint.Consume(context.Background(), Interest,
		endpoint.ConsumerOptions{})

	raw_Rtt := time.Since(t0)

	Rtt = float64(raw_Rtt / time.Millisecond)

	// fmt.Println(Rtt)

	if e == nil {
		// nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		// fmt.Println(data.Content)
		content = string(data.Content[:])
		// fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
		if Rtt != 0 {
			Thg = float64(len(content)) / float64(Rtt/1000)
		} else {
			Thg = 0
		}

	} else {
		// nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		// fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		return content, 0, 0, e
	}

	return content, Rtt, Thg, nil
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
		log.Println("Error occured : ", e)
	} else {
		parse_facelist(data.Content)
	}

	c.Close()
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
		log.Println("Error occured : ", e)
	}
	if cr.StatusCode != 200 {
		log.Println("unexpected response status %d", cr.StatusCode)
	} else {
		log.Println("Route registered")
	}
}

func register_route_controller(name string, cost int, faceid int, expired int) {
	c, _ := nfdmgmt.New()
	cr, e := c.Invoke(context.TODO(), nfdmgmt.RibRegisterCommand{
		Name:    ndn.ParseName(name),
		Origin:  0,
		Cost:    cost,
		FaceID:  faceid,
		Expires: nnduration.Milliseconds(expired),
	})
	if e != nil {
		log.Println("Error occured : ", e)
	}
	if cr.StatusCode != 200 {
		log.Println("unexpected response status %d", cr.StatusCode)
	} else {
		log.Println("Route registered")
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStriNgbytes(n int) string {
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

	log.Println("===== Update Facelist =====")
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
				} else if data := hex.EncodeToString([]byte{raw[pointer]}); data == "9a" {
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
			stoken := "/" + RandStriNgbytes(16)

			if strings.Contains(uri, ":6363") {
				log.Println("FaceId : ", uri)
				mutex.Lock()
				if _, ok := facelist[faceid]; ok {
					log.Println("Use existing")
					facelist[faceid] = faces{N_oi: outi, N_in: innack, Tkn: facelist[faceid].Tkn, Ngb: facelist[faceid].Ngb, Rtt: facelist[faceid].Rtt, Thg: facelist[faceid].Thg}
				} else {
					log.Println("Create new")
					facelist[faceid] = faces{N_oi: outi, N_in: innack, Tkn: stoken}
				}
				mutex.Unlock()
			} else {
				continue
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
