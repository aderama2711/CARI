package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func main() {
	producer()
}

func producer() {
	openUplink()
	payload := make([]byte, 1024)
	rand.New(rand.NewSource(rand.Int63())).Read(payload)

	var signer ndn.Signer

	for {
		ctx := context.Background()
		p, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
			Prefix:      ndn.ParseName("/ndn/coba"),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				fmt.Println(interest)
				return ndn.MakeData(interest, payload), nil
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
