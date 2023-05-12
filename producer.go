package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/urfave/cli/v2"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func main() {
	func(c *cli.Context) error {
		payload := make([]byte, 1024)
		rand.New(rand.NewSource(rand.Int63())).Read(payload)

		_, e := endpoint.Produce(c.Context, endpoint.ProducerOptions{
			Prefix:      ndn.ParseName("/ndn/coba"),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				fmt.Print(interest)
				return ndn.MakeData(interest, payload), nil
			},
		})

		if e != nil {
			fmt.Println(e)
			return e
		}

		<-c.Context.Done()
		return nil
	}
}
