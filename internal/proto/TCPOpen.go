package proto

func WriteTCPOpenMessage(addr string, cid uint32, dst []byte) int {
	SetKind(dst, TCPConnectionOpen)
	SetCID(dst, cid)
	return copy(dst[HeaderLength:], addr) + HeaderLength
}

func ReadTCPOpenMessage(raw []byte) (uint32, string) {
	return GetCID(raw), string(raw[HeaderLength:])
}
