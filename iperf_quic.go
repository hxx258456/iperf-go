package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/quic-go/quic-go"
)

type quic_proto struct {
	clientConn *quic.Conn // Store client QUIC connection for stream multiplexing
}

func (q *quic_proto) name() string {
	return QUIC_NAME
}

// generateTLSConfig creates a simple TLS configuration for QUIC
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Errorf("Failed to generate RSA key: %v", err)
		return nil
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour * 24 * 365),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Errorf("Failed to create certificate: %v", err)
		return nil
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Errorf("Failed to load X509 key pair: %v", err)
		return nil
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"iperf-quic"},
		InsecureSkipVerify: true,
	}
}

type quicListener struct {
	listener *quic.Listener
	conn     *quic.Conn // Store the accepted QUIC connection for stream multiplexing
}

func (ql *quicListener) Accept() (net.Conn, error) {
	log.Debugf("quicListener Accept: waiting for QUIC connection or stream...")

	// If we don't have a QUIC connection yet, accept one
	if ql.conn == nil {
		log.Debugf("quicListener Accept: no existing connection, accepting new one...")
		conn, err := ql.listener.Accept(context.Background())
		if err != nil {
			log.Errorf("quicListener Accept: listener.Accept failed: %v", err)
			return nil, err
		}
		ql.conn = conn
		log.Debugf("quicListener Accept: accepted new QUIC connection from %v", conn.RemoteAddr())
	}

	// Accept a stream on the existing connection
	log.Debugf("quicListener Accept: accepting stream on existing connection...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := ql.conn.AcceptStream(ctx)
	if err != nil {
		log.Errorf("quicListener Accept: AcceptStream failed: %v", err)
		return nil, err
	}
	log.Debugf("quicListener Accept: got stream ID %d, success", stream.StreamID())

	// Read and verify handshake byte
	handshakeBuf := make([]byte, 1)
	_, err = io.ReadFull(stream, handshakeBuf)
	if err != nil {
		log.Errorf("quicListener Accept: failed to read handshake: %v", err)
		stream.Close()
		return nil, err
	}
	if handshakeBuf[0] != 0xFF {
		log.Errorf("quicListener Accept: invalid handshake byte: 0x%02x", handshakeBuf[0])
		stream.Close()
		return nil, io.ErrUnexpectedEOF
	}
	log.Debugf("quicListener Accept: handshake verified")

	return &quicStreamConn{stream: stream, conn: ql.conn}, nil
}

func (ql *quicListener) Close() error {
	return ql.listener.Close()
}

func (ql *quicListener) Addr() net.Addr {
	return ql.listener.Addr()
}

type quicStreamConn struct {
	stream *quic.Stream
	conn   *quic.Conn
}

func (qc *quicStreamConn) Read(b []byte) (int, error) {
	return qc.stream.Read(b)
}

func (qc *quicStreamConn) Write(b []byte) (int, error) {
	return qc.stream.Write(b)
}

func (qc *quicStreamConn) Close() error {
	// Only close the stream, not the connection (connection is reused for multiplexing)
	log.Debugf("quicStreamConn Close: closing stream ID %d", qc.stream.StreamID())
	return qc.stream.Close()
}

func (qc *quicStreamConn) LocalAddr() net.Addr {
	return qc.conn.LocalAddr()
}

func (qc *quicStreamConn) RemoteAddr() net.Addr {
	return qc.conn.RemoteAddr()
}

func (qc *quicStreamConn) SetDeadline(t time.Time) error {
	return qc.stream.SetDeadline(t)
}

func (qc *quicStreamConn) SetReadDeadline(t time.Time) error {
	return qc.stream.SetReadDeadline(t)
}

func (qc *quicStreamConn) SetWriteDeadline(t time.Time) error {
	return qc.stream.SetWriteDeadline(t)
}

func (q *quic_proto) listen(test *iperf_test) (net.Listener, error) {
	tlsConf := generateTLSConfig()
	if tlsConf == nil {
		return nil, os.ErrInvalid
	}

	addr := ":" + strconv.Itoa(int(test.port))
	listener, err := quic.ListenAddr(addr, tlsConf, nil)
	if err != nil {
		return nil, err
	}

	return &quicListener{listener: listener}, nil
}

func (q *quic_proto) accept(test *iperf_test) (net.Conn, error) {
	log.Debugf("Enter QUIC accept")
	conn, err := test.proto_listener.Accept()
	if err != nil {
		log.Errorf("QUIC accept failed: %v", err)
		return nil, err
	}
	log.Debugf("QUIC accept succeed")
	return conn, nil
}

func (qp *quic_proto) connect(test *iperf_test) (net.Conn, error) {
	// If we don't have a QUIC connection yet, create one
	if qp.clientConn == nil {
		log.Debugf("QUIC connect: creating new QUIC connection...")
		tlsConf := generateTLSConfig()
		if tlsConf == nil {
			return nil, os.ErrInvalid
		}

		addr := test.addr + ":" + strconv.Itoa(int(test.port))
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		conn, err := quic.DialAddr(ctx, addr, tlsConf, nil)
		if err != nil {
			log.Errorf("QUIC connect: DialAddr failed: %v", err)
			return nil, err
		}
		qp.clientConn = conn
		log.Debugf("QUIC connect: new connection established to %v", conn.RemoteAddr())
	}

	// Open a stream on the existing connection
	log.Debugf("QUIC connect: opening stream...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := qp.clientConn.OpenStreamSync(ctx)
	if err != nil {
		log.Errorf("QUIC connect: OpenStreamSync failed: %v", err)
		qp.clientConn.CloseWithError(0, "failed to open stream")
		return nil, err
	}
	log.Debugf("QUIC connect: stream ID %d opened", stream.StreamID())

	// Send handshake byte to trigger AcceptStream on server
	handshakeBuf := []byte{0xFF}
	_, err = stream.Write(handshakeBuf)
	if err != nil {
		log.Errorf("QUIC connect: failed to send handshake: %v", err)
		stream.Close()
		return nil, err
	}
	log.Debugf("QUIC connect: handshake sent")

	log.Debugf("QUIC connect succeed")
	return &quicStreamConn{stream: stream, conn: qp.clientConn}, nil
}

func (q *quic_proto) send(sp *iperf_stream) int {
	n, err := sp.conn.Write(sp.buffer)
	if err != nil {
		if err == io.EOF || err == os.ErrClosed || err == io.ErrClosedPipe {
			log.Debugf("send QUIC socket close")
			return -1
		}
		log.Errorf("QUIC write err = %T %v", err, err)
		return -2
	}
	if n < 0 {
		log.Errorf("QUIC write err. n = %v", n)
		return n
	}
	sp.result.bytes_sent += uint64(n)
	sp.result.bytes_sent_this_interval += uint64(n)
	return n
}

func (q *quic_proto) recv(sp *iperf_stream) int {
	n, err := sp.conn.Read(sp.buffer)

	if err != nil {
		if err == io.EOF || err == os.ErrClosed || err == io.ErrClosedPipe {
			log.Debugf("recv QUIC socket close. EOF")
			return -1
		}
		log.Errorf("QUIC recv err = %T %v", err, err)
		return -2
	}
	if n < 0 {
		return n
	}
	if sp.test.state == TEST_RUNNING {
		sp.result.bytes_received += uint64(n)
		sp.result.bytes_received_this_interval += uint64(n)
	}
	return n
}

func (q *quic_proto) init(test *iperf_test) int {
	// QUIC-specific initialization if needed
	for _, sp := range test.streams {
		sp.conn.SetDeadline(time.Now().Add(time.Duration(test.duration+5) * time.Second))
	}
	return 0
}

func (q *quic_proto) stats_callback(test *iperf_test, sp *iperf_stream, temp_result *iperf_interval_results) int {
	rp := sp.result

	// QUIC uses simplified statistics similar to TCP
	// RTT estimation from interval duration
	if temp_result.interval_dur.Milliseconds() > 0 {
		temp_result.rtt = uint(temp_result.interval_dur.Microseconds() / 10) // rough estimation
	}

	if rp.stream_min_rtt == 0 || temp_result.rtt < rp.stream_min_rtt {
		rp.stream_min_rtt = temp_result.rtt
	}
	if rp.stream_max_rtt == 0 || temp_result.rtt > rp.stream_max_rtt {
		rp.stream_max_rtt = temp_result.rtt
	}
	rp.stream_sum_rtt += temp_result.rtt
	rp.stream_cnt_rtt++

	return 0
}

func (qp *quic_proto) teardown(test *iperf_test) int {
	log.Debugf("QUIC teardown: cleaning up resources...")

	// Close client QUIC connection if it exists
	if qp.clientConn != nil {
		log.Debugf("QUIC teardown: closing client connection...")
		err := qp.clientConn.CloseWithError(0, "test completed")
		if err != nil {
			log.Debugf("QUIC teardown: error closing client connection: %v", err)
		}
		qp.clientConn = nil
		log.Debugf("QUIC teardown: client connection closed")
	}

	// Close server listener connection if this is a server
	if test.is_server && test.proto_listener != nil {
		if ql, ok := test.proto_listener.(*quicListener); ok && ql.conn != nil {
			log.Debugf("QUIC teardown: closing server connection...")
			err := ql.conn.CloseWithError(0, "test completed")
			if err != nil {
				log.Debugf("QUIC teardown: error closing server connection: %v", err)
			}
			ql.conn = nil
			log.Debugf("QUIC teardown: server connection closed")
		}
	}

	log.Debugf("QUIC teardown: complete")
	return 0
}
