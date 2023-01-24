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
		log.Warnw("[REMOTE] stream reader terminated", log.M{})
	}()

	for {
		msg, err := stream.Recv()
		if err != nil {
			if strings.Contains(err.Error(), "Canceled desc") {
				break
			}
			log.Errorw("[REMOTE] stream receive", log.M{"err": err})
			return err
		}

		pid := msg.Target
		dmsg, err := deserialize(msg.Data, msg.TypeName)
		if err != nil {
			log.Warnw("[REMOTE] deserialize", log.M{"err": err})
			continue
		}

		r.remote.engine.Send(pid, dmsg)
	}

	return nil
}
