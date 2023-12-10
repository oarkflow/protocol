package main

import (
	"fmt"
	"time"

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
	}
	manager, err := smpp.NewManager(setting)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 2; i++ {
		msg := smpp.Message{
			Message: fmt.Sprintf("नेकपा माओवादी केन्द्रका अध्यक्ष एवं पूर्वप्रधानमन्त्री पुष्पकमल दाहालले आउँदो मंसिर ४ गतेको प्रदेश र प्रतिनिधिसभा निर्वाचनमा प्रतिगमन परास्त हुनेगरी जनताले एकता प्रदर्शन गर्नुपर्ने बताएका छन् । %d", i),
			To:      "9779856034616",
			From:    "verishore",
		}
		manager.Send(msg)
	}
	time.Sleep(5 * time.Second)
	fmt.Println(manager.GetMessages())
}
