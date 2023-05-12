package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func main() {
	payload := make([]byte, 1024)
	rand.New(rand.NewSource(rand.Int63())).Read(payload)

	_, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:      ndn.ParseName("/ndn/coba"),
		NoAdvertise: false,
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			fmt.Print(interest)
			return ndn.MakeData(interest, payload), nil
		},
	})
	if e != nil {
		fmt.Print(e)
	}
}
