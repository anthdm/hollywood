package remote

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type streamReader struct {
	DRPCRemoteUnimplementedServer

	remote       *Remote
	deserializer Deserializer
}

func newStreamReader(r *Remote) *streamReader {
	return &streamReader{
		remote:       r,
		deserializer: ProtoSerializer{},
	}
}

func (r *streamReader) Receive(stream DRPCRemote_ReceiveStream) error {

	defer slog.Debug("streamreader terminated")

	for {
		envelope, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			slog.Error("streamReader receive", "err", err)
			return err
		}

		for _, msg := range envelope.Messages {
			tname := envelope.TypeNames[msg.TypeNameIndex]
			payload, err := r.deserializer.Deserialize(msg.Data, tname)
			if err != nil {
				slog.Error("streamReader deserialize", "err", err)
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
