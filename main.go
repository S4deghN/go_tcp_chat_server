package main

import "os"
import "net"
import "time"
import "errors"
import "log"

type MessageType int
const (
	ClientConnect MessageType = iota
	ClientDisconnect
	TextMessage
)

type Message struct {
	t MessageType
	session *Session
	data []byte
}

type Session struct {
	conn net.Conn
	write_q chan []byte
}

func do_read(session *Session, server_msg_q chan Message) {
	for {
		buff := make([]byte, 512)
		n, err := session.conn.Read(buff)
		if err != nil || n == 0 {
			server_msg_q <- Message{t: ClientDisconnect, session: session}
			log.Print("Closing do_read now!")
			break
		}

		server_msg_q <- Message {
			t: TextMessage,
			session: session,
			data: buff[:n],
		}
	}
}

func do_write(session *Session) {
	for {
		bytes, more := <-session.write_q
		if !more {
			break
		}
		session.conn.SetWriteDeadline(time.Now().Add(3*time.Second))
		_, err := session.conn.Write(bytes)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				log.Print("Client write timed out")
			}
			break;
		}
	}
	log.Print("Closing do_write now!")
}

func server(sessions map[*Session]struct{}, server_msg_q chan Message) {
	for {
		msg := <- server_msg_q
		switch msg.t {
		case ClientConnect: {
			sessions[msg.session] = struct{}{}
			go do_read(msg.session, server_msg_q)
			go do_write(msg.session)
			log.Print("Client Connected!")
		}
		case ClientDisconnect: {
			close(msg.session.write_q)
			msg.session.conn.Close()
			delete(sessions, msg.session)
			log.Print("Client Disconnected!")
		}
		case TextMessage: {
			for k, _ := range sessions {
				k.write_q <- msg.data
			}
		}
		default:
		}
	}
}

func main() {
	address := ":8080"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listening on %s", address)

	sessions := make(map[*Session]struct{})
	server_msg_q := make(chan Message)
	go server(sessions, server_msg_q)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}

		session := &Session{
			conn: conn,
			write_q: make(chan []byte),
		}
		server_msg_q <- Message{t: ClientConnect, session: session}
	}
}
