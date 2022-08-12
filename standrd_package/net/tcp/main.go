package main

import (
	"fmt"
	"io"
	"net"
	"syscall"
	"time"
)

var closeChan = make(chan struct{})

func startServer(ch chan<- struct{}) {
	serverAddr := &net.TCPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 1024,
	}
	listener, err := net.ListenTCP("tcp", serverAddr)
	if err != nil {
		fmt.Printf("create listen socket error: %v\n", err)
		return
	}
	defer listener.Close()
	close(ch)
	fmt.Printf("Boot server successfully!\n")

	serveGoRoutineIndex := 0
	for {
		newConn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Printf("When accept new conn, error happened: %v\n", err)
			close(closeChan)
			return
		}

		go handleRequest(serveGoRoutineIndex, newConn)
		serveGoRoutineIndex++
	}
}

// handleRequest echo 请求中的信息
func handleRequest(index int, conn *net.TCPConn) {
	time.Sleep(10000 * time.Second)
	fmt.Printf("	goroutine: %d boot\n", index)
	defer fmt.Printf("	goroutine: %d exit\n", index)
	defer conn.Close()
	totalBuf := make([]byte, 0, 1024)
	for {
		tmpBuf := make([]byte, 1024)
		count, err := conn.Read(tmpBuf)
		if err != nil && err != io.EOF {
			fmt.Printf("recv info error: %v\n", err)
			return
		}
		if err == io.EOF {
			break
		}
		totalBuf = append(totalBuf, tmpBuf[:count]...)
	}
	fmt.Printf("	goroutine: %d recv info: %s\n", index, string(totalBuf))
	count, err := conn.Write(totalBuf)
	if err != nil {
		fmt.Printf("echo info error: %v\n", err)
		return
	}

	if count != len(totalBuf) {
		fmt.Printf("echo short info %d, should be: %d", count, len(totalBuf))
		return
	}
	fmt.Printf("	goroutine: %d send info: %s\n", index, string(totalBuf))
}

func clientFunc(info string, ch <-chan struct{}) {
	<-ch
	serverAddr := &net.TCPAddr{
		IP:   net.IP{127, 0, 0, 1},
		Port: 1024,
	}
	conn, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		fmt.Printf("dial tcp error: %v\n", err)
		return
	}
	fmt.Printf("client connect server success\n")

	defer conn.Close()                     // 如果 connection 错误,是否也需要关闭 conn
	count, err := conn.Write([]byte(info)) // string 转化为 info 可以使用更加高效的方式
	if err != nil || count != len(info) {
		fmt.Printf("client send info error: %v\n", err)
		return
	}
	conn.CloseWrite()
	fmt.Printf("client send info: %s\n", info)
	totalBuf := make([]byte, 0, 1024)
	tmpBuf := make([]byte, 1024)
	for {
		readn, err := conn.Read(tmpBuf)
		if err != nil && err != io.EOF {
			fmt.Printf("client recv info error: %v\n", err)
			return
		}
		if err == io.EOF {
			break
		}
		totalBuf = append(totalBuf, tmpBuf[:readn]...)
	}

	fmt.Printf("clent recv info: %s", string(totalBuf))
}

func setMaxOpenFile() {
	limit := &syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, limit)
	fmt.Printf("%d %d", limit.Cur, limit.Max)
	limit.Cur = limit.Max
	syscall.Setrlimit(syscall.RLIMIT_NOFILE, limit)
}

func main() {
	setMaxOpenFile()
	ch := make(chan struct{})
	go startServer(ch)
	for i := 0; i < 1000000; i++ {
		go clientFunc("ping", ch)
	}
	<-closeChan
}


/* 	
单机实现多个连接的瓶颈? 参考:https://www.jianshu.com/p/a55254c84f78?utm_campaign=studygolang.com&utm_medium=studygolang.com&utm_source=studygolang.com

1. 进程可打开文件上限,可以通过系统调用:syscall.G/Setrlimit(syscall.RLIMIT_NOFILE) 查看设置
2. 可用端口号范围:可以在文件系统:/proc/sys/net/ipv4/ip_local_port_range 查看并设置
3. 服务端监听套接子有两个队列:半连接队列 和 全连接队列, 全连接队列的长度为min(进程设置 backlog, /proc/sys/net/core/somaxconn),
当我们调用net.ListenTCP创建套接字的时候,内部就是通过读取/proc/sys/net/core/somaxconn 来设置 backlog 的.
我们可以通过 ss -lsn 来查看监听套接字全连接队列的当前长度和最大长度.
State                        Recv-Q                        Send-Q                                               Local Address:Port                                               Peer Address:Port                       Process                       
LISTEN                       0                             4096                                                 127.0.0.53%lo:53                                                      0.0.0.0:*                                                        
LISTEN                       0                             128                                                        0.0.0.0:22                                                      0.0.0.0:*                                                        
LISTEN                       0                             5                                                        127.0.0.1:631                                                     0.0.0.0:*                                                        
LISTEN                       0                             4096                                                     127.0.0.1:1024                                                    0.0.0.0:*
当 Recv > Send 的时候,说明全连接队列已满.另一种查看队列是否满的方法是:netstat -s | egrep "listen|LISTEN"
667399 times the listen queue of a socket overflowed
667399 SYNs to LISTEN sockets ignored
当我们重复执行该命令的时候,溢出次数是逐渐增加的.

当全连接满的时候,三次握手的第三个 ack, 服务端如何处理呢?两种处理方式,默认是丢弃 ack, 服务端 syn/ack 超时重传,
当重传次数达到限制(/proc/sys/net/ipv4/tcp_synack_retries),发送 RST 分节给客户端.另一种处理方式是直接发送 RST 分节给对端. 我们可以通过设置
/proc/sys/net/ipv4/tcp_abort_on_overflow(默认为0,可以设置为1)实现不同的处理方式.

半连接队列的长度由/proc/sys/net/ipv4/tcp_max_syn_backlog设置,当半连接满的时候,客户端发送的 syn 会直接被忽略,客户端会超时重传 syn, 超时后返回 TIME_OUT 错误.
*/