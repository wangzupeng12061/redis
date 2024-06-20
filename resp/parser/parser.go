package parser

import (
	"bufio"
	"errors"
	"io"
	"redis/interface/resp"
	"redis/lib/logger"
	"redis/resp/reply"
	"runtime/debug"
	"strconv"
	"strings"
)

type PayLoad struct {
	Data resp.Reply
	Err  error
}
type readState struct {
	readingMultiLine  bool
	expectedArgsCount int
	msgType           byte
	args              [][]byte
	bulkLen           int64
}

func (s *readState) finished() bool {
	return s.expectedArgsCount > 0 && len(s.args) == s.expectedArgsCount
}
func ParseStream(reader io.Reader) <-chan *PayLoad {
	ch := make(chan *PayLoad)
	go parse0(reader, ch)
	return ch
}
func parse0(reader io.Reader, ch chan<- *PayLoad) {
	defer func() {
		err := recover()
		if err != nil {
			logger.Error(string(debug.Stack()))
		}
	}()
	bufReader := bufio.NewReader(reader)
	var (
		state readState
		err   error
		msg   []byte
	)

	for {
		var ioErr bool
		msg, ioErr, err = readerLine(bufReader, &state)
		if err != nil {
			if ioErr {
				ch <- &PayLoad{Err: err}
				close(ch)
				return
			}
			ch <- &PayLoad{Err: err}
			state = readState{}
			continue
		}
		if !state.readingMultiLine {
			if msg[0] == '*' {
				err := parseMultiBulkHeader(msg, &state)
				if err != nil {
					ch <- &PayLoad{Err: errors.New("protocol error: " + string(msg))}
					state = readState{}
					continue
				}
				if state.expectedArgsCount == 0 {
					ch <- &PayLoad{Data: reply.EmptyMultiBulkReply{}}
					state = readState{}
				}

			} else if msg[0] == '$' {
				err := parseBulkHeader(msg, &state)
				if err != nil {
					ch <- &PayLoad{Err: errors.New("protocol error: " + string(msg))}
					state = readState{}
					continue
				}
				if state.bulkLen == -1 {
					ch <- &PayLoad{Data: reply.NullBullReply{}}
					state = readState{}
				}
			} else {
				result, err := parseSingleLineReply(msg)
				ch <- &PayLoad{Err: err, Data: result}
				state = readState{}
				continue
			}
		} else {
			err := readBody(msg, &state)
			if err != nil {
				ch <- &PayLoad{Err: errors.New("protocol error: " + string(msg))}
				state = readState{}
				continue
			}
			if state.finished() {
				var result resp.Reply
				if state.msgType == '*' {
					result = reply.MakeMultiBulkReply(state.args)
				} else if state.msgType == '$' {
					result = reply.MakeBulkReply(state.args[0])
				}
				ch <- &PayLoad{Err: err, Data: result}
				state = readState{}
			}
		}
	}
}

func parseBulkHeader(msg []byte, state *readState) error {
	var err error
	state.bulkLen, err = strconv.ParseInt(string(msg[1:len(msg)-1]), 10, 64)
	if err != nil {
		return errors.New("protocol error: " + string(msg))
	}
	if state.bulkLen == -1 {
		return nil
	} else if state.bulkLen > 0 {
		state.msgType = msg[0]
		state.readingMultiLine = true
		state.expectedArgsCount = 1
		state.args = make([][]byte, 0, 1)
		return nil
	} else {
		return errors.New("protocol error: " + string(msg))

	}
}
func readerLine(bufReader *bufio.Reader, state *readState) ([]byte, bool, error) {
	var msg []byte
	var err error
	if state.bulkLen == 0 {
		msg, err = bufReader.ReadBytes('\n')
		if err != nil {
			return nil, true, err
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' {
			return nil, false, errors.New("protocol error" + string(msg))

		}
	} else {
		msg = make([]byte, state.bulkLen+2)
		_, err := io.ReadFull(bufReader, msg)
		if err != nil {
			return nil, true, err
		}
		if len(msg) == 0 || msg[len(msg)-2] != '\r' || msg[len(msg)-1] != '\n' {
			return nil, false, errors.New("protocol error" + string(msg))

		}
		state.bulkLen = 0
	}
	return msg, false, nil
}
func parseMultiBulkHeader(msg []byte, state *readState) error {
	var err error
	var expectedLine uint64
	expectedLine, err = strconv.ParseUint(string(msg[1:len(msg)-2]), 10, 64)
	if err != nil {
		return errors.New("protocol error" + string(msg))

	}
	if expectedLine == 0 {
		state.expectedArgsCount = 0
		return nil
	} else if expectedLine > 0 {
		state.msgType = msg[0]
		state.readingMultiLine = true
		state.expectedArgsCount = int(expectedLine)
		state.args = make([][]byte, 0, expectedLine)
		return nil
	} else {
		return errors.New("protocol error" + string(msg))
	}
}
func parseSingleLineReply(msg []byte) (resp.Reply, error) {
	str := strings.TrimSuffix(string(msg), "\r\n")
	var result resp.Reply
	switch msg[0] {
	case '+':
		result = reply.MakeStatusReply(str[1:])
	case '-':
		result = reply.MakeErrReply(str[1:])
	case ':':
		val, err := strconv.ParseInt(str[1:], 10, 64)
		if err != nil {
			return nil, errors.New("protocol error" + string(msg))
		}
		result = reply.MakeIntReply(val)
	}
	return result, nil
}
func readBody(msg []byte, state *readState) error {
	line := msg[0 : len(msg)-2]

	var err error
	if msg[0] == '$' {
		state.bulkLen, err = strconv.ParseInt(string(line[1:]), 10, 64)
		if err != nil {
			return errors.New("protocol error" + string(msg))
		}
		if state.bulkLen <= 0 {
			state.args = append(state.args, []byte{})
			state.bulkLen = 0
		} else {
			state.args = append(state.args, line)
		}

	}
	return nil
}
