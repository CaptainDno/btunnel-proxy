package proto

func WriteTCPDialErrorMessage(cid uint32, error string, dst []byte) int {
	SetKind(dst, TCPDialError)
	SetCID(dst, cid)
	return copy(dst[HeaderLength:], error) + HeaderLength
}

func ReadTCPDialErrorMessage(src []byte) (uint32, string) {
	return GetCID(src), string(src[HeaderLength:])
}
