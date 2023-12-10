// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdu

import (
	"github.com/oarkflow/protocol/smpp/manager"
	"github.com/oarkflow/protocol/smpp/pdu/pdufield"
	"github.com/oarkflow/protocol/smpp/pdu/pdutlv"
)

// PDU Types.
const (
	GenericNACKID         ID = 0x80000000
	BindReceiverID        ID = 0x00000001
	BindReceiverRespID    ID = 0x80000001
	BindTransmitterID     ID = 0x00000002
	BindTransmitterRespID ID = 0x80000002
	QuerySMID             ID = 0x00000003
	QuerySMRespID         ID = 0x80000003
	SubmitSMID            ID = 0x00000004
	SubmitSMRespID        ID = 0x80000004
	DeliverSMID           ID = 0x00000005
	DeliverSMRespID       ID = 0x80000005
	UnbindID              ID = 0x00000006
	UnbindRespID          ID = 0x80000006
	ReplaceSMID           ID = 0x00000007
	ReplaceSMRespID       ID = 0x80000007
	CancelSMID            ID = 0x00000008
	CancelSMRespID        ID = 0x80000008
	BindTransceiverID     ID = 0x00000009
	BindTransceiverRespID ID = 0x80000009
	OutbindID             ID = 0x0000000B
	EnquireLinkID         ID = 0x00000015
	EnquireLinkRespID     ID = 0x80000015
	SubmitMultiID         ID = 0x00000021
	SubmitMultiRespID     ID = 0x80000021
	AlertNotificationID   ID = 0x00000102
	DataSMID              ID = 0x00000103
	DataSMRespID          ID = 0x80000103
)

// GenericNACK PDU.
type GenericNACK struct{ *codec }

func newGenericNACK(hdr *Header) *codec {
	return &codec{h: hdr}
}

// NewGenericNACK creates and initializes a GenericNACK PDU.
func NewGenericNACK(manager manager.IManager) Body {
	b := newGenericNACK(&Header{ID: GenericNACKID})
	b.manager = manager
	b.init()
	return b
}

// Bind PDU.
type Bind struct{ *codec }

func newBind(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.SystemID,
			pdufield.Password,
			pdufield.SystemType,
			pdufield.InterfaceVersion,
			pdufield.AddrTON,
			pdufield.AddrNPI,
			pdufield.AddressRange,
		}}
}

// NewBindReceiver creates a new Bind PDU.
func NewBindReceiver(manager manager.IManager) Body {
	b := newBind(&Header{ID: BindReceiverID})
	b.manager = manager
	b.init()
	return b
}

// NewBindTransceiver creates a new Bind PDU.
func NewBindTransceiver(manager manager.IManager) Body {
	b := newBind(&Header{ID: BindTransceiverID})
	b.manager = manager
	b.init()
	return b
}

// NewBindTransmitter creates a new Bind PDU.
func NewBindTransmitter(manager manager.IManager) Body {
	b := newBind(&Header{ID: BindTransmitterID})
	b.manager = manager
	b.init()
	return b
}

// BindResp PDU.
type BindResp struct{ *codec }

func newBindResp(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{pdufield.SystemID},
	}
}

// NewBindReceiverResp creates and initializes a new BindResp PDU.
func NewBindReceiverResp(manager manager.IManager) Body {
	b := newBindResp(&Header{ID: BindReceiverRespID})
	b.manager = manager
	b.init()
	return b
}

// NewBindTransceiverResp creates and initializes a new BindResp PDU.
func NewBindTransceiverResp(manager manager.IManager) Body {
	b := newBindResp(&Header{ID: BindTransceiverRespID})
	b.manager = manager
	b.init()
	return b
}

// NewBindTransmitterResp creates and initializes a new BindResp PDU.
func NewBindTransmitterResp(manager manager.IManager) Body {
	b := newBindResp(&Header{ID: BindTransmitterRespID})
	b.manager = manager
	b.init()
	return b
}

// Outbind PDU.
type Outbind struct{ *codec }

func newOutbind(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.SystemID,
			pdufield.Password,
		},
	}
}

// CancelSM PDU.
type CancelSM struct{ *codec }

func newCancelSM(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.MessageID,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.DestAddrTON,
			pdufield.DestAddrNPI,
			pdufield.DestinationAddr,
		},
	}
}

// NewCancelSM creates and initializes a new CancelSM PDU.
func NewCancelSM(manager manager.IManager) Body {
	b := newCancelSM(&Header{ID: CancelSMID})
	b.manager = manager
	b.init()
	return b
}

// CancelSMResp PDU.
type CancelSMResp struct{ *codec }

func newCancelSMResp(hdr *Header) *codec {
	return &codec{h: hdr}
}

func NewCancelSMResp(manager manager.IManager) Body {
	b := newCancelSMResp(&Header{ID: CancelSMRespID})
	b.manager = manager
	b.init()
	return b
}

// ReplaceSM PDU.
type ReplaceSM struct{ *codec }

func newReplaceSM(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.ShortMessage,
		},
	}
}

func NewReplaceSM(manager manager.IManager) Body {
	b := newReplaceSM(&Header{ID: ReplaceSMID})
	b.manager = manager
	b.init()
	return b
}

// ReplaceSMResp PDU.
type ReplaceSMResp struct{ *codec }

func newReplaceSMResp(hdr *Header) *codec {
	return &codec{h: hdr}
}

// AlertNotification PDU.
type AlertNotification struct{ *codec }

func newAlertNotification(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.ESMEAddrTON,
			pdufield.ESMEAddrNPI,
			pdufield.ESMEAddr,
		},
	}
}

// QuerySM PDU.
type QuerySM struct{ *codec }

func newQuerySM(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
		},
	}
}

// NewQuerySM creates and initializes a new QuerySM PDU.
func NewQuerySM(manager manager.IManager) Body {
	b := newQuerySM(&Header{ID: QuerySMID})
	b.manager = manager
	b.init()
	return b
}

// QuerySMResp PDU.
type QuerySMResp struct{ *codec }

func newQuerySMResp(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.FinalDate,
			pdufield.MessageState,
			pdufield.ErrorCode,
		},
	}
}

// NewQuerySMResp creates and initializes a new QuerySMResp PDU.
func NewQuerySMResp(manager manager.IManager) Body {
	b := newQuerySMResp(&Header{ID: QuerySMRespID})
	b.manager = manager
	b.init()
	return b
}

// SubmitSM PDU.
type SubmitSM struct{ *codec }

func newSubmitSM(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.DestAddrTON,
			pdufield.DestAddrNPI,
			pdufield.DestinationAddr,
			pdufield.ESMClass,
			pdufield.ProtocolID,
			pdufield.PriorityFlag,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.ReplaceIfPresentFlag,
			pdufield.DataCoding,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.UDHLength,
			pdufield.GSMUserData,
			pdufield.ShortMessage,
		},
	}
}

// NewSubmitSM creates and initializes a new SubmitSM PDU.
func NewSubmitSM(fields pdutlv.Fields, manager manager.IManager) Body {
	b := newSubmitSM(&Header{ID: SubmitSMID})
	b.manager = manager
	b.init()
	for tag, value := range fields {
		b.t.Set(tag, value)
	}
	return b
}

// SubmitSMResp PDU.
type SubmitSMResp struct{ *codec }

func newSubmitSMResp(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
		},
	}
}

// NewSubmitSMResp creates and initializes a new SubmitSMResp PDU.
func NewSubmitSMResp(manager manager.IManager) Body {
	b := newSubmitSMResp(&Header{ID: SubmitSMRespID})
	b.manager = manager
	b.init()
	return b
}

// DataSM PDU.
type DataSM struct{ *codec }

func newDataSM(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.DestAddrTON,
			pdufield.DestAddrNPI,
			pdufield.DestinationAddr,
			pdufield.ESMClass,
			pdufield.RegisteredDelivery,
			pdufield.DataCoding,
		},
	}
}

// NewDataSM creates and initializes a new DataSM PDU.
func NewDataSM(fields pdutlv.Fields, manager manager.IManager) Body {
	b := newDataSM(&Header{ID: DataSMID})
	b.manager = manager
	b.init()
	for tag, value := range fields {
		_ = b.t.Set(tag, value)
	}
	return b
}

// DataSMResp PDU.
type DataSMResp struct{ *codec }

// newDataSMResp creates and initializes a new newDataSMResp PDU.
func newDataSMResp(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
		},
	}
}

// NewDataSMResp creates and initializes a new NewDataSMResp PDU.
func NewDataSMResp(manager manager.IManager) Body {
	b := newDataSMResp(&Header{ID: DataSMRespID})
	b.manager = manager
	b.init()
	return b
}

// SubmitMulti PDU.
type SubmitMulti struct{ *codec }

func newSubmitMulti(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.NumberDests,
			pdufield.DestinationList, // contains DestFlag, DestAddrTON and DestAddrNPI for each address
			pdufield.ESMClass,
			pdufield.ProtocolID,
			pdufield.PriorityFlag,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.ReplaceIfPresentFlag,
			pdufield.DataCoding,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.ShortMessage,
		},
	}
}

// NewSubmitMulti creates and initializes a new SubmitMulti PDU.
func NewSubmitMulti(fields pdutlv.Fields, manager manager.IManager) Body {
	b := newSubmitMulti(&Header{ID: SubmitMultiID})
	b.manager = manager
	b.init()
	for tag, value := range fields {
		b.t.Set(tag, value)
	}
	return b
}

// SubmitMultiResp PDU.
type SubmitMultiResp struct{ *codec }

func newSubmitMultiResp(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
			pdufield.NoUnsuccess,
			pdufield.UnsuccessSme,
		},
	}
}

// NewSubmitMultiResp creates and initializes a new SubmitMultiResp PDU.
func NewSubmitMultiResp(manager manager.IManager) Body {
	b := newSubmitMultiResp(&Header{ID: SubmitMultiRespID})
	b.manager = manager
	b.init()
	return b
}

// DeliverSM PDU.
type DeliverSM struct{ *codec }

func newDeliverSM(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.ServiceType,
			pdufield.SourceAddrTON,
			pdufield.SourceAddrNPI,
			pdufield.SourceAddr,
			pdufield.DestAddrTON,
			pdufield.DestAddrNPI,
			pdufield.DestinationAddr,
			pdufield.ESMClass,
			pdufield.ProtocolID,
			pdufield.PriorityFlag,
			pdufield.ScheduleDeliveryTime,
			pdufield.ValidityPeriod,
			pdufield.RegisteredDelivery,
			pdufield.ReplaceIfPresentFlag,
			pdufield.DataCoding,
			pdufield.SMDefaultMsgID,
			pdufield.SMLength,
			pdufield.ShortMessage,
		},
	}
}

// NewDeliverSM creates and initializes a new DeliverSM PDU.
func NewDeliverSM(manager manager.IManager) Body {
	b := newDeliverSM(&Header{ID: DeliverSMID})
	b.manager = manager
	b.init()
	return b
}

// DeliverSMResp PDU.
type DeliverSMResp struct{ *codec }

func newDeliverSMResp(hdr *Header) *codec {
	return &codec{
		h: hdr,
		l: pdufield.List{
			pdufield.MessageID,
		},
	}
}

// NewDeliverSMResp creates and initializes a new DeliverSMResp PDU.
func NewDeliverSMResp(manager manager.IManager) Body {
	b := newDeliverSMResp(&Header{ID: DeliverSMRespID})
	b.manager = manager
	b.init()
	return b
}

// NewDeliverSMRespSeq creates and initializes a new DeliverSMResp PDU for a specific seq.
func NewDeliverSMRespSeq(seq uint32, manager manager.IManager) Body {
	b := newDeliverSMResp(&Header{ID: DeliverSMRespID, Seq: seq})
	b.manager = manager
	b.init()
	return b
}

// Unbind PDU.
type Unbind struct{ *codec }

func newUnbind(hdr *Header) *codec {
	return &codec{h: hdr}
}

// NewUnbind creates and initializes a Unbind PDU.
func NewUnbind(manager manager.IManager) Body {
	b := newUnbind(&Header{ID: UnbindID})
	b.manager = manager
	b.init()
	return b
}

// UnbindResp PDU.
type UnbindResp struct{ *codec }

func newUnbindResp(hdr *Header) *codec {
	return &codec{h: hdr}
}

// NewUnbindResp creates and initializes a UnbindResp PDU.
func NewUnbindResp(manager manager.IManager) Body {
	b := newUnbindResp(&Header{ID: UnbindRespID})
	b.manager = manager
	b.init()
	return b
}

// EnquireLink PDU.
type EnquireLink struct{ *codec }

func newEnquireLink(hdr *Header) *codec {
	return &codec{h: hdr}
}

// NewEnquireLink creates and initializes a EnquireLink PDU.
func NewEnquireLink(manager manager.IManager) Body {
	b := newEnquireLink(&Header{ID: EnquireLinkID})
	b.manager = manager
	b.init()
	return b
}

// EnquireLinkResp PDU.
type EnquireLinkResp struct{ *codec }

func newEnquireLinkResp(hdr *Header) *codec {
	return &codec{h: hdr}
}

// NewEnquireLinkResp creates and initializes a EnquireLinkResp PDU.
func NewEnquireLinkResp(manager manager.IManager) Body {
	b := newEnquireLinkResp(&Header{ID: EnquireLinkRespID})
	b.manager = manager
	b.init()
	return b
}

// NewEnquireLinkRespSeq creates and initializes a EnquireLinkResp PDU for a specific seq.
func NewEnquireLinkRespSeq(seq uint32, manager manager.IManager) Body {
	b := newEnquireLinkResp(&Header{ID: EnquireLinkRespID, Seq: seq})
	b.manager = manager
	b.init()
	return b
}
