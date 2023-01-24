package remote

import (
	"strings"

	"github.com/anthdm/hollywood/log"
)

type streamReader struct {
	DRPCRemoteUnimplementedServer

	remote *Remote
}

func newStreamReader(r *Remote) *streamReader {
	return &streamReader{
		remote: r,
	}
}

func (r *streamReader) Receive(stream DRPCRemote_ReceiveStream) error {
	defer func() {
		log.Tracew("[STREAM READER] terminated", log.M{})
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				break
			}
			log.Errorw("[STREAM READER] receive", log.M{"err": err})
			return err
		}

		pid := msg.Target
		dmsg, err := deserialize(msg.Data, msg.TypeName)
		if err != nil {
			log.Warnw("[STREAM READER] deserialize", log.M{"err": err})
			continue
		}

		r.remote.engine.Send(pid, dmsg)
	}

	return nil
}
