package publish

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	connection "github.com/jdattatr-tibco/messaging-contrib/pulsar/connection"
	"github.com/project-flogo/core/activity"
	"github.com/project-flogo/core/data"
	"github.com/project-flogo/core/data/coerce"
	"github.com/project-flogo/core/data/metadata"
	"github.com/project-flogo/core/engine"
	cnn "github.com/project-flogo/core/support/connection"
	"github.com/project-flogo/core/support/log"
	"github.com/project-flogo/core/support/trace"
)

func init() {
	_ = activity.Register(&Activity{}, New)
}

var activityMd = activity.ToMetadata(&Settings{}, &Input{}, &Output{})

//New optional factory method, should be used if one activity instance per configuration is desired
func New(ctx activity.InitContext) (activity.Activity, error) {

	s := &Settings{}
	err := metadata.MapToStruct(ctx.Settings(), s, true)
	if err != nil {
		return nil, err
	}

	pulsarConn, err := coerce.ToConnection(s.Connection)
	if err != nil {
		return nil, err
	}

	if ctx.Settings()["topic"] == nil {
		return nil, fmt.Errorf("no topic specified")
	}
	producerOptions := pulsar.ProducerOptions{
		Topic: ctx.Settings()["topic"].(string),
	}
	if ctx.Settings()["compressionType"] != nil {
		switch ctx.Settings()["compressionType"].(string) {
		case ("LZ4"):
			producerOptions.CompressionType = pulsar.LZ4
		case ("ZLIB"):
			producerOptions.CompressionType = pulsar.ZLib
		case ("ZSTD"):
			producerOptions.CompressionType = pulsar.ZSTD
		default:
			producerOptions.CompressionType = pulsar.NoCompression
		}
	}

	connMgr := pulsarConn.GetConnection().(connection.PulsarConnManager)

	act := &Activity{
		producerOpts: producerOptions,
		pulsarConn:   pulsarConn,
		connMgr:      connMgr,
	}
	return act, nil
}

// Activity is an sample Activity that can be used as a base to create a custom activity
type Activity struct {
	producer     pulsar.Producer
	producerOpts pulsar.ProducerOptions
	connMgr      connection.PulsarConnManager
	pulsarConn   cnn.Manager
}

// Metadata returns the activity's metadata
func (a *Activity) Metadata() *activity.Metadata {
	return activityMd
}

// Eval implements api.Activity.Eval - Logs the Message
func (a *Activity) Eval(ctx activity.Context) (done bool, err error) {
	a.connMgr = a.pulsarConn.GetConnection().(connection.PulsarConnManager)
	var logger log.Logger = ctx.Logger()

	if a.producer == nil {
		var hostName string
		hostName, err = os.Hostname()
		if err != nil {
			hostName = fmt.Sprintf("%d", time.Now().UnixMilli())
		}
		a.producerOpts.Name = fmt.Sprintf("%s-%s-%s-%s-%s", engine.GetAppName(), engine.GetAppVersion(), ctx.ActivityHost().Name(), ctx.Name(), hostName)
		a.producer, err = a.connMgr.GetProducer(a.producerOpts)
		if err != nil {

			return false, err
		}
	}

	input := &Input{}
	err = ctx.GetInputObject(input)
	if err != nil {
		return true, err
	}
	var msgBytes interface{}
	if input.Payload != nil {
		msgBytes, err = coerce.ToType(input.Payload, data.TypeBytes)
		if err != nil {
			return true, err
		}
	} else {
		msgBytes = make([]byte, 0)
	}

	msg := pulsar.ProducerMessage{
		Payload: msgBytes.([]byte),
	}
	if input.Properties != nil {
		props, err := coerce.ToType(input.Properties, data.TypeParams)
		if err != nil {
			return true, err
		}
		logger.Debugf("Publisher payload properties: %v", input.Properties)
		msg.Properties = props.(map[string]string)
	}
	if input.Key != "" {
		logger.Debugf("Publisher payload key: %s", input.Key)
		keyStr, err := coerce.ToType(input.Key, data.TypeString)
		if err != nil {
			return true, err
		}
		msg.Key = keyStr.(string)
	}
	if msg.Properties == nil {
		msg.Properties = make(map[string]string)
	}
	if trace.Enabled() {
		_ = trace.GetTracer().Inject(ctx.GetTracingContext(), trace.TextMap, msg.Properties)
	}
	msgID, err := a.producer.Send(context.Background(), &msg)
	if err != nil {
		return true, fmt.Errorf("Publisher could not send message: %v", err)
	}
	ctx.SetOutput("msgid", fmt.Sprintf("%x", msgID.Serialize()))
	return true, nil
}

func (a *Activity) Cleanup() error {
	if a.producer != nil {
		a.producer.Close()
	}
	return nil
}
