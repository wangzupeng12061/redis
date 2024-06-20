package reply

type PongReply struct {
}

var pongBytes = []byte("+PONG\r\n")

func (p PongReply) ToBytes() []byte {
	return pongBytes
}
func MakePongReply() *PongReply {
	return &PongReply{}

}

type OkReply struct {
}

var OkBytes = []byte("+OK\r\n")

func (r OkReply) ToBytes() []byte {
	return OkBytes
}

var theOkReply = new(OkReply)

func MakeReply() *OkReply {
	return theOkReply
}

type NullBullReply struct {
}

var nullBulkBytes = []byte("$-1\r\n")

func (n NullBullReply) ToBytes() []byte {
	return nullBulkBytes
}
func MakeNullBulkReply() *NullBullReply {
	return &NullBullReply{}

}

type EmptyMultiBulkReply struct {
}

var emptyMultiBulkBytes = []byte("*0\r\n")

func (e EmptyMultiBulkReply) ToBytes() []byte {
	return emptyMultiBulkBytes
}

type NoReply struct {
}

var noBytes = []byte("")

func (n NoReply) ToBytes() []byte {
	return noBytes
}
