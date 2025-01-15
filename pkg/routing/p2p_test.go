package routing

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	tlog "github.com/go-logr/logr/testing"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

func TestBootstrapFunc(t *testing.T) {
	t.Parallel()

	log := tlog.NewTestLogger(t)
	ctx := logr.NewContext(context.Background(), log)

	mn, err := mocknet.WithNPeers(2)
	require.NoError(t, err)

	tests := []struct {
		name     string
		peers    []peer.AddrInfo
		expected []string
	}{
		{
			name:     "no peers",
			peers:    []peer.AddrInfo{},
			expected: []string{},
		},
		{
			name: "nothing missing",
			peers: []peer.AddrInfo{
				{
					ID:    "foo",
					Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.1/tcp/8080")},
				},
			},
			expected: []string{"/ip4/192.168.1.1/tcp/8080/p2p/foo"},
		},
		{
			name: "only self",
			peers: []peer.AddrInfo{
				{
					ID:    mn.Hosts()[0].ID(),
					Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.1/tcp/8080")},
				},
			},
			expected: []string{},
		},
		{
			name: "missing port",
			peers: []peer.AddrInfo{
				{
					ID:    "foo",
					Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.1")},
				},
			},
			expected: []string{"/ip4/192.168.1.1/tcp/4242/p2p/foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b := NewStaticBootstrapper(tt.peers)
			f := bootstrapFunc(ctx, b, mn.Hosts()[0])
			peers := f()

			peerStrs := []string{}
			for _, p := range peers {
				id, err := p.ID.Marshal()
				require.NoError(t, err)
				peerStrs = append(peerStrs, fmt.Sprintf("%s/p2p/%s", p.Addrs[0].String(), string(id)))
			}
			require.ElementsMatch(t, tt.expected, peerStrs)
		})
	}
}

func TestListenMultiaddrs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		addr     string
		expected []string
	}{
		{
			name:     "listen address type not specified",
			addr:     ":9090",
			expected: []string{"/ip6/::/tcp/9090", "/ip4/0.0.0.0/tcp/9090"},
		},
		{
			name:     "ipv4 only",
			addr:     "0.0.0.0:9090",
			expected: []string{"/ip4/0.0.0.0/tcp/9090"},
		},
		{
			name:     "ipv6 only",
			addr:     "[::]:9090",
			expected: []string{"/ip6/::/tcp/9090"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			multiAddrs, err := listenMultiaddrs(tt.addr)
			require.NoError(t, err)
			require.Equal(t, len(tt.expected), len(multiAddrs))
			for i, e := range tt.expected {
				require.Equal(t, e, multiAddrs[i].String())
			}
		})
	}
}

func TestIsIp6(t *testing.T) {
	t.Parallel()

	m, err := ma.NewMultiaddr("/ip6/::")
	require.NoError(t, err)
	require.True(t, isIp6(m))
	m, err = ma.NewMultiaddr("/ip4/0.0.0.0")
	require.NoError(t, err)
	require.False(t, isIp6(m))
}

func TestCreateCid(t *testing.T) {
	t.Parallel()

	c, err := createCid("foobar")
	require.NoError(t, err)
	require.Equal(t, "bafkreigdvoh7cnza5cwzar65hfdgwpejotszfqx2ha6uuolaofgk54ge6i", c.String())
}

func TestHostMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		host     peer.AddrInfo
		addrInfo peer.AddrInfo
		expected bool
	}{
		{
			name: "ID match",
			host: peer.AddrInfo{
				ID:    "foo",
				Addrs: []ma.Multiaddr{},
			},
			addrInfo: peer.AddrInfo{
				ID:    "foo",
				Addrs: []ma.Multiaddr{},
			},
			expected: true,
		},
		{
			name: "ID do not match",
			host: peer.AddrInfo{
				ID:    "foo",
				Addrs: []ma.Multiaddr{},
			},
			addrInfo: peer.AddrInfo{
				ID:    "bar",
				Addrs: []ma.Multiaddr{},
			},
			expected: false,
		},
		{
			name: "IP4 match",
			host: peer.AddrInfo{
				ID:    "",
				Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.1")},
			},
			addrInfo: peer.AddrInfo{
				ID:    "",
				Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.1")},
			},
			expected: true,
		},
		{
			name: "IP4 do not match",
			host: peer.AddrInfo{
				ID:    "",
				Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.1")},
			},
			addrInfo: peer.AddrInfo{
				ID:    "",
				Addrs: []ma.Multiaddr{ma.StringCast("/ip4/192.168.1.2")},
			},
			expected: false,
		},
		{
			name: "IP6 match",
			host: peer.AddrInfo{
				ID:    "",
				Addrs: []ma.Multiaddr{ma.StringCast("/ip6/c3c9:152b:73d1:dad0:e2f9:a521:6356:88ba")},
			},
			addrInfo: peer.AddrInfo{
				ID:    "",
				Addrs: []ma.Multiaddr{ma.StringCast("/ip6/c3c9:152b:73d1:dad0:e2f9:a521:6356:88ba")},
			},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			matches, err := hostMatches(tt.host, tt.addrInfo)
			require.NoError(t, err)
			require.Equal(t, tt.expected, matches)
		})
	}
}
