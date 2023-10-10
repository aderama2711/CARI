package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/RyanCarrier/dijkstra"
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

type neighbor struct {
	Cst int64 `json:"Cst"`
	Fce int   `json:"Fce"`
}

// facelist for neighbor info
var facelist map[uint64]faces

// prefixlist for saving producer of prefixes
var prefixlist map[int][]string

var network map[int]map[int]neighbor

var mutex sync.Mutex

func main() {
	facelist = make(map[uint64]faces)
	prefixlist = make(map[int][]string)
	network = map[int]map[int]neighbor{}

	var wg sync.WaitGroup

	wg.Add(2)

	// hello procedure then asking route info
	go consumer_helloandinfo(&wg)

	// serve prefix registration
	go producer_prefix(&wg)

	wg.Wait()

}

func consumer_helloandinfo(wg *sync.WaitGroup) {
	// hello protocol every 5 second
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

	interval := 30 * time.Second
	for {
		//update facelist
		update_facelist()
		log.Println("Facelist : ", facelist)

		log.Println("===== Hello Procedure =====")
		//create route
		for k, v := range facelist {
			register_route(v.Tkn, 0, int(k))

			log.Print(k, v.Tkn)
			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("hello"), ndn.ForwardingHint{ndn.ParseName(v.Tkn), ndn.ParseName("hello")})
			interest.MustBeFresh = true

			data, rtt, thg, e := consumer_interest(interest)

			if e != nil {
				log.Println("Error occured : ", e)
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
			v.Rtt = rtt
			v.Thg = thg
			facelist[k] = v

			time.Sleep(500 * time.Millisecond)
		}

		log.Println("Updated Facelist : ", facelist)

		log.Println("===== Request Route Info =====")
		//request route info
		for k, v := range facelist {

			log.Print(k, v.Tkn)

			//request route info interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("info"), ndn.ForwardingHint{ndn.ParseName(v.Tkn), ndn.ParseName("info")})
			interest.MustBeFresh = true

			data, _, _, e := consumer_interest(interest)

			if e != nil {
				continue
			}

			log.Println(" route info : \n", data)

			// Create temp map for json string -> map
			var temp_fl map[uint64]faces

			err := json.Unmarshal([]byte(data), &temp_fl)
			if err != nil {
				log.Println("Error: ", err)
			}

			// convert facelist to network map
			var temp map[int]neighbor
			temp = make(map[int]neighbor)
			for key, value := range temp_fl {
				// cost := value.Rtt + (value.Thg * -1) + (float64(value.N_oi) / float64(value.N_in))
				cost := uint64(0)
				if value.N_oi != 0 {
					cost = uint64(value.Thg+value.Rtt) / (1 - uint64(value.N_in/value.N_oi))
					temp[value.Ngb] = neighbor{Cst: int64(cost), Fce: int(key)}
				}

			}
			mutex.Lock()
			network[v.Ngb] = temp
			mutex.Unlock()

			time.Sleep(500 * time.Millisecond)
		}

		go recalculate_route()

		log.Println("Registered prefix list : ", prefixlist)

		time.Sleep(interval)
	}

}

func recalculate_route() {
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

	// Insert neighbor data to graph for calculating the route using dijkstra library
	var temp_network map[int]map[int]neighbor
	var temp_prefixlist map[int][]string
	var temp_facelist map[uint64]faces

	mutex.Lock()
	temp_network = network
	temp_prefixlist = prefixlist
	temp_facelist = facelist
	mutex.Unlock()

	graph := dijkstra.NewGraph()

	log.Println("===== Calculating Routes =====")
	log.Println("Network : \n", temp_network)
	log.Println("Prefix : \n", temp_prefixlist)
	log.Println("Face : \n", temp_facelist)

	// Add vertex
	for key, _ := range temp_network {
		graph.AddVertex(key)
	}

	var validate []string

	// Iterate over the network map using a for range loop to create vertices
	for key, _ := range temp_network {
		for keys, _ := range temp_network[key] {
			if keys == 99116 || key == 99116 {
				continue
			}
			if key == keys {
				continue
			}
			// if (strings.Contains(strings.Join(validate, "-"), fmt.Sprintf("%d, %d", key, keys))) || strings.Contains(strings.Join(validate, "-"), fmt.Sprintf("%d, %d", keys, key)) {
			// 	continue
			// }
			log.Println("Add connection", key, keys, "cost", (temp_network[key][keys].Cst + temp_network[keys][key].Cst))
			graph.AddArc(key, keys, (temp_network[key][keys].Cst + temp_network[keys][key].Cst))
			validate = append(validate, fmt.Sprintf("%d, %d", key, keys))
		}
	}

	// Iterate over the prefixlist using a for range loop to calculate every node to producer with prefix
	for prod, _ := range temp_prefixlist {
		for cons, _ := range network {
			if cons == 99116 {
				continue
			}
			// if node is producer, skip
			if cons == prod {
				continue
			} else {
				log.Println("Calculate routes", cons, "to", prod)
				// Search the best path
				best, err := graph.ShortestSafe(cons, prod)
				if err != nil {
					log.Println("Error occured : ", err)
				} else {
					log.Println("Shortest distance ", cons, prod, best.Distance, " following path ", best.Path)

					router := uint64(0)

					for key, value := range temp_facelist {
						if value.Ngb == cons {
							router = key
						}
					}

					// Install prefix and list
					for _, prefix := range temp_prefixlist[prod] {
						log.Println("Installing routes : ", cons, prefix, network[cons][best.Path[1]].Fce)

						// update route
						interest := ndn.MakeInterest(ndn.ParseName("update"), []byte(fmt.Sprintf("%s,%d,%d", prefix, cons, network[cons][best.Path[1]].Fce)), ndn.ForwardingHint{ndn.ParseName(temp_facelist[router].Tkn), ndn.ParseName("update")})
						interest.MustBeFresh = true
						interest.UpdateParamsDigest() //Update SHA256 params

						data, _, _, err := consumer_interest(interest)

						if err != nil {
							log.Println("Error occured : ", err)
							continue
						}

						log.Println(data)
					}
				}

				// Search the longest path
				best, err = graph.LongestSafe(cons, prod)
				if err != nil {
					log.Println(err)
				} else {
					log.Println("Longest distance ", cons, prod, best.Distance, " following path ", best.Path)

					// Install prefix and list
					for _, prefix := range temp_prefixlist[prod] {
						log.Println("Installing routes : ", cons, prefix, network[cons][best.Path[1]].Fce)

						// update route
						interest := ndn.MakeInterest(ndn.ParseName("update"), []byte(fmt.Sprintf("%s,%d,%d", prefix, cons, network[0][best.Path[1]].Fce)), ndn.ForwardingHint{ndn.ParseName(temp_facelist[uint64(cons)].Tkn), ndn.ParseName("update")})
						interest.MustBeFresh = true
						interest.UpdateParamsDigest() //Update SHA256 params

						data, _, _, err := consumer_interest(interest)

						if err != nil {
							continue
						}

						log.Println(data)
					}
				}
			}
		}
	}

	client.Close()

}

func producer_prefix(wg *sync.WaitGroup) {
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
			Prefix:      ndn.ParseName("/prefix"),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				// Get App Param
				log.Println("Payload = " + string(interest.AppParameters))
				splits := strings.Split(string(interest.AppParameters), ",")
				prod, _ := strconv.Atoi(splits[0])
				prefix := splits[1]

				// Update prefixlist
				mutex.Lock()
				prefixlist[prod] = append(prefixlist[prod], prefix)
				// fmt.Println(prefixlist)
				mutex.Unlock()

				go recalculate_route()

				payload := []byte(string(interest.AppParameters))
				return ndn.MakeData(interest, payload, time.Duration(10)*time.Millisecond), nil
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

// func producer_info(name string, fresh int, wg *sync.WaitGroup) {
// 	defer wg.Done()
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
// 				content, err := json.Marshal(facelist)
// 				if err != nil {
// 					log.Printf(err.Error())
// 				}
// 				payload := []byte(string(content))
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

// func producer_update(name string, fresh int, wg *sync.WaitGroup) {
// 	defer wg.Done()
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
// 				// Get App Param
// 				log.Println("Payload = " + string(interest.AppParameters))
// 				splits := strings.Split(string(interest.AppParameters), ",")
// 				cost, _ := strconv.Atoi(splits[1])
// 				face, _ := strconv.Atoi(splits[2])
// 				register_route(splits[0], cost, face)
// 				payload := []byte(string(interest.AppParameters))
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
