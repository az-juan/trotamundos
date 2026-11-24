// Package cache provee un almacén clave-valor efímero con TTL (ADR-109/110).
// Si REDIS_ADDR está disponible habla protocolo RESP contra Redis 7; de lo contrario
// degrada a un mapa en memoria con expiración para tests unitarios y ejecución sin Redis.
package cache

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

type Store struct {
	addr string
	mu   sync.Mutex
	mem  map[string]entry
}

func New(addr string) *Store {
	return &Store{
		addr: strings.TrimSpace(addr),
		mem:  map[string]entry{},
	}
}

func (s *Store) Set(key, value string, ttl time.Duration) {
	if s.addr != "" {
		sec := int(ttl.Seconds())
		if sec <= 0 {
			sec = 60
		}
		if _, err := s.cmd("SET", key, value, "EX", strconv.Itoa(sec)); err == nil {
			return
		}
	}
	s.mu.Lock()
	s.mem[key] = entry{value: value, expiresAt: time.Now().Add(ttl)}
	s.mu.Unlock()
}

func (s *Store) Get(key string) (string, bool) {
	if s.addr != "" {
		if val, err := s.cmd("GET", key); err == nil {
			if val == "" {
				return "", false
			}
			return val, true
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.mem[key]
	if !ok || time.Now().After(e.expiresAt) {
		delete(s.mem, key)
		return "", false
	}
	return e.value, true
}

// GetDel consume un token de un solo uso (ADR-110: context_token en Redis).
func (s *Store) GetDel(key string) (string, bool) {
	if s.addr != "" {
		if val, err := s.cmd("GETDEL", key); err == nil {
			if val == "" {
				return "", false
			}
			return val, true
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.mem[key]
	delete(s.mem, key)
	if !ok || time.Now().After(e.expiresAt) {
		return "", false
	}
	return e.value, true
}

func (s *Store) cmd(args ...string) (string, error) {
	conn, err := net.DialTimeout("tcp", s.addr, 500*time.Millisecond)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(1 * time.Second))

	var b strings.Builder
	fmt.Fprintf(&b, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&b, "$%d\r\n%s\r\n", len(a), a)
	}
	if _, err := io.WriteString(conn, b.String()); err != nil {
		return "", err
	}
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if len(line) == 0 {
		return "", fmt.Errorf("respuesta RESP vacía")
	}
	switch line[0] {
	case '+':
		return line[1:], nil
	case '-':
		return "", fmt.Errorf("redis error: %s", line[1:])
	case '$':
		n, err := strconv.Atoi(line[1:])
		if err != nil || n < 0 {
			return "", nil
		}
		buf := make([]byte, n+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return "", err
		}
		return string(buf[:n]), nil
	default:
		return "", nil
	}
}
