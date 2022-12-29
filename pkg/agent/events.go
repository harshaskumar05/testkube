package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ common.ListenerLoader = &Agent{}

func (ag *Agent) Kind() string {
	return "agent"
}

func (ag *Agent) Load() (listeners common.Listeners, err error) {
	listeners = append(listeners, ag)

	return listeners, nil
}

func (ag *Agent) Name() string {
	return "agent"
}

func (ag *Agent) Selector() string {
	return ""
}

func (ag *Agent) Events() []testkube.EventType {
	return testkube.AllEventTypes
}
func (ag *Agent) Metadata() map[string]string {
	return map[string]string{
		"name":     ag.Name(),
		"selector": "",
		"events":   fmt.Sprintf("%v", ag.Events()),
	}
}

func (ag *Agent) Notify(event testkube.Event) (result testkube.EventResult) {
	// Non blocking send
	select {
	case ag.events <- event:
		return testkube.NewSuccessEventResult(event.Id, "message sent to websocket clients")
	default:
		return testkube.NewFailedEventResult(event.Id, errors.New("message not sent"))
	}
}

func (ag *Agent) runEventLoop(ctx context.Context) error {
	var opts []grpc.CallOption
	md := metadata.Pairs(apiKey, ag.apiKey)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := ag.client.Send(ctx, opts...)
	if err != nil {
		ag.logger.Errorf("failed to execute: %w", err)
		return errors.Wrap(err, "failed to setup stream")
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			msg := &cloud.WebsocketData{Opcode: cloud.Opcode_HEALTH_CHECK, Body: nil}

			err = ag.sendEvent(ctx, stream, msg)
			if err != nil {
				ag.logger.Errorf("websocket stream send healthcheck: %w", err)

				return err
			}

		case ev := <-ag.events:
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}

			msg := &cloud.WebsocketData{Opcode: cloud.Opcode_TEXT_FRAME, Body: b}
			err = ag.sendEvent(ctx, stream, msg)
			if err != nil {
				ag.logger.Errorf("websocket stream send: %w", err)
				return err
			}
		}
	}
}

func (ag *Agent) sendEvent(ctx context.Context, stream cloud.TestKubeCloudAPI_SendClient, event *cloud.WebsocketData) error {
	errChan := make(chan error, 1)
	go func() {
		errChan <- stream.Send(event)
		close(errChan)
	}()

	t := time.NewTimer(ag.sendTimeout)
	select {
	case err := <-errChan:
		if !t.Stop() {
			<-t.C
		}
		return err
	case <-ctx.Done():
		if !t.Stop() {
			<-t.C
		}

		return ctx.Err()
	case <-t.C:
		return errors.New("too slow")
	}
}
