package main

import (
	"time"
)

func main() {
	// var (
	// 	client mgmt.Client
	// 	face   mgmt.Face
	// 	fwFace l3.FwFace
	// )

	// face, e := client.OpenFace()

	// if e != nil {
	// 	fmt.Println(client, e)
	// }
	// l3face := face.Face()

	// fw := l3.GetDefaultForwarder()
	// if fwFace, e = fw.AddFace(l3face); e != nil {
	// 	fmt.Println(client, e)
	// }
	// fwFace.AddRoute(ndn.Name{})
	// fw.AddReadvertiseDestination(face)

	// log.Printf("uplink opened, state is %s", l3face.State())
	// l3face.OnStateChange(func(st l3.TransportState) {
	// 	log.Printf("uplink state changes to %s", l3face.State())
	// })

	// c, e := nfdmgmt.New()

	// var sigNonce [8]byte
	// rand.Read(sigNonce[:])

	// interest := ndn.Interest{
	// 	Name:        ndn.ParseName("/localhost/nfd/faces/list"),
	// 	MustBeFresh: true,
	// 	SigInfo: &ndn.SigInfo{
	// 		Nonce: sigNonce[:],
	// 		Time:  uint64(time.Now().UnixMilli()),
	// 	},
	// }

	// c.Signer.Sign(&interest)

	// data, _ := endpoint.Consume(context.Background(), interest,
	// 	endpoint.ConsumerOptions{})

	// fmt.Println(data.Content)

	consumer("/ndn/coba")

	// //Serve /hello interest
	// go serve_hello("R1")

	// //hello protocol every 5 second
	// go consum_hello(5)
}

func serve_hello(router string) {
	producer("/hello", router)
}

func consum_hello(delay int) {
	interval := 5 * time.Second
	for {
		consumer("/hello")
		time.Sleep(interval)
	}
}
