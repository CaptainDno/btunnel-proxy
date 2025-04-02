package proto

func WriteTCPCloseMessage(cid uint32, dst []byte) int {
	SetKind(dst, TCPConnectionClose)
	SetCID(dst, cid)
	return HeaderLength
}

func ReadTCPCloseMessage(src []byte) uint32 {
	return GetCID(src)
}
