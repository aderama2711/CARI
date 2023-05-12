package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/eric135/go-ndn"
	"github.com/eric135/go-ndn/endpoint"
)

func main() {
	for {
		payload := make([]byte, 1024)
		rand.New(rand.NewSource(rand.Int63())).Read(payload)

		p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
			Prefix:      go-ndn.ParseName("/ndn/coba"),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				fmt.Print(interest)
				return go-ndn.MakeData(interest, payload), nil
			},
		})

		if e != nil {
			fmt.Print(e)
		}

		defer p.Close()
	}

}
