package device

import (
	"bytes"
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mocks "github.com/netbirdio/netbird/client/iface/mocks"
)

func TestDeviceWrapperRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("read ICMP", func(t *testing.T) {
		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    net.IP{192, 168, 0, 1},
			DstIP:    net.IP{100, 200, 0, 1},
		}

		icmpLayer := &layers.ICMPv4{
			TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0),
			Id:       1,
			Seq:      1,
		}

		buffer := gopacket.NewSerializeBuffer()
		err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{},
			ipLayer,
			icmpLayer,
		)
		if err != nil {
			t.Errorf("serialize packet: %v", err)
			return
		}

		mockBufs := [][]byte{{}}
		mockSizes := []int{0}
		mockOffset := 0

		tun := mocks.NewMockDevice(ctrl)
		tun.EXPECT().Read(mockBufs, mockSizes, mockOffset).
			DoAndReturn(func(bufs [][]byte, sizes []int, offset int) (int, error) {
				bufs[0] = buffer.Bytes()
				sizes[0] = len(bufs[0])
				return 1, nil
			})

		wrapped := newDeviceFilter(tun)

		bufs := [][]byte{{}}
		sizes := []int{0}
		offset := 0

		n, err := wrapped.Read(bufs, sizes, offset)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if n != 1 {
			t.Errorf("expected n=1, got %d", n)
			return
		}
	})

	t.Run("write TCP", func(t *testing.T) {
		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    net.IP{100, 200, 0, 9},
			DstIP:    net.IP{100, 200, 0, 10},
		}

		// create TCP layer packet
		tcpLayer := &layers.TCP{
			SrcPort: layers.TCPPort(34423),
			DstPort: layers.TCPPort(8080),
		}

		buffer := gopacket.NewSerializeBuffer()
		err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{},
			ipLayer,
			tcpLayer,
		)
		if err != nil {
			t.Errorf("serialize packet: %v", err)
			return
		}

		mockBufs := [][]byte{buffer.Bytes()}

		mockBufs[0] = buffer.Bytes()
		tun := mocks.NewMockDevice(ctrl)
		tun.EXPECT().Write(mockBufs, 0).Return(1, nil)

		wrapped := newDeviceFilter(tun)

		bufs := [][]byte{buffer.Bytes()}

		n, err := wrapped.Write(bufs, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if n != 1 {
			t.Errorf("expected n=1, got %d", n)
			return
		}
	})

	t.Run("drop write UDP package", func(t *testing.T) {
		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    net.IP{100, 200, 0, 11},
			DstIP:    net.IP{100, 200, 0, 20},
		}

		// create TCP layer packet
		tcpLayer := &layers.UDP{
			SrcPort: layers.UDPPort(27278),
			DstPort: layers.UDPPort(53),
		}

		buffer := gopacket.NewSerializeBuffer()
		err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{},
			ipLayer,
			tcpLayer,
		)
		if err != nil {
			t.Errorf("serialize packet: %v", err)
			return
		}

		mockBufs := [][]byte{}

		tun := mocks.NewMockDevice(ctrl)
		tun.EXPECT().Write(mockBufs, 0).Return(0, nil)

		filter := mocks.NewMockPacketFilter(ctrl)
		filter.EXPECT().FilterInbound(gomock.Any(), gomock.Any()).Return(true)

		wrapped := newDeviceFilter(tun)
		wrapped.filter = filter

		bufs := [][]byte{buffer.Bytes()}

		n, err := wrapped.Write(bufs, 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if n != 1 {
			t.Errorf("expected n=1, got %d", n)
			return
		}
	})

	t.Run("drop read UDP package", func(t *testing.T) {
		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      64,
			Protocol: layers.IPProtocolICMPv4,
			SrcIP:    net.IP{100, 200, 0, 11},
			DstIP:    net.IP{100, 200, 0, 20},
		}

		// create TCP layer packet
		tcpLayer := &layers.UDP{
			SrcPort: layers.UDPPort(19243),
			DstPort: layers.UDPPort(1024),
		}

		buffer := gopacket.NewSerializeBuffer()
		err := gopacket.SerializeLayers(buffer, gopacket.SerializeOptions{},
			ipLayer,
			tcpLayer,
		)
		if err != nil {
			t.Errorf("serialize packet: %v", err)
			return
		}

		mockBufs := [][]byte{{}}
		mockSizes := []int{0}
		mockOffset := 0

		tun := mocks.NewMockDevice(ctrl)
		tun.EXPECT().Read(mockBufs, mockSizes, mockOffset).
			DoAndReturn(func(bufs [][]byte, sizes []int, offset int) (int, error) {
				bufs[0] = buffer.Bytes()
				sizes[0] = len(bufs[0])
				return 1, nil
			})
		filter := mocks.NewMockPacketFilter(ctrl)
		filter.EXPECT().FilterOutbound(gomock.Any(), gomock.Any()).Return(true)

		wrapped := newDeviceFilter(tun)
		wrapped.filter = filter

		bufs := [][]byte{{}}
		sizes := []int{0}
		offset := 0

		n, err := wrapped.Read(bufs, sizes, offset)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if n != 1 {
			t.Errorf("expected n=1, got %d", n)
			return
		}
		if sizes[0] != 0 {
			t.Errorf("expected dropped packet size 0, got %d", sizes[0])
			return
		}
	})
}

func TestDeviceWrapperReadPanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tun := mocks.NewMockDevice(ctrl)
	tun.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(bufs [][]byte, sizes []int, offset int) (int, error) {
			// Reproduce the wintun zero-length packet panic (index out of range).
			packet := make([]byte, 0)
			return int(packet[0]), nil
		})

	wrapped := newDeviceFilter(tun)

	handlerCalled := false
	wrapped.SetPanicHandler(func() { handlerCalled = true })

	n, err := wrapped.Read([][]byte{{}}, []int{0}, 0)
	if err == nil {
		t.Errorf("expected error from recovered panic, got nil")
	}
	if n != 0 {
		t.Errorf("expected n=0, got %d", n)
	}
	if !handlerCalled {
		t.Errorf("expected panic handler to be called")
	}
}

func TestDeviceWrapperWritePanic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tun := mocks.NewMockDevice(ctrl)
	tun.EXPECT().Write(gomock.Any(), gomock.Any()).
		DoAndReturn(func(bufs [][]byte, offset int) (int, error) {
			packet := make([]byte, 0)
			return int(packet[0]), nil
		})

	wrapped := newDeviceFilter(tun)

	handlerCalled := false
	wrapped.SetPanicHandler(func() { handlerCalled = true })

	n, err := wrapped.Write([][]byte{{0x45, 0x00}}, 0)
	if err == nil {
		t.Errorf("expected error from recovered panic, got nil")
	}
	if n != 0 {
		t.Errorf("expected n=0, got %d", n)
	}
	if !handlerCalled {
		t.Errorf("expected panic handler to be called")
	}
}

type recordingCapture struct {
	packets [][]byte
}

func (c *recordingCapture) Offer(data []byte, outbound bool) {
	if outbound {
		c.packets = append(c.packets, append([]byte(nil), data...))
	}
}

// TestDeviceWrapperReadDropKeepsBuffers checks that dropping an outbound
// packet does not reorder the caller's bufs/sizes. wireguard-go keeps
// bufs[i] pointing at the buffer of its own element i across reads, so
// compacting the slices in place leaves later slots pointing at a neighbour's
// buffer for every following read.
func TestDeviceWrapperReadDropKeepsBuffers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	const (
		count   = 4
		offset  = 16
		dropIdx = 1
	)

	packets := make([][]byte, count)
	for i := range packets {
		packets[i] = bytes.Repeat([]byte{byte(0x10 + i)}, 20+i)
	}

	backing := make([][]byte, count)
	bufs := make([][]byte, count)
	sizes := make([]int, count)
	for i := range backing {
		backing[i] = make([]byte, 128)
		bufs[i] = backing[i]
	}

	tun := mocks.NewMockDevice(ctrl)
	tun.EXPECT().Read(gomock.Any(), gomock.Any(), offset).
		DoAndReturn(func(bufs [][]byte, sizes []int, offset int) (int, error) {
			for i, p := range packets {
				sizes[i] = copy(bufs[i][offset:], p)
			}
			return count, nil
		})

	filter := mocks.NewMockPacketFilter(ctrl)
	filter.EXPECT().FilterOutbound(gomock.Any(), gomock.Any()).
		DoAndReturn(func(packet []byte, size int) bool {
			return packet[0] == packets[dropIdx][0]
		}).Times(count)

	capture := &recordingCapture{}
	wrapped := newDeviceFilter(tun)
	wrapped.SetFilter(filter)
	wrapped.SetCapture(capture)

	n, err := wrapped.Read(bufs, sizes, offset)
	require.NoError(t, err)
	assert.Equal(t, count, n, "n must cover every slot so the caller visits each one")

	for i := range bufs {
		assert.Same(t, &backing[i][0], &bufs[i][0], "bufs[%d] must still point at its own buffer", i)
	}

	assert.Equal(t, 0, sizes[dropIdx], "dropped packet must be marked with size 0")

	var kept [][]byte
	for i, p := range packets {
		if i == dropIdx {
			continue
		}
		assert.Equal(t, len(p), sizes[i], "sizes[%d]", i)
		assert.Equal(t, p, backing[i][offset:offset+len(p)], "packet %d must be in its own buffer", i)
		kept = append(kept, p)
	}

	assert.Equal(t, kept, capture.packets, "capture must see exactly the packets that were not dropped")
}
