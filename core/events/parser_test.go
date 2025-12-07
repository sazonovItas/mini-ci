package events

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type unknownEvent struct{}

func (unknownEvent) Type() EventType { return "unknown-event" }

func TestMessageParsing(t *testing.T) {
	type args struct {
		Event Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Error event",
			args: args{
				Event: Error{
					Origin:  Origin{ID: "86dd0fd2-ce19-4452-bf6f-c1102475eb18"},
					Time:    time.Now().UTC(),
					Message: "test errror event",
				},
			},
		},
		{
			name: "CreateContainer event",
			args: args{
				Event: StartInitContainer{
					Origin: Origin{ID: "86dd0fd2-ce19-4452-bf6f-c1102475eb18"},
					Time:   time.Now().UTC(),
					Config: ContainerConfig{
						Image: "test",
						Cwd:   "/test",
						Env:   []string{"TEST=test"},
					},
				},
			},
		},
		{
			name: "FinishCreateContainer event",
			args: args{
				Event: FinishInitContainer{
					Origin:      Origin{ID: "86dd0fd2-ce19-4452-bf6f-c1102475eb18"},
					Time:        time.Now().UTC(),
					ContainerID: "test",
				},
			},
		},
		{
			name: "StartScript event",
			args: args{
				Event: StartScript{
					Origin:      Origin{ID: "86dd0fd2-ce19-4452-bf6f-c1102475eb18"},
					Time:        time.Now().UTC(),
					ContainerID: "test",
					Config: ScriptConfig{
						Command: []string{"/usr/bin/ls"},
						Args:    []string{"-la"},
					},
				},
			},
		},
		{
			name: "FinishScript event",
			args: args{
				Event: FinishScript{
					Origin:     Origin{ID: "86dd0fd2-ce19-4452-bf6f-c1102475eb18"},
					Time:       time.Now().UTC(),
					ExitStatus: 0,
					Succeeded:  true,
				},
			},
		},
		{
			name: "Unknown event",
			args: args{
				Event: unknownEvent{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := Message{Event: tt.args.Event}

			payload, err := json.Marshal(msg)
			if err != nil {
				t.Errorf("json marshal message error = %v", err)
				return
			}

			var got Message
			err = json.Unmarshal(payload, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if !reflect.DeepEqual(msg, got) {
				t.Errorf("got = %v, want = %v", got.Event, msg.Event)
			}
		})
	}
}
