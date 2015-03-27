package main

type hub struct {
	// Registered connections.
	connections map[*connection]bool

	// Registered telnet connections.
	telnetconns map[*telnetconn]bool

	// Inbound messages from the connections.
	broadcast chan []byte

	// Inbound messages from the telnet connections.
	telbroadcast chan []byte

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection

	// Register telnet requests from the connections.
	telregis chan *telnetconn

	// Unregister telnet requests from connections.
	telunregis chan *telnetconn
}

var h = hub{
	broadcast:   make(chan []byte),
	telbroadcast:   make(chan []byte),
	register:    make(chan *connection),
	unregister:  make(chan *connection),
	connections: make(map[*connection]bool),
	telnetconns: make(map[*telnetconn]bool),
	telregis:    make(chan *telnetconn),
	telunregis:  make(chan *telnetconn),
}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
			}
		case m := <-h.broadcast:
			for c := range h.connections {
				select {
				case c.send <- m:
				default:
					delete(h.connections, c)
					close(c.send)
				}
			}
		case c := <-h.telregis:
			h.telnetconns[c] = true
		case c := <-h.telunregis:
			if _, ok := h.telnetconns[c]; ok {
				delete(h.telnetconns, c)
				close(c.send)
			}
		case m := <-h.telbroadcast:
			for c := range h.telnetconns {
				select {
				case c.send <- m:
				default:
					delete(h.telnetconns, c)
					close(c.send)
				}
			}
		}




	}
}
