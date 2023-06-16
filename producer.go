package main

import (
	"context"
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func producer(name string, content string, fresh int) {
	openUplink()
	payload := []byte(content)

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
