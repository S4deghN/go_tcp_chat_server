package main

import "os"
import "fmt"
import "net"
import "sync"

var sessions_mutex sync.RWMutex

type Session struct {
	conn net.Conn
	q chan []byte
}

func do_read(sessions map[*Session]struct{}, session *Session) {
	var buff [128]byte
	for {
		n , err := session.conn.Read(buff[:])
		if err != nil || n == 0 {
			sessions_mutex.Lock()
			{
				close(session.q)
				session.conn.Close()
				delete(sessions, session)
			}
			sessions_mutex.Unlock()
			fmt.Print("Closing do_read now!\n")
			break;
		}

		sessions_mutex.RLock()
		{
			for k, _ := range sessions {
				msg := make([]byte, n)
				copy(msg, buff[:n])
				k.q <- msg
			}
		}
		sessions_mutex.RUnlock()
	}
}

func do_write(session *Session) {
	for {
		bytes, more := <- session.q
		session.conn.Write(bytes)
		if !more {
			fmt.Print("Closing do_write now!\n")
			break;
		}
	}
}

func main() {

	sessions := make(map[*Session]struct{})

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		session := &Session{
			conn: conn,
			q: make(chan []byte),
		}

		sessions_mutex.Lock()
		{
			sessions[session] = struct{}{}
		}
		sessions_mutex.Unlock()

		go do_read(sessions, session)
		go do_write(session)

		fmt.Println("Sessions len: ", len(sessions))
	}
}
