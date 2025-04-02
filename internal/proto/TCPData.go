package proto

func ReadTCPDataMessage(raw []byte) (uint32, []byte) {
	return GetCID(raw), raw[HeaderLength:]
}

func WriteTCPDataMessageHeader(cid uint32, dst []byte) {
	SetKind(dst, TCPConnectionData)
	SetCID(dst, cid)
}
