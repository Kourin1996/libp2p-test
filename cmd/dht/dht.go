package main

import (
	"context"
	"fmt"

	"github.com/ipfs/go-ipns"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
)

func NewDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT, error) {
	nsValidator := record.NamespacedValidator{}
	nsValidator["ipns"] = ipns.Validator{}
	nsValidator["pk"] = record.PublicKeyValidator{}

	dht, err := dht.New(ctx, host, dht.Mode(dht.ModeServer), dht.Validator(nsValidator), dht.BootstrapPeers(), dht.DisableAutoRefresh(), dht.QueryFilter(func(dht *dht.IpfsDHT, ai peer.AddrInfo) bool {
		connected := dht.Host().Network().Connectedness(ai.ID) == network.Connected
		fmt.Printf("QueryFilter: %+v => %t\n", ai, connected)
		return connected
	}))
	if err != nil {
		return nil, err
	} else if err = dht.Bootstrap(ctx); err != nil {
		return nil, err
	}

	return dht, nil
}
