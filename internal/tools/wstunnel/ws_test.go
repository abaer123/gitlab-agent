package wstunnel

import (
	"context"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nhooyr.io/websocket"
)

type testStuff struct {
	ctx        context.Context
	serverAddr net.Addr
	wrappedLis Listener
}

func TestClientServerVariousBufferSizes(t *testing.T) {
	t.Run("1kbyte", func(t *testing.T) {
		testHarness(t, func(t *testing.T, stuff *testStuff) {
			testEcho(t, 1024, 128, stuff)
		})
	})
	t.Run("64kbyte", func(t *testing.T) {
		testHarness(t, func(t *testing.T, stuff *testStuff) {
			testEcho(t, 64*1024, 128, stuff)
		})
	})
	t.Run("128kbyte", func(t *testing.T) {
		testHarness(t, func(t *testing.T, stuff *testStuff) {
			testEcho(t, 128*1024, 128, stuff)
		})
	})
}

func testEcho(t *testing.T, writeSize, writeCount int, stuff *testStuff) {
	var serverWg sync.WaitGroup
	defer serverWg.Wait()
	defer stuff.wrappedLis.Close()
	serverWg.Add(1)
	go func() {
		defer serverWg.Done()
		serverConn, err := stuff.wrappedLis.Accept()
		if !assert.NoError(t, err) {
			return
		}
		defer serverConn.Close()
		_, err = io.Copy(serverConn, serverConn) // echo
		assert.NoError(t, err)
	}()

	conn, _, err := Dial(stuff.ctx, fmt.Sprintf("ws://%s", stuff.serverAddr.String()), &websocket.DialOptions{})
	require.NoError(t, err)
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(1024 * 1024)

	// Read and hash data
	var clientWg sync.WaitGroup
	readHash := fnv.New128()
	clientWg.Add(1)
	go func() {
		defer clientWg.Done()
		toRead := int64(writeSize * writeCount)
		netConn := websocket.NetConn(stuff.ctx, conn, websocket.MessageBinary)
		copied, err := io.Copy(readHash, io.LimitReader(netConn, toRead))
		if assert.NoError(t, err) {
			assert.Equal(t, toRead, copied)
		}
	}()

	// Generate, hash and write random data
	writeHash := fnv.New128()
	buf := make([]byte, writeSize)
	for i := 0; i < writeCount; i++ {
		rand.Read(buf)
		writeHash.Write(buf)
		connErr := conn.Write(stuff.ctx, websocket.MessageBinary, buf)
		if !assert.NoError(t, connErr) {
			break
		}
	}

	clientWg.Wait() // wait for client to be done

	assert.Equal(t, writeHash.Sum(nil), readHash.Sum(nil))
}

func testHarness(t *testing.T, test func(*testing.T, *testStuff)) {
	lis, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer lis.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wrapper := ListenerWrapper{
		ReadLimit: 1024 * 1024,
	}
	wrappedLis := wrapper.Wrap(lis)
	ts := &testStuff{
		ctx:        ctx,
		serverAddr: lis.Addr(),
		wrappedLis: wrappedLis,
	}
	defer func() {
		assert.NoError(t, wrappedLis.Close())       // stop accepting connections
		assert.NoError(t, wrappedLis.Shutdown(ctx)) // wait for running connections
	}()
	test(t, ts)
}
