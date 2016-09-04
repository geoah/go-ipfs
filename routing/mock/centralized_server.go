package mockrouting

import (
	"math/rand"
	"sync"
	"time"

	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	key "github.com/ipfs/go-key"
	"github.com/libp2p/go-testutil"

	peer "github.com/ipfs/go-libp2p-peer"
	pstore "github.com/ipfs/go-libp2p-peerstore"
	context "golang.org/x/net/context"
)

// server is the mockrouting.Client's private interface to the routing server
type server interface {
	Announce(pstore.PeerInfo, key.Key) error
	Providers(key.Key) []pstore.PeerInfo

	Server
}

// s is an implementation of the private server interface
type s struct {
	delayConf DelayConfig

	lock      sync.RWMutex
	providers map[key.Key]map[peer.ID]providerRecord
}

type providerRecord struct {
	Peer    pstore.PeerInfo
	Created time.Time
}

func (rs *s) Announce(p pstore.PeerInfo, k key.Key) error {
	rs.lock.Lock()
	defer rs.lock.Unlock()

	_, ok := rs.providers[k]
	if !ok {
		rs.providers[k] = make(map[peer.ID]providerRecord)
	}
	rs.providers[k][p.ID] = providerRecord{
		Created: time.Now(),
		Peer:    p,
	}
	return nil
}

func (rs *s) Providers(k key.Key) []pstore.PeerInfo {
	rs.delayConf.Query.Wait() // before locking

	rs.lock.RLock()
	defer rs.lock.RUnlock()

	var ret []pstore.PeerInfo
	records, ok := rs.providers[k]
	if !ok {
		return ret
	}
	for _, r := range records {
		if time.Now().Sub(r.Created) > rs.delayConf.ValueVisibility.Get() {
			ret = append(ret, r.Peer)
		}
	}

	for i := range ret {
		j := rand.Intn(i + 1)
		ret[i], ret[j] = ret[j], ret[i]
	}

	return ret
}

func (rs *s) Client(p testutil.Identity) Client {
	return rs.ClientWithDatastore(context.Background(), p, dssync.MutexWrap(ds.NewMapDatastore()))
}

func (rs *s) ClientWithDatastore(_ context.Context, p testutil.Identity, datastore ds.Datastore) Client {
	return &client{
		peer:      p,
		datastore: datastore,
		server:    rs,
	}
}
