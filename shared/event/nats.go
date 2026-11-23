package event

// Adaptador NATS JetStream de la arista Filius↔Spiritus (ADR-103/108).
// Se activa cuando NATS_URL está definida; si no, el prototipo usa HTTPPublisher.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

var streamSubjects = map[string][]string{
	"TRIPS":         {"trip.>"},
	"BOOKINGS":      {"booking.>"},
	"NOTIFICATIONS": {"notification.>"},
	"ASSISTANCE":    {"assistance.>"},
}

// EnsureStreams crea los streams del catálogo si no existen (idempotente).
func EnsureStreams(js nats.JetStreamContext) error {
	for name, subjects := range streamSubjects {
		if _, err := js.StreamInfo(name); err == nil {
			continue
		}
		_, err := js.AddStream(&nats.StreamConfig{
			Name: name, Subjects: subjects, Storage: nats.FileStorage, MaxAge: 30 * 24 * time.Hour,
		})
		if err != nil {
			return fmt.Errorf("crear stream %s: %w", name, err)
		}
	}
	return nil
}

// NATSPublisher publica hechos ricos en JetStream.
type NATSPublisher struct {
	URL  string
	Conn *nats.Conn
}

func NewNATSPublisher(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url, nats.Name("trotamundos-filius"))
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, err
	}
	if err := EnsureStreams(js); err != nil {
		nc.Close()
		return nil, err
	}
	return &NATSPublisher{URL: url, Conn: nc}, nil
}

func (p *NATSPublisher) Close() {
	if p.Conn != nil {
		p.Conn.Close()
	}
}

func (p *NATSPublisher) Publish(_ context.Context, e Event) error {
	var nc *nats.Conn
	if p.Conn != nil {
		nc = p.Conn
	} else {
		var err error
		nc, err = nats.Connect(p.URL)
		if err != nil {
			return err
		}
		defer nc.Close()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	js, err := nc.JetStream()
	if err != nil {
		return err
	}
	_, err = js.Publish(e.Type, data)
	if err != nil {
		return fmt.Errorf("publicar %s en JetStream: %w", e.Type, err)
	}
	return nil
}

// Subscribe procesa eventos de un subject con consumidor durable y ack manual.
// Se usa en Spiritus para alimentar el read-model (ADR-107).
func Subscribe(ctx context.Context, url, subject, durable string, handle func(Event) error) error {
	nc, err := nats.Connect(url, nats.Name("trotamundos-"+durable))
	if err != nil {
		return err
	}
	defer nc.Close()
	js, err := nc.JetStream()
	if err != nil {
		return err
	}
	if err := EnsureStreams(js); err != nil {
		return err
	}
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return '-'
	}, durable)
	sub, err := js.SubscribeSync(subject, nats.Durable(safe), nats.ManualAck())
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		msg, err := sub.NextMsg(2 * time.Second)
		if err != nil {
			if err == nats.ErrTimeout {
				continue
			}
			return err
		}
		var e Event
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			_ = msg.Nak()
			continue
		}
		if err := handle(e); err != nil {
			_ = msg.Nak()
			continue
		}
		_ = msg.Ack()
	}
}
