// Package mail guarda buzones, remitentes seguros y dedup del correo (FR-2/FR-3).
package mail

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
)

type Mailbox struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	Email      string `json:"email"`
	LastScanAt string `json:"last_scan_at,omitempty"`
}

type Sender struct {
	ID     string `json:"id"`
	Sender string `json:"sender"`
}

type Memory struct {
	mu        sync.RWMutex
	mailboxes []Mailbox
	senders   []Sender
	seen      map[string]bool
	seq       int
}

func New() *Memory {
	m := &Memory{seen: map[string]bool{}}
	m.AddMailbox("gmail", "dev@trotamundos.app")
	m.AddSender("@aerolineasdelplata.com")
	m.AddSender("@hotelpresidente.com")
	return m
}

func (s *Memory) nextID() string {
	s.seq++
	return hex.EncodeToString([]byte{byte(s.seq)}) + "-" + hex.EncodeToString(sha256.New().Sum(nil)[:4])
}

func (s *Memory) AddMailbox(provider, email string) Mailbox {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.mailboxes {
		if m.Email == email {
			return m
		}
	}
	m := Mailbox{ID: hex.EncodeToString(sha256.New().Sum([]byte(email))[:6]), Provider: provider, Email: email}
	s.mailboxes = append(s.mailboxes, m)
	return m
}

func (s *Memory) Mailboxes() []Mailbox {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Mailbox{}, s.mailboxes...)
}

func (s *Memory) AddSender(sender string) Sender {
	s.mu.Lock()
	defer s.mu.Unlock()
	sender = strings.ToLower(strings.TrimSpace(sender))
	for _, x := range s.senders {
		if x.Sender == sender {
			return x
		}
	}
	x := Sender{ID: hex.EncodeToString(sha256.New().Sum([]byte(sender))[:6]), Sender: sender}
	s.senders = append(s.senders, x)
	return x
}

func (s *Memory) Senders() []Sender {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Sender{}, s.senders...)
}

func (s *Memory) RemoveSender(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.senders {
		if x.ID == id {
			s.senders = append(s.senders[:i], s.senders[i+1:]...)
			return true
		}
	}
	return false
}

// Allowed aplica la whitelist: un dominio (@...) o un email exacto.
func (s *Memory) Allowed(from string) bool {
	if strings.TrimSpace(from) == "" {
		return true // el demo no indica remitente
	}
	from = strings.ToLower(from)
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, x := range s.senders {
		if strings.HasPrefix(x.Sender, "@") && strings.HasSuffix(from, x.Sender) {
			return true
		}
		if x.Sender == from {
			return true
		}
	}
	return false
}

// Seen implementa dedup SHA256 (FR-2): devuelve true si el correo ya fue procesado.
func (s *Memory) Seen(subject, body string) bool {
	sum := sha256.Sum256([]byte(subject + "\x00" + body))
	key := hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[key] {
		return true
	}
	s.seen[key] = true
	return false
}
