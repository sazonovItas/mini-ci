package events

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

type unknownEvent struct {
	EventOrigin `json:",inline"`
}

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
			name: "Task error event",
			args: args{
				Event: TaskError{
					EventOrigin: NewEventOrigin("86dd0fd2-ce19-4452-bf6f-c1102475eb18"),
					Message:     "test  task error event",
				},
			},
		},
		{
			name: "Log event",
			args: args{
				Event: TaskLog{
					EventOrigin: NewEventOrigin("86dd0fd2-ce19-4452-bf6f-c1102475eb18"),
					Messages:    []LogMessage{{Msg: "log", Time: time.Now().UTC()}},
				},
			},
		},
		{
			name: "ContainerInitStart event",
			args: args{
				Event: InitContainerStart{
					EventOrigin: NewEventOrigin("86dd0fd2-ce19-4452-bf6f-c1102475eb18"),
					Config: ContainerConfig{
						Image: "test",
						Cwd:   "/test",
						Env:   []string{"TEST=test"},
					},
				},
			},
		},
		{
			name: "ContainerInitFinish event",
			args: args{
				Event: InitContainerFinish{
					EventOrigin: NewEventOrigin("86dd0fd2-ce19-4452-bf6f-c1102475eb18"),
					ContainerID: "test",
				},
			},
		},
		{
			name: "ScriptStart event",
			args: args{
				Event: ScriptStart{
					EventOrigin: NewEventOrigin("86dd0fd2-ce19-4452-bf6f-c1102475eb18"),
					Config: ScriptConfig{
						ContainerID: "test",
						Command:     []string{"/usr/bin/ls"},
						Args:        []string{"-la"},
					},
				},
			},
		},
		{
			name: "FinishScript event",
			args: args{
				Event: ScriptFinish{
					EventOrigin: NewEventOrigin("86dd0fd2-ce19-4452-bf6f-c1102475eb18"),
					ExitStatus:  0,
					Succeeded:   true,
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
