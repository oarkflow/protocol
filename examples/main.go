package main

import (
	"fmt"

	"github.com/oarkflow/protocol/smpp"
	"github.com/oarkflow/protocol/smpp/pdu/pdufield"
)

func main() {
	setting := smpp.Setting{
		Name: "Dove Cote",
		Slug: "dove-cote",
		// URL:  "138.201.53.230:5001",
		URL: "localhost:2775",
		Auth: smpp.Auth{
			SystemID: "verisoR",
			Password: "Ver!12@$",
		},
		Register: pdufield.FinalDeliveryReceipt,
		OnMessageReport: func(manager *smpp.Manager, sms *smpp.Message, parts []*smpp.Part) {
			fmt.Println("Message Report", sms)
			for _, part := range parts {
				fmt.Println(part)
			}
		},
	}
	manager, err := smpp.NewManager(setting)
	if err != nil {
		panic(err)
	}

	for i := 0; i < 2; i++ {
		msg := smpp.Message{
			Message: fmt.Sprintf("This is test. %d", i),
			To:      "9779856034616",
			From:    "verishore",
		}
		manager.Send(msg)
	}
	manager.Wait()
}
