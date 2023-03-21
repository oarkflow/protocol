// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdu

import (
	"encoding/binary"
	"fmt"
	"io"
)

type (
	// ID of the PDU header.
	ID uint32

	// Status is a property of the PDU header.
	Status uint32
)

var idString = map[ID]string{
	GenericNACKID:         "GenericNACK",
	BindReceiverID:        "BindReceiver",
	BindReceiverRespID:    "BindReceiverResp",
	BindTransmitterID:     "BindTransmitter",
	BindTransmitterRespID: "BindTransmitterResp",
	QuerySMID:             "QuerySM",
	QuerySMRespID:         "QuerySMResp",
	SubmitSMID:            "SubmitSM",
	SubmitSMRespID:        "SubmitSMResp",
	DeliverSMID:           "DeliverSM",
	DeliverSMRespID:       "DeliverSMResp",
	UnbindID:              "Unbind",
	UnbindRespID:          "UnbindResp",
	ReplaceSMID:           "ReplaceSM",
	ReplaceSMRespID:       "ReplaceSMResp",
	CancelSMID:            "CancelSM",
	CancelSMRespID:        "CancelSMResp",
	BindTransceiverID:     "BindTransceiver",
	BindTransceiverRespID: "BindTransceiverResp",
	OutbindID:             "Outbind",
	EnquireLinkID:         "EnquireLink",
	EnquireLinkRespID:     "EnquireLinkResp",
	SubmitMultiID:         "SubmitMulti",
	SubmitMultiRespID:     "SubmitMultiResp",
	AlertNotificationID:   "AlertNotification",
	DataSMID:              "DataSM",
	DataSMRespID:          "DataSMResp",
}

// String returns the PDU type as a string.
func (id ID) String() string {
	return idString[id]
}

// Group returns group of a given ID.
// Example: SubmitSM, SubmitSMResp should return the same group: 0x04.
func (id ID) Group() uint16 {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(id))
	return binary.BigEndian.Uint16(b[2:4])
}

// HeaderLen is the PDU header length.
const HeaderLen = 16

// Header is a PDU header.
type Header struct {
	Len    uint32
	ID     ID
	Status Status
	Seq    uint32 // Sequence number.
}

// Key return key of a given header based on ID and Seq
func (h *Header) Key() string {
	return fmt.Sprintf("%o-%d", h.ID.Group(), h.Seq)
}

// DecodeHeader decodes binary PDU header data.
func DecodeHeader(r io.Reader) (*Header, error) {
	b := make([]byte, HeaderLen)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	l := binary.BigEndian.Uint32(b[0:4])
	if l < HeaderLen {
		return nil, fmt.Errorf("PDU too small: %d < %d", l, HeaderLen)
	}
	if l > MaxSize {
		return nil, fmt.Errorf("PDU too large: %d > %d", l, MaxSize)
	}
	hdr := &Header{
		Len:    l,
		ID:     ID(binary.BigEndian.Uint32(b[4:8])),
		Status: Status(binary.BigEndian.Uint32(b[8:12])),
		Seq:    binary.BigEndian.Uint32(b[12:16]),
	}
	return hdr, nil
}

// SerializeTo serializes the Header to its binary form to the given writer.
func (h *Header) SerializeTo(w io.Writer) error {
	b := make([]byte, HeaderLen)
	binary.BigEndian.PutUint32(b[0:4], h.Len)
	binary.BigEndian.PutUint32(b[4:8], uint32(h.ID))
	binary.BigEndian.PutUint32(b[8:12], uint32(h.Status))
	binary.BigEndian.PutUint32(b[12:16], h.Seq)
	_, err := w.Write(b)
	return err
}

// Error implements the Error interface.
func (s Status) Error() string {
	m, ok := esmeStatus[s]
	if !ok {
		return fmt.Sprintf("unknown status: %d", s)
	}
	return m
}

const (
	ESME_ROK              Status = 0x00000000 // No Error
	ESME_RINVMSGLEN       Status = 0x00000001 // Message Length is invalid
	ESME_RINVCMDLEN       Status = 0x00000002 // Command Length is invalid
	ESME_RINVCMDID        Status = 0x00000003 // Invalid Command ID
	ESME_RINVBNDSTS       Status = 0x00000004 // Incorrect BIND Status for given command
	ESME_RALYBND          Status = 0x00000005 // ESME Already in Bound State
	ESME_RINVPRTFLG       Status = 0x00000006 // Invalid Priority Flag
	ESME_RINVREGDLVFLG    Status = 0x00000007 // Invalid Registered Delivery Flag
	ESME_RSYSERR          Status = 0x00000008 // System Error
	ESME_RINVSRCADR       Status = 0x0000000A // Invalid Source Address
	ESME_RINVDSTADR       Status = 0x0000000B // Invalid Dest Addr
	ESME_RINVMSGID        Status = 0x0000000C // Message ID is invalid
	ESME_RBINDFAIL        Status = 0x0000000D // Bind Failed
	ESME_RINVPASWD        Status = 0x0000000E // Invalid Password
	ESME_RINVSYSID        Status = 0x0000000F // Invalid System ID
	ESME_RCANCELFAIL      Status = 0x00000011 // Cancel SM Failed
	ESME_RREPLACEFAIL     Status = 0x00000013 // Replace SM Failed
	ESME_RMSGQFUL         Status = 0x00000014 // Message Queue Full
	ESME_RINVSERTYP       Status = 0x00000015 // Invalid Service Type
	ESME_RINVNUMDESTS     Status = 0x00000033 // Invalid number of destinations
	ESME_RINVDLNAME       Status = 0x00000034 // Invalid Distribution List name
	ESME_RINVDESTFLAG     Status = 0x00000040 // Destination flag (submit_multi)
	ESME_RINVSUBREP       Status = 0x00000042 // Invalid ‘submit with replace’ request (i.e. submit_sm with replace_if_present_flag set)
	ESME_RINVESMSUBMIT    Status = 0x00000043 // Invalid esm_SUBMIT field data
	ESME_RCNTSUBDL        Status = 0x00000044 // Cannot Submit to Distribution List
	ESME_RSUBMITFAIL      Status = 0x00000045 // submit_sm or submit_multi failed
	ESME_RINVSRCTON       Status = 0x00000048 // Invalid Source address TON
	ESME_RINVSRCNPI       Status = 0x00000049 // Invalid Source address NPI
	ESME_RINVDSTTON       Status = 0x00000050 // Invalid Destination address TON
	ESME_RINVDSTNPI       Status = 0x00000051 // Invalid Destination address NPI
	ESME_RINVSYSTYP       Status = 0x00000053 // Invalid system_type field
	ESME_RINVREPFLAG      Status = 0x00000054 // Invalid replace_if_present flag
	ESME_RINVNUMMSGS      Status = 0x00000055 // Invalid number of messages
	ESME_RTHROTTLED       Status = 0x00000058 // Throttling error (ESME has exceeded allowed message limits)
	ESME_RINVSCHED        Status = 0x00000061 // Invalid Scheduled Delivery Time
	ESME_RINVEXPIRY       Status = 0x00000062 // Invalid message (Expiry time)
	ESME_RINVDFTMSGID     Status = 0x00000063 // Predefined Message Invalid or Not Found
	ESME_RX_T_APPN        Status = 0x00000064 // ESME Receiver Temporary App Error Code
	ESME_RX_P_APPN        Status = 0x00000065 // ESME Receiver Permanent App Error Code
	ESME_RX_R_APPN        Status = 0x00000066 // ESME Receiver Reject Message Error Code
	ESME_RQUERYFAIL       Status = 0x00000067 // query_sm request failed
	ESME_RINVOPTPARSTREAM Status = 0x000000C0 // Error in the optional part of the PDU Body.
	ESME_ROPTPARNOTALLWD  Status = 0x000000C1 // Optional Parameter not allowed
	ESME_RINVPARLEN       Status = 0x000000C2 // Invalid Parameter Length.
	ESME_RMISSINGOPTPARAM Status = 0x000000C3 // Expected Optional Parameter missing
	ESME_RINVOPTPARAMVAL  Status = 0x000000C4 // Invalid Optional Parameter Value
	ESME_RDELIVERYFAILURE Status = 0x000000FE // Delivery Failure (data_sm_resp)
	ESME_RUNKNOWNERR      Status = 0x000000FF // Unknown Error
)

var esmeStatus = map[Status]string{
	ESME_ROK:           "OK",
	ESME_RINVMSGLEN:    "invalid message length",
	ESME_RINVCMDLEN:    "invalid command length",
	ESME_RINVCMDID:     "invalid command id",
	ESME_RINVBNDSTS:    "incorrect bind status for given command",
	ESME_RALYBND:       "already in bound state",
	ESME_RINVPRTFLG:    "invalid priority flag",
	ESME_RINVREGDLVFLG: "invalid registered delivery flag",
	ESME_RSYSERR:       "system error",
	ESME_RINVSRCADR:    "invalid source address",
	ESME_RINVDSTADR:    "invalid destination address",
	ESME_RINVMSGID:     "invalid message id",
	ESME_RBINDFAIL:     "bind failed",
	ESME_RINVPASWD:     "invalid password",
	ESME_RINVSYSID:     "invalid system id",
	ESME_RCANCELFAIL:   "cancelsm failed",
	ESME_RREPLACEFAIL:  "replacesm failed",
	ESME_RMSGQFUL:      "message queue full",
	ESME_RINVSERTYP:    "invalid service type",

	ESME_RINVNUMDESTS: "invalid number of destinations",
	ESME_RINVDLNAME:   "invalid distribution list name",

	ESME_RINVDESTFLAG:  "invalid destination flag",
	ESME_RINVSUBREP:    "invalid 'submit with replace' request",
	ESME_RINVESMSUBMIT: "invalid esm class field data",
	ESME_RCNTSUBDL:     "cannot submit to distribution list",
	ESME_RSUBMITFAIL:   "submitsm or submitmulti failed",
	ESME_RINVSRCTON:    "invalid source address ton",
	ESME_RINVSRCNPI:    "invalid source address npi",
	ESME_RINVDSTTON:    "invalid destination address ton",
	ESME_RINVDSTNPI:    "invalid destination address npi",
	ESME_RINVSYSTYP:    "invalid system type field",
	ESME_RINVREPFLAG:   "invalid replace_if_present flag",
	ESME_RINVNUMMSGS:   "invalid number of messages",
	ESME_RTHROTTLED:    "throttling error",

	ESME_RINVSCHED:    "invalid scheduled delivery time",
	ESME_RINVEXPIRY:   "invalid message validity period (expiry time)",
	ESME_RINVDFTMSGID: "predefined message invalid or not found",
	ESME_RX_T_APPN:    "esme receiver temporary app error code",
	ESME_RX_P_APPN:    "esme receiver permanent app error code",
	ESME_RX_R_APPN:    "esme receiver reject message error code",
	ESME_RQUERYFAIL:   "querysm request failed",

	ESME_RINVOPTPARSTREAM: "error in the optional part of the pdu body",
	ESME_ROPTPARNOTALLWD:  "optional parameter not allowed",
	ESME_RINVPARLEN:       "invalid parameter length",
	ESME_RMISSINGOPTPARAM: "expected optional parameter missing",
	ESME_RINVOPTPARAMVAL:  "invalid optional parameter value",
	ESME_RDELIVERYFAILURE: "delivery failure (used for datasmresp)",
	ESME_RUNKNOWNERR:      "unknown error",
}
