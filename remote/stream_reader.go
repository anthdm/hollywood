package remote

import (
	"context"
	errors "errors"

	"github.com/anthdm/hollywood/actor"
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
		envelope, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			log.Errorw("[STREAM READER] receive", log.M{"err": err})
			return err
		}

		for _, msg := range envelope.Messages {
			payload, err := registryGetType(envelope.TypeNames[msg.TypeNameIndex])
			if err != nil {
				return err
			}
			if err := payload.UnmarshalVT(msg.Data); err != nil {
				return err
			}

			target := envelope.Targets[msg.TargetIndex]
			var sender *actor.PID
			if len(envelope.Senders) > 0 {
				sender = envelope.Senders[msg.SenderIndex]
			}

			r.remote.engine.SendLocal(target, payload, sender)
		}
	}

	return nil
}
