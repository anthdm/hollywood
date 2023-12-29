package remote

import (
	"context"
	"errors"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

// streamReader is a struct that implements the DRPCRemoteUnimplementedServer interface.
// It contains a reference to a Remote and a Deserializer.
type streamReader struct {
	DRPCRemoteUnimplementedServer

	remote       *Remote
	deserializer Deserializer
}

// newStreamReader is a function that creates a new instance of a streamReader.
// It takes a Remote as a parameter and returns a pointer to a streamReader.
// The returned streamReader has its remote field set to the provided Remote and its deserializer field set to a new ProtoSerializer.
func newStreamReader(r *Remote) *streamReader {
	return &streamReader{
		remote:       r,
		deserializer: ProtoSerializer{},
	}
}

// Receive is a method that receives messages from a stream.
// It takes a DRPCRemote_ReceiveStream as a parameter and returns an error.
// It continuously receives envelopes from the stream until an error occurs or the context is canceled.
// For each received envelope, it deserializes the messages and sends them to their respective targets.
func (r *streamReader) Receive(stream DRPCRemote_ReceiveStream) error {
	defer slog.Debug("streamreader terminated")

	// Continuously receive envelopes from the stream.
	for {
		envelope, err := stream.Recv()
		// If an error occurs, check if the context was canceled.
		// If the context was canceled, break the loop.
		// Otherwise, log an error message and return the error.
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}
			slog.Error("streamReader receive", "err", err)
			return err
		}

		// For each message in the envelope, deserialize the message and send it to its target.
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
			// Send the deserialized message to its target.
			r.remote.engine.SendLocal(target, payload, sender)
		}
	}

	return nil
}
