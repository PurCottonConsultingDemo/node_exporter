package proc

import (
	"reflect"
	"testing"
)

func TestFiledesc_getSockets(t *testing.T) {
	type fields struct {
		Open    int64
		Targets []string
		Limit   uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   []uint64
	}{
		{
			name: "should",
			fields: fields{
				Targets: []string{
					"/dev/null",
					"/dev/pts/4 (deleted)",
					"socket:[9603969]",
					"socket:[9603971]",
					"socket:[21347918]",
					"/dev/pts/4 (deleted)",
					"/root/rm.run.log",
					"anon_inode:[eventpoll]",
					"pipe:[9603958]",
					"pipe:[9603958]",
					"socket:[21622363]",
				},
			},
			want: []uint64{
				9603969,
				9603971,
				21347918,
				21622363,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Filedesc{
				Open:    tt.fields.Open,
				Targets: tt.fields.Targets,
				Limit:   tt.fields.Limit,
			}
			if got := f.getSockets(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSockets() = %v, want %v", got, tt.want)
			}
		})
	}
}
