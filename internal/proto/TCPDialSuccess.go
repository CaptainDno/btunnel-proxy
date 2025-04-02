package proto

import "net"

func WriteTCPDialSuccessMessage(cid uint32, dst []byte, bndAddr net.Addr) int {
	SetKind(dst, TCPDialSuccess)
	SetCID(dst, cid)
	return copy(dst[HeaderLength:], bndAddr.String()) + HeaderLength
}

func ReadTCPDialSuccessMessage(src []byte) (uint32, net.Addr) {
	addr, err := net.ResolveTCPAddr("tcp", string(src[HeaderLength:]))
	if err != nil {
		panic(err)
	}
	return GetCID(src), addr
}
