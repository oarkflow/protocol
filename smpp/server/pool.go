// Copyright 2015 sms-api-server authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package server

import (
	"encoding/hex"
	"github.com/oarkflow/protocol/smpp/pdu"
	"github.com/oarkflow/protocol/smpp/pdu/pdufield"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
)

// DeliveryReceipt contains the arguments of RPC call to SM.Deliver.
// We only call it, not handle.
type DeliveryReceipt struct {
	Src      string `json:"src"`
	Dst      string `json:"dst"`
	Text     string `json:"text"`
	EMSClass uint8  `json:"emsclass"`
	// random local ID to de-dup websocket messages
	LocalID string `json:"localid"`
}

var deliveryID uint64

// deliveryPool let peers register themselves to receive broadcast
// notifications with delivery receipts.
type deliveryPool struct {
	mu sync.Mutex
	m  map[uint64]chan *DeliveryReceipt
}

func newPool() *deliveryPool {
	return &deliveryPool{m: make(map[uint64]chan *DeliveryReceipt)}
}

// Handler handles DeliverSM coming from a Transceiver SMPP connection.
// It broadcasts received messages and delivery receipt to all registered
// peers.
func (pool *deliveryPool) Handler(p pdu.Body) {
	log.Printf("incoming message: %s", p.Header().ID)
	switch p.Header().ID {
	case pdu.DeliverSMID:
		f := p.Fields()
		dr := &DeliveryReceipt{
			Src:     f[pdufield.SourceAddr].String(),
			Dst:     f[pdufield.DestinationAddr].String(),
			Text:    f[pdufield.ShortMessage].String(),
			LocalID: randomID(),
		}
		if cl := f[pdufield.ESMClass].Bytes(); len(cl) > 0 {
			dr.EMSClass = cl[0]
		}
		pool.Broadcast(dr)
	default:
	}
}

// Register returns a channel that get broadcasts from the pool.
// The returned ID (uint64) is used to Unregister.
func (pool *deliveryPool) Register() (uint64, <-chan *DeliveryReceipt) {
	id := atomic.AddUint64(&deliveryID, 1)
	c := make(chan *DeliveryReceipt, 10)
	pool.mu.Lock()
	pool.m[id] = c
	pool.mu.Unlock()
	return id, c
}

// Unregister removes an entry from the delivery receipt broadcast,
// and closes the channel previously returned by Register.
func (pool *deliveryPool) Unregister(id uint64) {
	pool.mu.Lock()
	c := pool.m[id]
	if c != nil {
		delete(pool.m, id)
		close(c)
	}
	pool.mu.Unlock()
}

// Broadcast broadcasts the given delivery receipt to all registered peers.
func (pool *deliveryPool) Broadcast(r *DeliveryReceipt) {
	pool.mu.Lock()
	log.Printf("broadcasting dlr to %d peers: %s->%s, text:%s, id:%s", len(pool.m), r.Src, r.Dst, r.Text, r.LocalID)
	for _, c := range pool.m {
		select {
		case c <- r:
		default:
			// TODO: Increment drop counter here.
		}
	}
	pool.mu.Unlock()
}

func randomID() string {
	var m []byte
	for i := 0; i < 12; i++ {
		m = append(m, byte(rand.Int()))
	}
	return hex.EncodeToString(m)
}
