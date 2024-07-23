package batch

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/stretchr/testify/assert"
)

var dummyInstanceID = "i-11111111111111111"

func TestEntryToEventConverter(t *testing.T) {
	timeUnixMilli := time.UnixMilli(int64(1722650790111))
	now := func() time.Time {
		return timeUnixMilli
	}
	cursor := "cursor-0"
	converter := NewEntryToEventConverter(dummyInstanceID, now)
	entry, expectedEvent := getExampleEntryAndEvent(dummyInstanceID, timeUnixMilli, cursor)
	event := converter(entry)
	assert.JSONEq(t, *expectedEvent.Message, *event.Message)
	assert.Equal(t, *(expectedEvent.Timestamp), *(event.Timestamp))
}

// TestBatchOnMaxEvents tests batching entries into batch every maxEvents.
func TestBatchOnMaxEvents(t *testing.T) {
	timeUnixMilli := time.UnixMilli(int64(1722650790111))
	now := func() time.Time {
		return timeUnixMilli
	}
	converter := NewEntryToEventConverter(dummyInstanceID, now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entriesChan := make(chan *sdjournal.JournalEntry)
	go func() {
		for i := 0; i < 10; i++ {
			entry, _ := getExampleEntryAndEvent(dummyInstanceID, timeUnixMilli, fmt.Sprintf("cursor-%d", i))
			entriesChan <- entry
		}
	}()

	// Set MaxWait long enough to not interfer with MaxEvents.
	batcher := NewBatcher(entriesChan, converter, WithMaxEvents(2), WithMaxWait(time.Minute))
	go batcher.Batch(ctx)

	// Verify we get two batches.
	batch := <-batcher.Batches()
	assert.Equal(t, "cursor-1", batch.Cursor)
	batch = <-batcher.Batches()
	assert.Equal(t, "cursor-3", batch.Cursor)
}

func TestBatchOnMaxWait(t *testing.T) {
	timeUnixMilli := time.UnixMilli(int64(1722650790111))
	now := func() time.Time {
		return timeUnixMilli
	}
	converter := NewEntryToEventConverter(dummyInstanceID, now)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entriesChan := make(chan *sdjournal.JournalEntry)
	// Set MaxEvents big enough to not interfer with MaxWait.
	batcher := NewBatcher(entriesChan, converter, WithMaxEvents(1000), WithMaxWait(2*time.Second))
	go batcher.Batch(ctx)

	for i := 0; i < 10; i++ {
		entry, _ := getExampleEntryAndEvent(dummyInstanceID, timeUnixMilli, fmt.Sprintf("cursor-%d", i))
		entriesChan <- entry
	}
	batch := <-batcher.Batches()
	assert.Equal(t, "cursor-9", batch.Cursor)

	for i := 10; i < 30; i++ {
		entry, _ := getExampleEntryAndEvent(dummyInstanceID, timeUnixMilli, fmt.Sprintf("cursor-%d", i))
		entriesChan <- entry
	}
	batch = <-batcher.Batches()
	assert.Equal(t, "cursor-29", batch.Cursor)
}

func getExampleEntryAndEvent(instanceID string, timestamp time.Time, cursor string) (*sdjournal.JournalEntry, cloudwatchlogs.InputLogEvent) {
	entry := sdjournal.JournalEntry{
		Fields: map[string]string{
			"_PID":              "1",
			"_UID":              "2",
			"_GID":              "3",
			"_ERRNO":            "1",
			"_COMM":             "cowsay",
			"_SYSTEMD_UNIT":     "sshd",
			"PRIORITY":          "6",
			"SYSLOG_FACILITY":   "4",
			"SYSLOG_IDENTIFIER": "sshd",
			"SYSLOG_PID":        "1",
			"_BOOT_ID":          "f595e6391111111111111111372bf520",
			"_MACHINE_ID":       "ec22e31111111111111111111111115b",
			"_HOSTNAME":         "hello-server1.us-west-2.amazon.com",
			"_TRANSPORT":        "syslog",
			"MESSAGE":           "connection lost",
			"OTHER_KEY":         "OTHTER_VALUE",
		},
		Cursor:             cursor,
		RealtimeTimestamp:  1722650790111473,
		MonotonicTimestamp: 897993707018,
	}

	message := fmt.Sprintf(`
{
    "instanceId": "%s",
    "realTimestamp": 1722650790111473,
    "pid": 1,
    "uid": 2,
    "gid": 3,
    "cmdName": "cowsay",
    "systemdUnit": "sshd",
    "bootId": "f595e6391111111111111111372bf520",
    "machineId": "ec22e31111111111111111111111115b",
    "hostname": "hello-server1.us-west-2.amazon.com",
    "transport": "syslog",
    "priority": "info",
    "message": "connection lost",
    "syslog": {
        "facility": 4,
        "ident": "sshd",
        "pid": 1
    }
}
`, instanceID)

	event := cloudwatchlogs.InputLogEvent{
		Message:   aws.String(message),
		Timestamp: aws.Int64(timestamp.UnixMilli()),
	}
	return &entry, event
}
