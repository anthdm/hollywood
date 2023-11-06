package remote

import (
	"context"
	"errors"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/log"
)

type streamReader struct {
	DRPCRemoteUnimplementedServer

	remote       *Remote
	deserializer Deserializer
	logger       log.Logger
}

func newStreamReader(r *Remote) *streamReader {
	return &streamReader{
		remote:       r,
		deserializer: ProtoSerializer{},
		logger:       r.logger.SubLogger("[stream_reader]"),
	}
}

func (r *streamReader) Receive(stream DRPCRemote_ReceiveStream) error {
	defer func() {
		r.logger.Debugw("terminated")
	}()

	for {
		envelope, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			r.logger.Errorw("receive", "err", err)
			return err
		}

		for _, msg := range envelope.Messages {
			tname := envelope.TypeNames[msg.TypeNameIndex]
			payload, err := r.deserializer.Deserialize(msg.Data, tname)
			if err != nil {
				r.logger.Errorw("deserialize", "err", err)
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
