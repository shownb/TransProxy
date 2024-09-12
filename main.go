package main

import (
	"net"
	"os"
	"syscall"
	"unsafe"
	"flag"
	"io"
	"log"
	"net/url"
	"os/signal"
	"time"
	"golang.org/x/net/proxy"
)

var ListenAddr = flag.String("l", ":9040", "transproxy port for iptables")
var socks5 = flag.String("p", "socks5://192.168.1.210:1080", "socks5 proxy address")
var ioFlag= flag.Bool("d", false, "custom version of io.Copy without using splice system call")

const (
	// SO_ORIGINAL_DST is a Linux getsockopt optname.
	SO_ORIGINAL_DST = 80
	// IP6T_SO_ORIGINAL_DST a Linux getsockopt optname.
	IP6T_SO_ORIGINAL_DST = 80
)

func getsockopt(s int, level int, optname int, optval unsafe.Pointer, optlen *uint32) (err error) {
	_, _, e := syscall.Syscall6(
		syscall.SYS_GETSOCKOPT, uintptr(s), uintptr(level), uintptr(optname),
		uintptr(optval), uintptr(unsafe.Pointer(optlen)), 0)
	if e != 0 {
		return e
	}
	return
}

// GetOriginalDST retrieves the original destination address from
// NATed connection.  Currently, only Linux iptables using DNAT/REDIRECT
// is supported.  For other operating systems, this will just return
// conn.LocalAddr().
//
// Note that this function only works when nf_conntrack_ipv4 and/or
// nf_conntrack_ipv6 is loaded in the kernel.
func GetOriginalDST(conn *net.TCPConn) (*net.TCPAddr, error) {
	f, err := conn.File()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fd := int(f.Fd())
	// revert to non-blocking mode.
	// see http://stackoverflow.com/a/28968431/1493661
	if err = syscall.SetNonblock(fd, true); err != nil {
		return nil, os.NewSyscallError("setnonblock", err)
	}

	v6 := conn.LocalAddr().(*net.TCPAddr).IP.To4() == nil
	if v6 {
		var addr syscall.RawSockaddrInet6
		var len uint32
		len = uint32(unsafe.Sizeof(addr))
		err = getsockopt(fd, syscall.IPPROTO_IPV6, IP6T_SO_ORIGINAL_DST,
			unsafe.Pointer(&addr), &len)
		if err != nil {
			return nil, os.NewSyscallError("getsockopt", err)
		}
		ip := make([]byte, 16)
		for i, b := range addr.Addr {
			ip[i] = b
		}
		pb := *(*[2]byte)(unsafe.Pointer(&addr.Port))
		return &net.TCPAddr{
			IP:   ip,
			Port: int(pb[0])*256 + int(pb[1]),
		}, nil
	}

	// IPv4
	var addr syscall.RawSockaddrInet4
	var len uint32
	len = uint32(unsafe.Sizeof(addr))
	err = getsockopt(fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST,
		unsafe.Pointer(&addr), &len)
	if err != nil {
		return nil, os.NewSyscallError("getsockopt", err)
	}
	ip := make([]byte, 4)
	for i, b := range addr.Addr {
		ip[i] = b
	}
	pb := *(*[2]byte)(unsafe.Pointer(&addr.Port))
	return &net.TCPAddr{
		IP:   ip,
		Port: int(pb[0])*256 + int(pb[1]),
	}, nil
}

var (
	proxyDialer proxy.Dialer
	running     bool
)

func customCopy(dst io.Writer, src io.Reader, buf []byte) (err error) {
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			_, ew := dst.Write(buf[0:nr])
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return err
}

func handleConnection(conn *net.TCPConn) {
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(20 * time.Second)
	conn.SetNoDelay(true)

	dstAddr, err := GetOriginalDST(conn)
	if err != nil {
		log.Printf("[ERROR] GetOriginalDST error: %s\n", err.Error())
		return
	}

	log.Printf("[INFO] Connect to %s.\n", dstAddr.String())

	proxy, err := proxyDialer.Dial("tcp", dstAddr.String())
	if err != nil {
		log.Printf("[ERROR] Dial proxy error: %s.\n", err.Error())
		return
	}

	proxy.(*net.TCPConn).SetKeepAlive(true)
	proxy.(*net.TCPConn).SetKeepAlivePeriod(20 * time.Second)
	proxy.(*net.TCPConn).SetNoDelay(true)

	copyEnd := false

	go func() {
		buf := make([]byte, 32*1024)

		var err error
		if *ioFlag {
			err = customCopy(proxy, conn, buf)
		} else {
			_, err = io.CopyBuffer(proxy, conn, buf)
		}

		if err != nil && !copyEnd {
			log.Printf("[ERROR] Copy error: %s.\n", err.Error())
		}
		copyEnd = true
		conn.Close()
		proxy.Close()
	}()

	buf := make([]byte, 32*1024)
	if *ioFlag {
		err = customCopy(conn, proxy, buf)
	} else {
		_, err = io.CopyBuffer(conn, proxy, buf)
	}

	if err != nil && !copyEnd {
		log.Printf("[ERROR] Copy error: %s.\n", err.Error())
	}
	copyEnd = true
	conn.Close()
	proxy.Close()
}

func main() {
	flag.Parse()
	listener, err := net.Listen("tcp", *ListenAddr)
	if err != nil {
		log.Fatalf("[ERROR] Listen tcp error: %s.\n", err.Error())
	}

	defer listener.Close()

	proxyURL, err := url.Parse(*socks5)
	if err != nil {
		log.Fatalf("[ERROR] Parse proxy address error: %s.\n", err.Error())
	}

	proxyDialer, err = proxy.FromURL(proxyURL, &net.Dialer{})
	if err != nil {
		log.Fatalf("[ERROR] Create proxy error: %s.\n", err.Error())
	}

	log.Println("[INFO] Starting Kumasocks...")

	if *ioFlag {
		log.Println("[INFO] Using io.Copy hack.")
	}

	running = true

	go func() {
		for running {
			conn, err := listener.Accept()
			if err != nil {
				if running {
					log.Printf("[ERROR] TCP accept error: %s.\n", err.Error())
				}
				continue
			}
			go handleConnection(conn.(*net.TCPConn))
		}
	}()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-c

	log.Println("[INFO] Exiting Kumasocks...")

	running = false
	listener.Close()
}
